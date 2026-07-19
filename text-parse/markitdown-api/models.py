from pydantic import BaseModel, Field
from typing import Optional, Any, Generic, TypeVar


class ErrorCode:
    SUCCESS = 0
    FILE_TYPE_NOT_ACCEPTED = 1001
    FILE_TOO_LARGE = 1002
    TASK_NOT_FOUND = 1003
    TASK_CANNOT_CANCEL = 1004
    CONVERSION_FAILED = 1005
    CONVERSION_TIMEOUT = 1006
    VALIDATION_ERROR = 1007
    UNAUTHORIZED = 2001
    RATE_LIMITED = 2002
    UNKNOWN_ERROR = 9999
    NOT_FOUND = 404
    INVALID_API_KEY = 2001


T = TypeVar("T")


class ApiResponse(BaseModel, Generic[T]):
    code: int = Field(ErrorCode.SUCCESS, description="业务状态码，0 表示成功")
    msg: str = Field("success", description="结果描述信息")
    data: Optional[T] = None
