from typing import List

from fastapi import FastAPI, UploadFile, File, HTTPException, Depends, Request, BackgroundTasks
from fastapi.responses import JSONResponse, PlainTextResponse, FileResponse
from fastapi.security import APIKeyHeader
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded
from markitdown import MarkItDown
from loguru import logger
from pydantic import ValidationError
import tempfile
import os
import asyncio
from datetime import datetime
from config import settings
from tasks import celery_app, convert_single_task, convert_batch_task
from celery.result import AsyncResult
from models import ApiResponse, ErrorCode


def ok(data=None, msg="success"):
    return ApiResponse(code=ErrorCode.SUCCESS, msg=msg, data=data)


def fail(code: int, msg: str, data=None):
    return ApiResponse(code=code, msg=msg, data=data)


app = FastAPI(
    title="MarkItDown API",
    version="1.0.0",
    default_response_class=JSONResponse,
)
limiter = Limiter(key_func=get_remote_address)
app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)


# ---------- 统一异常处理器 ----------

@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    return JSONResponse(
        status_code=200,
        content=fail(code=exc.status_code, msg=exc.detail).model_dump()
    )


@app.exception_handler(ValidationError)
async def validation_exception_handler(request: Request, exc: ValidationError):
    return JSONResponse(
        status_code=200,
        content=fail(code=ErrorCode.VALIDATION_ERROR, msg=str(exc)).model_dump()
    )


@app.exception_handler(Exception)
async def general_exception_handler(request: Request, exc: Exception):
    return JSONResponse(
        status_code=200,
        content=fail(code=ErrorCode.UNKNOWN_ERROR, msg=str(exc)).model_dump()
    )


# ---------- 鉴权 ----------
api_key_header = APIKeyHeader(name=settings.API_KEY_HEADER, auto_error=True)


