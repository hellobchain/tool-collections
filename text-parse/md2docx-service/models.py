from pydantic import BaseModel, Field
from typing import Optional, List, Literal
from datetime import datetime
from enum import Enum


class TaskStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class ConvertRequest(BaseModel):
    markdown: str = Field(..., description="Markdown 文本内容", min_length=1)
    filename: str = Field("document.docx", description="输出文件名")
    reference_doc: Optional[str] = Field(None, description="参考模板文件路径")
    extra_args: List[str] = Field(default_factory=list, description="额外的 pandoc 参数")
    callback_url: Optional[str] = Field(None, description="完成后的回调 URL")


class BatchConvertRequest(BaseModel):
    items: List[ConvertRequest] = Field(..., min_length=1, max_length=50)
    callback_url: Optional[str] = Field(None, description="批量完成后的回调 URL")


class TaskInfo(BaseModel):
    task_id: str
    status: TaskStatus
    created_at: datetime
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    progress: int = Field(0, ge=0, le=100)
    message: str = ""
    output_url: Optional[str] = None
    error_message: Optional[str] = None


class TaskResponse(BaseModel):
    success: bool
    task_id: str
    message: str
    status_url: str


class HealthResponse(BaseModel):
    status: Literal["ok", "error"]
    pandoc_available: bool
    queue_size: int
    active_tasks: int
    version: str = "1.0.0"