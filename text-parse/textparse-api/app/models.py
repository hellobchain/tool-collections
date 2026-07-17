from pydantic import BaseModel
from typing import List
from enum import Enum

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