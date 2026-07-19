import io
import uuid
import zipfile
from pathlib import Path
from fastapi import FastAPI, UploadFile, File, Form, HTTPException, BackgroundTasks, Request
from fastapi.responses import JSONResponse, Response
from fastapi.middleware.cors import CORSMiddleware
from pydantic import ValidationError

from app.converter import DoclingConverter
from app.utils import disposition_filename
from app.models import ApiResponse, ErrorCode

# 初始化应用
app = FastAPI(
    title="Docling HTTP Service",
    description="文档转换服务 - 支持PDF、Word、图片等多种格式",
    version="1.0.0",
    default_response_class=JSONResponse,
)

# CORS配置
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 初始化转换器
converter = DoclingConverter()

# 简单的任务存储（生产环境建议使用Redis）
task_store = {}


def ok(data=None, msg="success"):
    return ApiResponse(code=ErrorCode.SUCCESS, msg=msg, data=data)


def fail(code: int, msg: str, data=None):
    return ApiResponse(code=code, msg=msg, data=data)


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

# api 路由找不到错误拦截
@app.exception_handler(404)
async def not_found_exception_handler(request: Request, exc):
    return JSONResponse(
        status_code=200,
        content=fail(code=ErrorCode.NOT_FOUND, msg="Endpoint not found").model_dump()
    )

# ---------- 端点 ----------

@app.get("/text-parse/v1/health")
async def health_check():
    return ok(data={"status": "healthy", "service": "docling"})


@app.post("/text-parse/v1/convert/file")
async def convert_file(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    to_formats: str = Form("md"),
    do_ocr: bool = Form(True),
    table_mode: str = Form("fast"),
    pdf_backend: str = Form("dlparse_v2")
):
    MAX_FILE_SIZE = 100 * 1024 * 1024
    content = await file.read()
    if len(content) > MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE, "File too large. Max size: 100MB")

    formats = [f.strip() for f in to_formats.split(',')]
    params = {
        'to_formats': formats,
        'do_ocr': do_ocr,
        'table_mode': table_mode,
        'pdf_backend': pdf_backend
    }

    try:
        result = converter.convert(content, file.filename, params)
        return ok(data={
            "filename": file.filename,
            "formats": formats,
            "document": result
        })
    except Exception as e:
        return fail(ErrorCode.CONVERSION_FAILED, f"Conversion failed: {str(e)}")


@app.post("/text-parse/v1/convert/fileStream")
async def convert_file_stream(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    to_formats: str = Form("md"),
    do_ocr: bool = Form(True),
    table_mode: str = Form("fast"),
    pdf_backend: str = Form("dlparse_v2")
):
    MAX_FILE_SIZE = 100 * 1024 * 1024
    content = await file.read()
    if len(content) > MAX_FILE_SIZE:
        return fail(ErrorCode.FILE_TOO_LARGE, "File too large. Max size: 100MB")

    formats = [f.strip() for f in to_formats.split(',')]
    params = {
        'to_formats': formats,
        'do_ocr': do_ocr,
        'table_mode': table_mode,
        'pdf_backend': pdf_backend
    }

    try:
        result = converter.convert(content, file.filename, params)

        if len(formats) == 1:
            fmt = formats[0]
            body = result[fmt]
            ext = fmt if fmt != 'text' else 'txt'
            media_types = {
                'md': 'text/markdown',
                'html': 'text/html',
                'json': 'application/json',
                'text': 'text/plain',
            }
            stem = Path(file.filename).stem
            return Response(
                content=body,
                media_type=media_types.get(fmt, 'application/octet-stream'),
                headers={"Content-Disposition": disposition_filename(f"{stem}.{ext}")}
            )

        stem = Path(file.filename).stem
        buf = io.BytesIO()
        with zipfile.ZipFile(buf, 'w', zipfile.ZIP_DEFLATED) as zf:
            for fmt in formats:
                body = result.get(fmt)
                if body is None:
                    continue
                ext = fmt if fmt != 'text' else 'txt'
                zf.writestr(f"{stem}.{ext}", body)
        buf.seek(0)
        return Response(
            content=buf.getvalue(),
            media_type='application/zip',
            headers={"Content-Disposition": disposition_filename(f"{stem}.zip")}
        )

    except Exception as e:
        return fail(ErrorCode.CONVERSION_FAILED, f"Conversion failed: {str(e)}")


