from pydantic import BaseModel, Field
from typing import Optional, TypeVar, Generic, List
from enum import Enum


# ---------- 业务错误码 ----------
class ErrorCode:
    SUCCESS = 0
    FILE_TOO_LARGE = 1001
    TASK_NOT_FOUND = 1002
    TASK_NOT_COMPLETED = 1003
    CONVERSION_FAILED = 1004
    NO_RESULT = 1005
    VALIDATION_ERROR = 1006
    UNKNOWN_ERROR = 9999
    NOT_FOUND = 404
    INVALID_API_KEY = 2001


T = TypeVar("T")


class ApiResponse(BaseModel, Generic[T]):
    code: int = Field(ErrorCode.SUCCESS, description="业务状态码，0 表示成功")
    msg: str = Field("success", description="结果描述信息")
    data: Optional[T] = None


# ---------- 业务模型 ----------

class OutputFormat(str, Enum):
    MD = "md"
    JSON = "json"
    HTML = "html"
    TEXT = "text"


class ConvertParams(BaseModel):
    to_formats: List[OutputFormat] = [OutputFormat.MD]
    do_ocr: bool = True
    table_mode: str = "fast"  # fast 或 accurate
    pdf_backend: str = "dlparse_v2"