async def verify_api_key(api_key: str = Depends(api_key_header)):
    if api_key != settings.API_KEY:
        logger.warning(f"非法API Key尝试")
        raise HTTPException(
            status_code=2001,
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
    return ok(data={"status": "ok", "timestamp": datetime.now().isoformat()})


# ---------- 核心转换接口 ----------

@app.post("/convert")
@limiter.limit(settings.RATE_LIMIT)
async def convert_file(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    ext = os.path.splitext(file.filename)[1].lower()
    if ext not in settings.ALLOWED_EXTENSIONS:
        return fail(ErrorCode.FILE_TYPE_NOT_ACCEPTED,
                    f"不支持 {ext} 格式，支持: {settings.ALLOWED_EXTENSIONS}")

    content = await file.read()
    file_size_mb = len(content) / (1024 * 1024)
    if file_size_mb > settings.MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE,
                    f"文件大小 {file_size_mb:.2f}MB 超过限制 {settings.MAX_FILE_SIZE}MB")

    async with semaphore:
        tmp_path = None
        try:
            with tempfile.NamedTemporaryFile(delete=False, suffix=ext) as tmp:
                tmp.write(content)
                tmp_path = tmp.name

            logger.info(f"开始转换: {file.filename}, 大小: {file_size_mb:.2f}MB")

            loop = asyncio.get_running_loop()
            result = await asyncio.wait_for(
                loop.run_in_executor(None, md.convert, tmp_path),
                timeout=settings.CONVERT_TIMEOUT
            )

            logger.success(f"转换成功: {file.filename}")

            return ok(data={
                "filename": file.filename,
                "markdown": result.text_content,
                "size_mb": round(file_size_mb, 2),
                "converted_at": datetime.now().isoformat()
            })

        except asyncio.TimeoutError:
            logger.error(f"转换超时: {file.filename}")
            return fail(ErrorCode.CONVERSION_TIMEOUT,
                        f"转换超时（{settings.CONVERT_TIMEOUT}秒）")
        except Exception as e:
            logger.exception(f"转换失败: {file.filename}, 错误: {str(e)}")
            return fail(ErrorCode.CONVERSION_FAILED, f"转换失败: {str(e)}")
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
    result = await convert_file(request, file, api_key)
    if result.code != 0:
        return PlainTextResponse(result.msg)
    return PlainTextResponse(result.data["markdown"])


# ---------- 转换并返回 Markdown 文件流 ----------

@app.post("/convertToMdFile")
async def convert_to_md_file(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    ext = os.path.splitext(file.filename)[1].lower()
    if ext not in settings.ALLOWED_EXTENSIONS:
        return fail(ErrorCode.FILE_TYPE_NOT_ACCEPTED,
                    f"不支持 {ext} 格式，支持: {settings.ALLOWED_EXTENSIONS}")

    content = await file.read()
    file_size_mb = len(content) / (1024 * 1024)
    if file_size_mb > settings.MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE,
                    f"文件大小 {file_size_mb:.2f}MB 超过限制 {settings.MAX_FILE_SIZE}MB")

    async with semaphore:
        tmp_input_path = None
        tmp_output_path = None

        def cleanup():
            if tmp_input_path and os.path.exists(tmp_input_path):
                os.unlink(tmp_input_path)
            if tmp_output_path and os.path.exists(tmp_output_path):
                os.unlink(tmp_output_path)

        try:
            with tempfile.NamedTemporaryFile(delete=False, suffix=ext) as tmp:
                tmp.write(content)
                tmp_input_path = tmp.name

            logger.info(f"开始转换: {file.filename}, 大小: {file_size_mb:.2f}MB")

            loop = asyncio.get_running_loop()
            result = await asyncio.wait_for(
                loop.run_in_executor(None, md.convert, tmp_input_path),
                timeout=settings.CONVERT_TIMEOUT
            )

            output_filename = f"{os.path.splitext(file.filename)[0]}.md"
            tmp_output_path = tempfile.mktemp(suffix=".md")

            with open(tmp_output_path, "w", encoding="utf-8") as f:
                f.write(result.text_content)

            logger.success(f"转换成功: {file.filename} -> {output_filename}")

            background_tasks = BackgroundTasks()
            background_tasks.add_task(cleanup)

            return FileResponse(
                path=tmp_output_path,
                filename=output_filename,
                media_type="text/markdown; charset=utf-8",
                background=background_tasks
            )

        except asyncio.TimeoutError:
            logger.error(f"转换超时: {file.filename}")
            cleanup()
            return fail(ErrorCode.CONVERSION_TIMEOUT,
                        f"转换超时（{settings.CONVERT_TIMEOUT}秒）")
        except Exception as e:
            logger.exception(f"转换失败: {file.filename}, 错误: {str(e)}")
            cleanup()
            return fail(ErrorCode.CONVERSION_FAILED, f"转换失败: {str(e)}")


# ---------- 异步单文件任务 ----------

@app.post("/async/convert")
async def async_convert(
    request: Request,
    file: UploadFile = File(...),
    api_key: str = Depends(verify_api_key)
):
    ext = os.path.splitext(file.filename)[1].lower()
    if ext not in settings.ALLOWED_EXTENSIONS:
        return fail(ErrorCode.FILE_TYPE_NOT_ACCEPTED, f"不支持 {ext} 格式")

    content = await file.read()
    file_size_mb = len(content) / (1024 * 1024)
    if file_size_mb > settings.MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE,
                    f"文件大小超过限制 {settings.MAX_FILE_SIZE}MB")

    task = convert_single_task.delay(content, file.filename, ext)

    logger.info(f"异步任务提交: task_id={task.id}, filename={file.filename}")

    return ok(data={
        "task_id": task.id,
        "status": "submitted",
        "filename": file.filename,
        "query_url": f"/async/status/{task.id}"
    })


# ---------- 批量异步任务 ----------

