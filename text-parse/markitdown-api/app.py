from typing import List

from fastapi import FastAPI, UploadFile, File, HTTPException, Depends, status, Request, BackgroundTasks
from fastapi.responses import  PlainTextResponse, FileResponse
from fastapi.security import APIKeyHeader
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded
from markitdown import MarkItDown
from loguru import logger
import tempfile
import os
import asyncio
from datetime import datetime
from config import settings
from tasks import celery_app, convert_single_task, convert_batch_task
from celery.result import AsyncResult

app = FastAPI(title="MarkItDown API", version="1.0.0")
limiter = Limiter(key_func=get_remote_address)
app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)

# ---------- 鉴权配置 ----------
api_key_header = APIKeyHeader(name=settings.API_KEY_HEADER, auto_error=True)

async def verify_api_key(api_key: str = Depends(api_key_header)):
    if api_key != settings.API_KEY:
        logger.warning(f"非法API Key尝试")
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无效的API Key"
        )
    return api_key
# ---------- 初始化MarkItDown ----------
def init_markitdown():
    if settings.ENABLE_LLM and settings.OPENAI_API_KEY:
        from openai import OpenAI
        client = OpenAI(api_key=settings.OPENAI_API_KEY)
        return MarkItDown(llm_client=client, llm_model=settings.OPENAI_MODEL)
    return MarkItDown()

md = init_markitdown()

# ---------- 并发控制 ----------
semaphore = asyncio.Semaphore(5)

# ---------- 健康检查 ----------
@app.get("/health")
async def health():
    return {"status": "ok", "timestamp": datetime.now().isoformat()}

