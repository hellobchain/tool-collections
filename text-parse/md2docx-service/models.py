from pydantic import BaseModel, Field
from typing import Optional, List, Literal, Any, Generic, TypeVar
from datetime import datetime
from enum import Enum


# ---------- 业务错误码 ----------
class ErrorCode:
    SUCCESS = 0
    FILE_TYPE_NOT_ACCEPTED = 1001
    FILE_TOO_LARGE = 1002
    TASK_NOT_FOUND = 1003
    TASK_CANNOT_CANCEL = 1004
    CONVERSION_FAILED = 1005
    PANDOC_NOT_FOUND = 1006
    VALIDATION_ERROR = 1007
    UNAUTHORIZED = 2001
    UNKNOWN_ERROR = 9999
    NOT_FOUND = 404


T = TypeVar("T")


class ApiResponse(BaseModel, Generic[T]):
    code: int = Field(ErrorCode.SUCCESS, description="业务状态码，0 表示成功")
    msg: str = Field("success", description="结果描述信息")
    data: Optional[T] = None


# ---------- 业务模型 ----------

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
    task_id: str
    message: str
    status_url: str


class HealthData(BaseModel):
    status: Literal["ok", "error"]
    pandoc_available: bool
    queue_size: int
    active_tasks: int
    version: str = "2.0.0"