@app.post("/async/batch")
async def async_batch_convert(
    request: Request,
    files: List[UploadFile] = File(...),
    api_key: str = Depends(verify_api_key)
):
    if len(files) > settings.MAX_BATCH_SIZE:
        return fail(ErrorCode.VALIDATION_ERROR,
                    f"批量文件数量 {len(files)} 超过限制 {settings.MAX_BATCH_SIZE}")

    file_infos = []
    total_size = 0

    for file in files:
        ext = os.path.splitext(file.filename)[1].lower()
        if ext not in settings.ALLOWED_EXTENSIONS:
            return fail(ErrorCode.FILE_TYPE_NOT_ACCEPTED,
                        f"不支持的文件格式: {file.filename}")

        content = await file.read()
        file_size_mb = len(content) / (1024 * 1024)
        if file_size_mb > settings.MAX_FILE_SIZE:
            return fail(ErrorCode.FILE_TOO_LARGE,
                        f"文件 {file.filename} 大小超过限制 {settings.MAX_FILE_SIZE}MB")

        total_size += file_size_mb
        file_infos.append({
            "data": content,
            "filename": file.filename,
            "ext": ext
        })

    task = convert_batch_task.delay(file_infos)

    logger.info(f"批量任务提交: task_id={task.id}, 文件数={len(files)}, 总大小={total_size:.2f}MB")

    return ok(data={
        "task_id": task.id,
        "status": "submitted",
        "total_files": len(files),
        "total_size_mb": round(total_size, 2),
        "query_url": f"/async/status/{task.id}"
    })


# ---------- 查询任务状态 ----------

@app.get("/async/status/{task_id}")
async def get_task_status(
    task_id: str,
    api_key: str = Depends(verify_api_key)
):
    task_result = AsyncResult(task_id, app=celery_app)

    if not task_result:
        return fail(ErrorCode.TASK_NOT_FOUND, f"任务 {task_id} 不存在")

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

    if task_result.ready():
        if task_result.successful():
            result = task_result.result
            if "results" in result:
                response["result"] = result
            else:
                response["result"] = result

            expires_in = settings.RESULT_EXPIRE_SECONDS - (
                datetime.now().timestamp() - task_result.date_done.timestamp()
            ) if task_result.date_done else None
            response["expires_in_seconds"] = max(0, int(expires_in)) if expires_in else None
        else:
            response["error"] = str(task_result.info)
            response["result"] = None
    else:
        info = task_result.info or {}
        if task_result.status == "PROGRESS":
            response["progress"] = info.get("progress", 0)
            response["current"] = info.get("current", 0)
            response["total"] = info.get("total", 0)
            response["status_message"] = info.get("status", "处理中")
        else:
            response["progress"] = 0
            response["status_message"] = "等待执行"

    return ok(data=response)


# ---------- 取消任务 ----------

@app.delete("/async/cancel/{task_id}")
async def cancel_task(
    task_id: str,
    api_key: str = Depends(verify_api_key)
):
    task_result = AsyncResult(task_id, app=celery_app)

    if not task_result:
        return fail(ErrorCode.TASK_NOT_FOUND, f"任务 {task_id} 不存在")

    if task_result.ready():
        return ok(data={
            "task_id": task_id,
            "cancelled": False,
            "message": "任务已完成或已失败，无法取消"
        })

    celery_app.control.revoke(task_id, terminate=True)

    logger.info(f"任务已取消: task_id={task_id}")

    return ok(data={
        "task_id": task_id,
        "cancelled": True,
        "message": "任务已取消"
    })


# ---------- 批量任务列表 ----------

@app.get("/async/tasks")
async def list_recent_tasks(
    limit: int = 20,
    api_key: str = Depends(verify_api_key)
):
    try:
        import redis
        r = redis.from_url(settings.REDIS_URL)
        recent_tasks = r.lrange("task_history", 0, limit - 1)
        return ok(data={
            "count": len(recent_tasks),
            "task_ids": [t.decode() for t in recent_tasks],
        })
    except Exception as e:
        return ok(data={
            "count": 0,
            "task_ids": [],
        })


logger.info("异步接口已启用: /async/convert, /async/batch, /async/status/{task_id}")

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
