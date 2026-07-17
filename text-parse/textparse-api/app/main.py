import io
import uuid
import zipfile
from pathlib import Path
from fastapi import FastAPI, UploadFile, File, Form, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse, Response
from fastapi.middleware.cors import CORSMiddleware

from app.converter import DoclingConverter
from app.utils import disposition_filename

# 初始化应用
app = FastAPI(
    title="Docling HTTP Service",
    description="文档转换服务 - 支持PDF、Word、图片等多种格式",
    version="1.0.0"
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

@app.get("/health")
async def health_check():
    """健康检查端点"""
    return {"status": "healthy", "service": "docling"}

@app.post("/v1/convert/file")
async def convert_file(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    to_formats: str = Form("md"),  # 逗号分隔的格式列表
    do_ocr: bool = Form(True),
    table_mode: str = Form("fast"),
    pdf_backend: str = Form("dlparse_v2")
):
    """
    同步转换文档
    """
    # 限制文件大小（100MB）
    MAX_FILE_SIZE = 100 * 1024 * 1024
    content = await file.read()
    if len(content) > MAX_FILE_SIZE:
        raise HTTPException(status_code=413, detail="File too large. Max size: 100MB")
    
    # 解析参数
    formats = [f.strip() for f in to_formats.split(',')]
    params = {
        'to_formats': formats,
        'do_ocr': do_ocr,
        'table_mode': table_mode,
        'pdf_backend': pdf_backend
    }
    
    try:
        # 执行转换
        result = converter.convert(content, file.filename, params)
        
        return JSONResponse(content={
            "status": "success",
            "filename": file.filename,
            "formats": formats,
            "document": result
        })
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Conversion failed: {str(e)}")

@app.post("/v1/convert/fileStream")
async def convert_file_stream(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    to_formats: str = Form("md"),  # 逗号分隔的格式列表
    do_ocr: bool = Form(True),
    table_mode: str = Form("fast"),
    pdf_backend: str = Form("dlparse_v2")
):
    """
    同步转换文档
    """
    # 限制文件大小（100MB）
    MAX_FILE_SIZE = 100 * 1024 * 1024
    content = await file.read()
    if len(content) > MAX_FILE_SIZE:
        raise HTTPException(status_code=413, detail="File too large. Max size: 100MB")
    
    # 解析参数
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
            # Single format → return directly as file
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

        # Multiple formats → zip
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
        raise HTTPException(status_code=500, detail=f"Conversion failed: {str(e)}")

@app.post("/v1/convert/file/async")
async def convert_file_async(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    to_formats: str = Form("md"),
    do_ocr: bool = Form(True),
    table_mode: str = Form("fast"),
    pdf_backend: str = Form("dlparse_v2")
):
    """
    异步转换文档（支持大文件）
    """
    # 读取文件
    content = await file.read()
    
    # 生成任务ID
    task_id = str(uuid.uuid4())
    
    # 保存文件到临时目录
    suffix = Path(file.filename).suffix
    temp_dir = Path("/tmp/docling")
    temp_dir.mkdir(exist_ok=True)
    temp_path = temp_dir / f"{task_id}{suffix}"
    temp_path.write_bytes(content)
    
    # 解析参数
    formats = [f.strip() for f in to_formats.split(',')]
    params = {
        'to_formats': formats,
        'do_ocr': do_ocr,
        'table_mode': table_mode,
        'pdf_backend': pdf_backend
    }
    
    # 存储任务信息
    task_store[task_id] = {
        "status": "pending",
        "filename": file.filename,
        "params": params,
        "temp_path": str(temp_path)
    }
    
    # 后台执行转换
    background_tasks.add_task(process_conversion, task_id, temp_path, params)
    
    return JSONResponse(content={
        "task_id": task_id,
        "status": "pending",
        "message": "Task submitted successfully"
    })

async def process_conversion(task_id: str, temp_path: Path, params: dict):
    """后台处理转换任务"""
    try:
        # 更新状态
        task_store[task_id]["status"] = "processing"
        
        # 读取文件内容
        content = temp_path.read_bytes()
        filename = task_store[task_id]["filename"]
        
        # 执行转换
        result = converter.convert(content, filename, params)
        
        # 更新结果
        task_store[task_id]["status"] = "completed"
        task_store[task_id]["result"] = result
        
    except Exception as e:
        task_store[task_id]["status"] = "failed"
        task_store[task_id]["error"] = str(e)
    finally:
        # 清理临时文件
        if temp_path.exists():
            temp_path.unlink()

@app.get("/v1/status/{task_id}")
async def get_task_status(task_id: str):
    """查询任务状态"""
    if task_id not in task_store:
        raise HTTPException(status_code=404, detail="Task not found")
    
    task = task_store[task_id]
    return JSONResponse(content={
        "task_id": task_id,
        "status": task["status"],
        "filename": task.get("filename"),
        "result": task.get("result") if task["status"] == "completed" else None,
        "error": task.get("error")
    })

@app.get("/v1/download/{task_id}")
async def download_task(task_id: str):
    """下载转换结果"""
    if task_id not in task_store:
        raise HTTPException(status_code=404, detail="Task not found")

    task = task_store[task_id]
    if task["status"] != "completed":
        raise HTTPException(status_code=400, detail="Task not yet completed")

    result = task["result"]
    formats = task["params"]["to_formats"]

    stem = Path(task["filename"]).stem

    if len(formats) == 1:
        fmt = formats[0]
        body = result.get(fmt)
        if body is None:
            raise HTTPException(status_code=500, detail="No result found")
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

    # Multiple formats → zip
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

@app.delete("/v1/task/{task_id}")
async def delete_task(task_id: str):
    """删除任务"""
    if task_id in task_store:
        del task_store[task_id]
        return {"status": "deleted", "task_id": task_id}
    raise HTTPException(status_code=404, detail="Task not found")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=5001)