# ---------- 核心转换接口 ----------
@app.post("/convert")
@limiter.limit(settings.RATE_LIMIT)
async def convert_file(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    # 1. 校验扩展名
    ext = os.path.splitext(file.filename)[1].lower()
    if ext not in settings.ALLOWED_EXTENSIONS:
        raise HTTPException(400, f"不支持 {ext} 格式，支持: {settings.ALLOWED_EXTENSIONS}")
    
    # 2. 校验文件大小
    content = await file.read()
    file_size_mb = len(content) / (1024 * 1024)
    if file_size_mb > settings.MAX_FILE_SIZE:
        raise HTTPException(413, f"文件大小 {file_size_mb:.2f}MB 超过限制 {settings.MAX_FILE_SIZE}MB")
    
    # 3. 并发控制 + 转换
    async with semaphore:
        tmp_path = None
        try:
            # 保存临时文件
            with tempfile.NamedTemporaryFile(delete=False, suffix=ext) as tmp:
                tmp.write(content)
                tmp_path = tmp.name
            
            logger.info(f"开始转换: {file.filename}, 大小: {file_size_mb:.2f}MB")
            
            # 设置超时
            loop = asyncio.get_running_loop()
            result = await asyncio.wait_for(
                loop.run_in_executor(None, md.convert, tmp_path),
                timeout=settings.CONVERT_TIMEOUT
            )
            
            logger.success(f"转换成功: {file.filename}")
            
            return {
                "code": 200,
                "filename": file.filename,
                "markdown": result.text_content,
                "size_mb": round(file_size_mb, 2),
                "converted_at": datetime.now().isoformat()
            }
            
        except asyncio.TimeoutError:
            logger.error(f"转换超时: {file.filename}")
            raise HTTPException(408, f"转换超时（{settings.CONVERT_TIMEOUT}秒）")
        except Exception as e:
            logger.exception(f"转换失败: {file.filename}, 错误: {str(e)}")
            raise HTTPException(500, f"转换失败: {str(e)}")
        finally:
            if tmp_path and os.path.exists(tmp_path):
                os.unlink(tmp_path)

# ---------- 纯文本返回 ----------
@app.post("/convert-raw")
@limiter.limit(settings.RATE_LIMIT)
async def convert_raw(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    """直接返回 .md 纯文本"""
    result = await convert_file(request, file, api_key)
    return PlainTextResponse(result["markdown"])


# ---------- 转换并返回 Markdown 文件流（带自动清理） ----------
@app.post("/convertToMdFile")
async def convert_to_md_file(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    """
    转换文档并直接返回 .md 文件下载
    响应完成后自动清理临时文件
    """
    # 1. 校验扩展名
    ext = os.path.splitext(file.filename)[1].lower()
    if ext not in settings.ALLOWED_EXTENSIONS:
        raise HTTPException(400, f"不支持 {ext} 格式，支持: {settings.ALLOWED_EXTENSIONS}")
    
    # 2. 校验文件大小
    content = await file.read()
    file_size_mb = len(content) / (1024 * 1024)
    if file_size_mb > settings.MAX_FILE_SIZE:
        raise HTTPException(413, f"文件大小 {file_size_mb:.2f}MB 超过限制 {settings.MAX_FILE_SIZE}MB")
    
    # 3. 并发控制 + 转换
    async with semaphore:
        tmp_input_path = None
        tmp_output_path = None
        
        def cleanup():
            """清理临时文件"""
            if tmp_input_path and os.path.exists(tmp_input_path):
                os.unlink(tmp_input_path)
            if tmp_output_path and os.path.exists(tmp_output_path):
                os.unlink(tmp_output_path)
        
        try:
            # 保存输入临时文件
            with tempfile.NamedTemporaryFile(delete=False, suffix=ext) as tmp:
                tmp.write(content)
                tmp_input_path = tmp.name
            
            logger.info(f"开始转换: {file.filename}, 大小: {file_size_mb:.2f}MB, IP: {request.client.host}")
            
            # 执行转换
            loop = asyncio.get_running_loop()
            result = await asyncio.wait_for(
                loop.run_in_executor(None, md.convert, tmp_input_path),
                timeout=settings.CONVERT_TIMEOUT
            )
            
            # 创建输出 Markdown 文件
            output_filename = f"{os.path.splitext(file.filename)[0]}.md"
            tmp_output_path = tempfile.mktemp(suffix=".md")
            
            with open(tmp_output_path, "w", encoding="utf-8") as f:
                f.write(result.text_content)
            
            logger.success(f"转换成功: {file.filename} -> {output_filename}")
            
            # 创建后台任务清理临时文件
            background_tasks = BackgroundTasks()
            background_tasks.add_task(cleanup)
            
            # 返回文件流
            return FileResponse(
                path=tmp_output_path,
                filename=output_filename,
                media_type="text/markdown; charset=utf-8",
                background=background_tasks
            )
            
        except asyncio.TimeoutError:
            logger.error(f"转换超时: {file.filename}")
            cleanup()
            raise HTTPException(408, f"转换超时（{settings.CONVERT_TIMEOUT}秒）")
        except Exception as e:
            logger.exception(f"转换失败: {file.filename}, 错误: {str(e)}")
            cleanup()
            raise HTTPException(500, f"转换失败: {str(e)}")


# ---------- 提交异步单文件任务 ----------
@app.post("/async/convert")
async def async_convert(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    """
    异步单文件转换
    提交任务后返回 task_id，通过 /async/status/{task_id} 查询结果
    """
    # 1. 校验扩展名
    ext = os.path.splitext(file.filename)[1].lower()
    if ext not in settings.ALLOWED_EXTENSIONS:
        raise HTTPException(400, f"不支持 {ext} 格式")
    
    # 2. 校验文件大小
    content = await file.read()
    file_size_mb = len(content) / (1024 * 1024)
    if file_size_mb > settings.MAX_FILE_SIZE:
        raise HTTPException(413, f"文件大小超过限制 {settings.MAX_FILE_SIZE}MB")
    
    # 3. 提交任务到 Celery
    task = convert_single_task.delay(content, file.filename, ext)
    
    logger.info(f"异步任务提交: task_id={task.id}, filename={file.filename}")
    
    return {
        "code": 200,
        "task_id": task.id,
        "status": "submitted",
        "filename": file.filename,
        "message": "任务已提交，请通过 /async/status/{task_id} 查询进度",
        "query_url": f"/async/status/{task.id}"
    }

# ---------- 提交批量异步任务 ----------
@app.post("/async/batch")
async def async_batch_convert(
    request: Request,
    files: List[UploadFile] = File(...),
    api_key: str = Depends(verify_api_key)
):
    """
    异步批量文件转换
    一次最多支持 {settings.MAX_BATCH_SIZE} 个文件
    """
    # 1. 校验文件数量
    if len(files) > settings.MAX_BATCH_SIZE:
        raise HTTPException(413, f"批量文件数量 {len(files)} 超过限制 {settings.MAX_BATCH_SIZE}")
    
    # 2. 校验每个文件
    file_infos = []
    total_size = 0
    
    for file in files:
        ext = os.path.splitext(file.filename)[1].lower()
        if ext not in settings.ALLOWED_EXTENSIONS:
            raise HTTPException(400, f"不支持的文件格式: {file.filename}")
        
        content = await file.read()
        file_size_mb = len(content) / (1024 * 1024)
        if file_size_mb > settings.MAX_FILE_SIZE:
            raise HTTPException(413, f"文件 {file.filename} 大小超过限制 {settings.MAX_FILE_SIZE}MB")
        
        total_size += file_size_mb
        file_infos.append({
            "data": content,
            "filename": file.filename,
            "ext": ext
        })
    
    # 3. 提交批量任务
    task = convert_batch_task.delay(file_infos)
    
    logger.info(f"批量任务提交: task_id={task.id}, 文件数={len(files)}, 总大小={total_size:.2f}MB")
    
    return {
        "code": 200,
        "task_id": task.id,
        "status": "submitted",
        "total_files": len(files),
        "total_size_mb": round(total_size, 2),
        "message": "批量任务已提交，请通过 /async/status/{task_id} 查询进度",
        "query_url": f"/async/status/{task.id}"
    }

# ---------- 查询任务状态 ----------
@app.get("/async/status/{task_id}")
async def get_task_status(
    task_id: str,
    api_key: str = Depends(verify_api_key)
):
    """
    查询异步任务状态和结果
    """
    task_result = AsyncResult(task_id, app=celery_app)
    
    # 任务不存在
    if not task_result:
        raise HTTPException(404, f"任务 {task_id} 不存在")
    
    # 任务状态
    status_map = {
        "PENDING": "等待中",
        "STARTED": "处理中",
        "PROGRESS": "处理中",
        "SUCCESS": "已完成",
        "FAILURE": "失败",
        "RETRY": "重试中"
    }
    
    response = {
        "task_id": task_id,
        "status": task_result.status,
        "status_text": status_map.get(task_result.status, "未知"),
        "submitted_at": task_result.date_done.isoformat() if task_result.date_done else None
    }
    
    # 如果任务完成，返回结果
    if task_result.ready():
        if task_result.successful():
            result = task_result.result
            # 判断是单文件还是批量结果
            if "results" in result:
                # 批量结果
                response["result"] = result
            else:
                # 单文件结果
                response["result"] = result
            
            # 添加结果过期时间
            expires_in = settings.RESULT_EXPIRE_SECONDS - (
                datetime.now().timestamp() - task_result.date_done.timestamp()
            ) if task_result.date_done else None
            response["expires_in_seconds"] = max(0, int(expires_in)) if expires_in else None
        else:
            response["error"] = str(task_result.info)
            response["result"] = None
    else:
        # 任务进行中，返回进度
        info = task_result.info or {}
        if task_result.status == "PROGRESS":
            response["progress"] = info.get("progress", 0)
            response["current"] = info.get("current", 0)
            response["total"] = info.get("total", 0)
            response["status_message"] = info.get("status", "处理中")
        else:
            response["progress"] = 0
            response["status_message"] = "等待执行"
    
    return response

# ---------- 取消任务 ----------
@app.delete("/async/cancel/{task_id}")
async def cancel_task(
    task_id: str,
    api_key: str = Depends(verify_api_key)
):
    """
    取消正在执行的任务
    """
    task_result = AsyncResult(task_id, app=celery_app)
    
    if not task_result:
        raise HTTPException(404, f"任务 {task_id} 不存在")
    
    if task_result.ready():
        return {
            "task_id": task_id,
            "cancelled": False,
            "message": "任务已完成或已失败，无法取消"
        }
    
    # 尝试撤销任务
    celery_app.control.revoke(task_id, terminate=True)
    
    logger.info(f"任务已取消: task_id={task_id}")
    
    return {
        "task_id": task_id,
        "cancelled": True,
        "message": "任务已取消"
    }

# ---------- 批量任务列表（最近N个） ----------
@app.get("/async/tasks")
async def list_recent_tasks(
    limit: int = 20,
    api_key: str = Depends(verify_api_key)
):
    """
    获取最近的任务列表（需要 Redis 支持）
    """
    # 简单实现：从 Redis 获取最近任务 ID
    # 更完善的方案可以用 Celery 的监控或单独记录
    try:
        import redis
        r = redis.from_url(settings.REDIS_URL)
        # 使用 Redis 记录任务历史（需要手动维护）
        recent_tasks = r.lrange("task_history", 0, limit - 1)
        return {
            "count": len(recent_tasks),
            "task_ids": [t.decode() for t in recent_tasks],
            "message": "仅显示最近任务ID，详细状态请查询 /async/status/{task_id}"
        }
    except Exception as e:
        return {
            "count": 0,
            "task_ids": [],
            "error": "无法获取任务列表，请检查 Redis 连接"
        }

# 启动时输出新增路由信息
logger.info("异步接口已启用: /async/convert, /async/batch, /async/status/{task_id}")

# ---------- 启动脚本 ----------
if __name__ == "__main__":
    import uvicorn
    logger.info(f"API服务启动: http://{settings.API_HOST}:{settings.API_PORT}")
    logger.info(f"限流策略: {settings.RATE_LIMIT}")
    logger.info(f"文件大小限制: {settings.MAX_FILE_SIZE}MB")
    uvicorn.run(
        app,
        host=settings.API_HOST,
        port=settings.API_PORT,
        log_level="info"
    )