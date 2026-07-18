import os
import shutil
from contextlib import asynccontextmanager
from typing import Optional

import uvicorn
from fastapi import FastAPI, File, Form, UploadFile, HTTPException, Depends
from fastapi.responses import FileResponse
from fastapi.middleware.cors import CORSMiddleware

from config import UPLOAD_DIR, RESULTS_DIR, HOST, PORT, MAX_FILE_SIZE
from models import (
    ConvertRequest, BatchConvertRequest, TaskResponse, 
    TaskInfo, HealthResponse, TaskStatus
)
from auth import verify_api_key
from tasks import task_manager


@asynccontextmanager
async def lifespan(app: FastAPI):
    # 启动
    await task_manager.start()
    
    # 检查 pandoc
    try:
        result = shutil.which("pandoc")
        if not result:
            raise RuntimeError("Pandoc not found!")
    except Exception:
        pass
    
    yield
    
    # 关闭
    await task_manager.stop()


app = FastAPI(
    title="Markdown to Word Converter",
    description="Production-ready HTTP API for converting Markdown to DOCX",
    version="2.0.0",
    lifespan=lifespan
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ============ 同步转换端点（小文件） ============

@app.post("/md2docx/v1/convert/file", dependencies=[Depends(verify_api_key)])
async def convert_file(
    file: UploadFile = File(...),
    reference_doc: Optional[UploadFile] = File(None)
):
    """同步转换：上传 Markdown 文件，直接返回 Word 文件"""
    
    if not file.filename.endswith(('.md', '.markdown', '.txt')):
        raise HTTPException(400, "Only .md, .markdown, or .txt files accepted")
    
    content = await file.read()
    if len(content) > MAX_FILE_SIZE:
        raise HTTPException(413, f"File too large, max {MAX_FILE_SIZE//1024//1024}MB")
    
    md_text = content.decode("utf-8")
    
    # 处理模板
    ref_path = None
    if reference_doc:
        ref_path = UPLOAD_DIR / f"ref_{reference_doc.filename}"
        with open(ref_path, "wb") as f:
            f.write(await reference_doc.read())
    
    try:
        from tasks import task_manager
        final_path = str(RESULTS_DIR / f"sync_{file.filename.rsplit('.', 1)[0]}.docx")
        await task_manager._convert(
            "sync", md_text, str(ref_path) if ref_path else None, None,
            output_path=final_path
        )
        
        return FileResponse(
            final_path,
            filename=f"{file.filename.rsplit('.', 1)[0]}.docx",
            media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document"
        )
    except Exception as e:
        raise HTTPException(500, f"Conversion failed: {str(e)}")


@app.post("/md2docx/v1/convert/text", dependencies=[Depends(verify_api_key)])
async def convert_text(
    markdown: str = Form(...),
    filename: str = Form("document.docx"),
    reference_doc: Optional[str] = Form(None)
):
    """同步转换：直接提交 Markdown 文本"""
    
    if len(markdown.encode('utf-8')) > MAX_FILE_SIZE:
        raise HTTPException(413, f"Content too large, max {MAX_FILE_SIZE//1024//1024}MB")
    
    try:
        final_path = str(RESULTS_DIR / f"sync_{filename}")
        await task_manager._convert("sync", markdown, reference_doc, None, output_path=final_path)
        
        return FileResponse(
            final_path,
            filename=filename,
            media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document"
        )
    except Exception as e:
        raise HTTPException(500, f"Conversion failed: {str(e)}")


# ============ 异步任务端点（大文件/批量） ============

@app.post("/md2docx/v1/convert/async", response_model=TaskResponse, dependencies=[Depends(verify_api_key)])
async def convert_async(request: ConvertRequest):
    """异步转换：提交任务，返回任务 ID"""
    
    task_id = await task_manager.create_task(
        markdown=request.markdown,
        filename=request.filename,
        reference_doc=request.reference_doc,
        extra_args=request.extra_args
    )
    
    return TaskResponse(
        success=True,
        task_id=task_id,
        message="Task created successfully",
        status_url=f"/tasks/{task_id}"
    )


@app.post("/md2docx/v1/convert/batch", response_model=TaskResponse, dependencies=[Depends(verify_api_key)])
async def convert_batch(request: BatchConvertRequest):
    """批量异步转换"""
    
    # 创建批量任务（这里简化为单个任务处理多个文件）
    # 实际生产环境建议使用 Celery/RQ
    combined_md = "\n\n---\n\n".join([item.markdown for item in request.items])
    
    task_id = await task_manager.create_task(
        markdown=combined_md,
        filename="batch_document.docx",
        callback_url=request.callback_url
    )
    
    return TaskResponse(
        success=True,
        task_id=task_id,
        message=f"Batch task created with {len(request.items)} items",
        status_url=f"/tasks/{task_id}"
    )


# ============ 任务状态查询 ============

@app.get("/md2docx/v1/tasks/{task_id}", response_model=TaskInfo, dependencies=[Depends(verify_api_key)])
async def get_task_status(task_id: str):
    """查询任务状态和进度"""
    
    task = await task_manager.get_task(task_id)
    if not task:
        raise HTTPException(404, "Task not found")
    
    return task


@app.get("/md2docx/v1/tasks/{task_id}/progress", dependencies=[Depends(verify_api_key)])
async def get_task_progress(task_id: str):
    """获取任务进度（SSE 流式推送）"""
    
    async def event_generator():
        while True:
            task = await task_manager.get_task(task_id)
            if not task:
                yield f"data: {{'error': 'Task not found'}}\\n\\n"
                break
            
            yield f"data: {{'task_id': '{task_id}', 'progress': {task.progress}, 'status': '{task.status.value}'}}\\n\\n"
            
            if task.status in [TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED]:
                break
            
            await asyncio.sleep(1)
    
    from fastapi.responses import StreamingResponse
    import asyncio
    return StreamingResponse(event_generator(), media_type="text/event-stream")


@app.delete("/md2docx/v1/tasks/{task_id}", dependencies=[Depends(verify_api_key)])
async def cancel_task(task_id: str):
    """取消任务"""
    
    success = await task_manager.cancel_task(task_id)
    if not success:
        raise HTTPException(400, "Task cannot be cancelled or not found")
    
    return {"success": True, "message": "Task cancelled"}


# ============ 文件下载 ============

@app.get("/md2docx/v1/download/{task_id}/{filename}", dependencies=[Depends(verify_api_key)])
async def download_result(task_id: str, filename: str):
    """下载转换结果"""
    
    file_path = RESULTS_DIR / f"{task_id}_{filename}"
    if not file_path.exists():
        raise HTTPException(404, "File not found")
    
    return FileResponse(
        file_path,
        filename=filename,
        media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    )


# ============ 健康检查 ============

@app.get("/md2docx/v1/health", response_model=HealthResponse)
async def health_check():
    """健康检查"""
    
    import shutil
    pandoc_ok = shutil.which("pandoc") is not None
    
    active = sum(1 for t in task_manager.tasks.values() 
                 if t.status == TaskStatus.PROCESSING)
    
    return HealthResponse(
        status="ok" if pandoc_ok else "error",
        pandoc_available=pandoc_ok,
        queue_size=len(task_manager.tasks),
        active_tasks=active
    )


@app.get("/")
async def root():
    return {
        "service": "Markdown to Word Converter",
        "version": "2.0.0",
        "endpoints": {
            "sync_file": "POST /md2docx/v1/convert/file",
            "sync_text": "POST /md2docx/v1/convert/text", 
            "async": "POST /md2docx/v1/convert/async",
            "batch": "POST /md2docx/v1/convert/batch",
            "status": "GET /md2docx/v1/tasks/{task_id}",
            "progress": "GET /md2docx/v1/tasks/{task_id}/progress",
            "download": "GET /md2docx/v1/download/{task_id}/{filename}",
            "health": "GET /md2docx/v1/health"
        }
    }


if __name__ == "__main__":
    uvicorn.run(app, host=HOST, port=PORT)