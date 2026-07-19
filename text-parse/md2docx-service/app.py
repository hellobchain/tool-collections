import os
import shutil
import asyncio
from contextlib import asynccontextmanager
from typing import Optional

import uvicorn
from fastapi import FastAPI, File, Form, UploadFile, HTTPException, Depends, Request
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from pydantic import ValidationError

from config import UPLOAD_DIR, RESULTS_DIR, HOST, PORT, MAX_FILE_SIZE
from models import (
    ConvertRequest, BatchConvertRequest, TaskResponse,
    TaskInfo, HealthData, TaskStatus, ApiResponse, ErrorCode
)
from auth import verify_api_key
from tasks import task_manager


def ok(data=None, msg="success"):
    return ApiResponse(code=ErrorCode.SUCCESS, msg=msg, data=data)


def fail(code: int, msg: str, data=None):
    return ApiResponse(code=code, msg=msg, data=data)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await task_manager.start()
    yield
    await task_manager.stop()


app = FastAPI(
    title="Markdown to Word Converter",
    description="Production-ready HTTP API for converting Markdown to DOCX",
    version="2.0.0",
    lifespan=lifespan,
    default_response_class=JSONResponse,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ============ 统一异常处理器 ============

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


# ============ 同步转换端点 ============

@app.post("/convert/file")
async def convert_file(
    file: UploadFile = File(...),
    reference_doc: Optional[UploadFile] = File(None),
    _=Depends(verify_api_key)
):
    if not file.filename.endswith(('.md', '.markdown', '.txt')):
        return fail(ErrorCode.FILE_TYPE_NOT_ACCEPTED, "Only .md, .markdown, or .txt files accepted")

    content = await file.read()
    if len(content) > MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE, f"File too large, max {MAX_FILE_SIZE // 1024 // 1024}MB")

    md_text = content.decode("utf-8")

    ref_path = None
    if reference_doc:
        ref_path = UPLOAD_DIR / f"ref_{reference_doc.filename}"
        with open(ref_path, "wb") as f:
            f.write(await reference_doc.read())

    try:
        final_path = str(RESULTS_DIR / f"sync_{file.filename.rsplit('.', 1)[0]}.docx")
        await task_manager._convert(
            "sync", md_text, str(ref_path) if ref_path else None, None,
            output_path=final_path
        )
        return ok(data={"filename": f"{file.filename.rsplit('.', 1)[0]}.docx"})
    except Exception as e:
        return fail(ErrorCode.CONVERSION_FAILED, f"Conversion failed: {str(e)}")


@app.post("/convert/text")
async def convert_text(
    markdown: str = Form(...),
    filename: str = Form("document.docx"),
    reference_doc: Optional[str] = Form(None),
    _=Depends(verify_api_key)
):
    if len(markdown.encode('utf-8')) > MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE, f"Content too large, max {MAX_FILE_SIZE // 1024 // 1024}MB")

    try:
        final_path = str(RESULTS_DIR / f"sync_{filename}")
        await task_manager._convert("sync", markdown, reference_doc, None, output_path=final_path)
        return ok(data={"filename": filename})
    except Exception as e:
        return fail(ErrorCode.CONVERSION_FAILED, f"Conversion failed: {str(e)}")


# ============ 异步任务端点 ============

@app.post("/convert/async")
async def convert_async(request: ConvertRequest, _=Depends(verify_api_key)):
    task_id = await task_manager.create_task(
        markdown=request.markdown,
        filename=request.filename,
        reference_doc=request.reference_doc,
        extra_args=request.extra_args
    )
    return ok(data=TaskResponse(
        task_id=task_id,
        message="Task created successfully",
        status_url=f"/tasks/{task_id}"
    ))


@app.post("/convert/batch")
async def convert_batch(request: BatchConvertRequest, _=Depends(verify_api_key)):
    combined_md = "\n\n---\n\n".join([item.markdown for item in request.items])

    task_id = await task_manager.create_task(
        markdown=combined_md,
        filename="batch_document.docx",
        callback_url=request.callback_url
    )

    return ok(data=TaskResponse(
        task_id=task_id,
        message=f"Batch task created with {len(request.items)} items",
        status_url=f"/tasks/{task_id}"
    ))


# ============ 任务状态查询 ============

@app.get("/tasks/{task_id}")
async def get_task_status(task_id: str, _=Depends(verify_api_key)):
    task = await task_manager.get_task(task_id)
    if not task:
        return fail(ErrorCode.TASK_NOT_FOUND, "Task not found")
    return ok(data=task)


@app.get("/tasks/{task_id}/progress")
async def get_task_progress(task_id: str, _=Depends(verify_api_key)):
    async def event_generator():
        while True:
            task = await task_manager.get_task(task_id)
            if not task:
                yield f"data: {fail(ErrorCode.TASK_NOT_FOUND, 'Task not found').model_dump_json()}\n\n"
                break

            yield f"data: {ok(data={'task_id': task_id, 'progress': task.progress, 'status': task.status.value}).model_dump_json()}\n\n"

            if task.status in [TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED]:
                break

            await asyncio.sleep(1)

    from fastapi.responses import StreamingResponse
    return StreamingResponse(event_generator(), media_type="text/event-stream")


@app.delete("/tasks/{task_id}")
async def cancel_task(task_id: str, _=Depends(verify_api_key)):
    success = await task_manager.cancel_task(task_id)
    if not success:
        return fail(ErrorCode.TASK_CANNOT_CANCEL, "Task cannot be cancelled or not found")
    return ok(msg="Task cancelled")


# ============ 文件下载 ============

@app.get("/download/{task_id}/{filename}")
async def download_result(task_id: str, filename: str, _=Depends(verify_api_key)):
    file_path = RESULTS_DIR / f"{task_id}_{filename}"
    if not file_path.exists():
        return fail(ErrorCode.TASK_NOT_FOUND, "File not found")

    from fastapi.responses import FileResponse
    return FileResponse(
        file_path,
        filename=filename,
        media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    )


# ============ 健康检查 ============

@app.get("/health")
async def health_check():
    pandoc_ok = shutil.which("pandoc") is not None

    active = sum(1 for t in task_manager.tasks.values()
                 if t.status == TaskStatus.PROCESSING)

    return ok(data=HealthData(
        status="ok" if pandoc_ok else "error",
        pandoc_available=pandoc_ok,
        queue_size=len(task_manager.tasks),
        active_tasks=active,
        version="2.0.0"
    ))


@app.get("/")
async def root():
    return ok(data={
        "service": "Markdown to Word Converter",
        "version": "2.0.0",
        "endpoints": {
            "sync_file": "POST /convert/file",
            "sync_text": "POST /convert/text",
            "async": "POST /convert/async",
            "batch": "POST /convert/batch",
            "status": "GET /tasks/{task_id}",
            "progress": "GET /tasks/{task_id}/progress",
            "download": "GET /download/{task_id}/{filename}",
            "health": "GET /health"
        }
    })


if __name__ == "__main__":
    uvicorn.run(app, host=HOST, port=PORT)