@app.post("/text-parse/v1/convert/file/async")
async def convert_file_async(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    to_formats: str = Form("md"),
    do_ocr: bool = Form(True),
    table_mode: str = Form("fast"),
    pdf_backend: str = Form("dlparse_v2")
):
    content = await file.read()

    task_id = str(uuid.uuid4())

    suffix = Path(file.filename).suffix
    temp_dir = Path("/tmp/docling")
    temp_dir.mkdir(exist_ok=True)
    temp_path = temp_dir / f"{task_id}{suffix}"
    temp_path.write_bytes(content)

    formats = [f.strip() for f in to_formats.split(',')]
    params = {
        'to_formats': formats,
        'do_ocr': do_ocr,
        'table_mode': table_mode,
        'pdf_backend': pdf_backend
    }

    task_store[task_id] = {
        "status": "pending",
        "filename": file.filename,
        "params": params,
        "temp_path": str(temp_path)
    }

    background_tasks.add_task(process_conversion, task_id, temp_path, params)

    return ok(data={
        "task_id": task_id,
        "status": "pending",
        "message": "Task submitted successfully"
    })


async def process_conversion(task_id: str, temp_path: Path, params: dict):
    try:
        task_store[task_id]["status"] = "processing"

        content = temp_path.read_bytes()
        filename = task_store[task_id]["filename"]

        result = converter.convert(content, filename, params)

        task_store[task_id]["status"] = "completed"
        task_store[task_id]["result"] = result

    except Exception as e:
        task_store[task_id]["status"] = "failed"
        task_store[task_id]["error"] = str(e)
    finally:
        if temp_path.exists():
            temp_path.unlink()


@app.get("/text-parse/v1/status/{task_id}")
async def get_task_status(task_id: str):
    if task_id not in task_store:
        return fail(ErrorCode.TASK_NOT_FOUND, "Task not found")

    task = task_store[task_id]
    return ok(data={
        "task_id": task_id,
        "status": task["status"],
        "filename": task.get("filename"),
        "result": task.get("result") if task["status"] == "completed" else None,
        "error": task.get("error")
    })


@app.get("/text-parse/v1/download/{task_id}")
async def download_task(task_id: str):
    if task_id not in task_store:
        return fail(ErrorCode.TASK_NOT_FOUND, "Task not found")

    task = task_store[task_id]
    if task["status"] != "completed":
        return fail(ErrorCode.TASK_NOT_COMPLETED, "Task not yet completed")

    result = task["result"]
    formats = task["params"]["to_formats"]

    stem = Path(task["filename"]).stem

    if len(formats) == 1:
        fmt = formats[0]
        body = result.get(fmt)
        if body is None:
            return fail(ErrorCode.NO_RESULT, "No result found")
        ext = fmt if fmt != 'text' else 'txt'
        media_types = {
            'md': 'text/markdown',
            'html': 'text/html',
            'json': 'application/json',
            'text': 'text/plain',
        }
        return Response(
            content=body,
            media_type=media_types.get(fmt, 'application/octet-stream'),
            headers={"Content-Disposition": disposition_filename(f"{stem}.{ext}")}
        )

    buf = io.BytesIO()
    with zipfile.ZipFile(buf, 'w', zipfile.ZIP_DEFLATED) as zf:
        for fmt in formats:
            body = result.get(fmt)
            if body is None:
                continue
            ext = fmt if fmt != 'text' else 'txt'
            zf.writestr(f"{stem}.{ext}", body)
    buf.seek(0)
    return Response(
        content=buf.getvalue(),
        media_type='application/zip',
        headers={"Content-Disposition": disposition_filename(f"{stem}.zip")}
    )


@app.delete("/text-parse/v1/task/{task_id}")
async def delete_task(task_id: str):
    if task_id in task_store:
        del task_store[task_id]
        return ok(data={"task_id": task_id})
    return fail(ErrorCode.TASK_NOT_FOUND, "Task not found")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=5001)
