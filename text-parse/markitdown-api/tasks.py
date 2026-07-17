from celery import Celery
from markitdown import MarkItDown
from loguru import logger
import tempfile
import os
import asyncio
from config import settings
import time

# 初始化 Celery
celery_app = Celery(
    "markitdown_tasks",
    broker=settings.CELERY_BROKER_URL,
    backend=settings.CELERY_RESULT_BACKEND
)

celery_app.conf.update(
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="Asia/Shanghai",
    enable_utc=True,
    task_track_started=True,
    task_time_limit=settings.CONVERT_TIMEOUT + 30,
    task_soft_time_limit=settings.CONVERT_TIMEOUT,
    result_expires=settings.RESULT_EXPIRE_SECONDS,
)

# 初始化 MarkItDown（Worker 中单例）
_md = None

def get_markitdown():
    global _md
    if _md is None:
        if settings.ENABLE_LLM and settings.OPENAI_API_KEY:
            from openai import OpenAI
            client = OpenAI(api_key=settings.OPENAI_API_KEY)
            _md = MarkItDown(llm_client=client, llm_model=settings.OPENAI_MODEL)
        else:
            _md = MarkItDown()
    return _md

@celery_app.task(bind=True, name="tasks.convert_single")
def convert_single_task(self, file_data: bytes, filename: str, ext: str):
    """
    单个文件转换任务
    """
    task_id = self.request.id
    logger.info(f"[Task {task_id}] 开始转换: {filename}")
    
    tmp_path = None
    try:
        # 保存临时文件
        with tempfile.NamedTemporaryFile(delete=False, suffix=ext) as tmp:
            tmp.write(file_data)
            tmp_path = tmp.name
        
        # 更新进度
        self.update_state(state="PROGRESS", meta={"progress": 30, "status": "正在转换..."})
        
        # 执行转换
        md = get_markitdown()
        start_time = time.time()
        result = md.convert(tmp_path)
        elapsed = time.time() - start_time
        
        logger.info(f"[Task {task_id}] 转换成功: {filename}, 耗时: {elapsed:.2f}s")
        
        return {
            "filename": filename,
            "markdown": result.text_content,
            "size_mb": len(file_data) / (1024 * 1024),
            "elapsed_seconds": round(elapsed, 2),
            "status": "completed"
        }
        
    except Exception as e:
        logger.error(f"[Task {task_id}] 转换失败: {filename}, 错误: {str(e)}")
        return {
            "filename": filename,
            "error": str(e),
            "status": "failed"
        }
    finally:
        if tmp_path and os.path.exists(tmp_path):
            os.unlink(tmp_path)

@celery_app.task(bind=True, name="tasks.convert_batch")
def convert_batch_task(self, files: list):
    """
    批量文件转换任务
    files: [{"data": bytes, "filename": str, "ext": str}, ...]
    """
    task_id = self.request.id
    total = len(files)
    logger.info(f"[Task {task_id}] 开始批量转换, 共 {total} 个文件")
    
    results = []
    for idx, file_info in enumerate(files):
        # 更新进度
        progress = int((idx / total) * 100)
        self.update_state(
            state="PROGRESS", 
            meta={
                "progress": progress, 
                "current": idx + 1, 
                "total": total,
                "status": f"正在转换第 {idx+1}/{total} 个文件"
            }
        )
        
        # 调用单文件任务
        sub_result = convert_single_task(
            file_info["data"], 
            file_info["filename"], 
            file_info["ext"]
        )
        results.append(sub_result)
    
    # 统计结果
    success_count = sum(1 for r in results if r["status"] == "completed")
    failed_count = total - success_count
    
    logger.info(f"[Task {task_id}] 批量转换完成: 成功 {success_count}, 失败 {failed_count}")
    
    return {
        "task_id": task_id,
        "total": total,
        "success": success_count,
        "failed": failed_count,
        "results": results,
        "status": "completed"
    }