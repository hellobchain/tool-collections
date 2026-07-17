from pydantic_settings import BaseSettings
from typing import Optional, List

class Settings(BaseSettings):
    # 服务配置
    API_HOST: str = "0.0.0.0"
    API_PORT: int = 8001
    
    # 安全配置
    API_KEY: str = "your-secret-api-key-here"
    API_KEY_HEADER: str = "X-API-Key"
    
    # 限流配置
    RATE_LIMIT: str = "10/minute"
    
    # 文件限制
    MAX_FILE_SIZE: int = 50  # MB
    MAX_BATCH_SIZE: int = 10  # 批量最大文件数
    ALLOWED_EXTENSIONS: List[str] = [".pdf", ".docx", ".xlsx", ".pptx", ".txt", ".md", ".jpg", ".png"]
    
    # LLM配置
    OPENAI_API_KEY: Optional[str] = None
    OPENAI_MODEL: str = "gpt-4o-mini"
    ENABLE_LLM: bool = False
    
    # 超时设置
    CONVERT_TIMEOUT: int = 120
    
    # Redis/Celery 配置
    REDIS_URL: str = "redis://redis:6379/0"
    CELERY_BROKER_URL: str = "redis://redis:6379/0"
    CELERY_RESULT_BACKEND: str = "redis://redis:6379/0"
    
    # 任务结果过期时间（秒）
    RESULT_EXPIRE_SECONDS: int = 3600
    
    class Config:
        env_file = ".env"

settings = Settings()