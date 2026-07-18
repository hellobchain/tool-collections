import os
from pathlib import Path

# 基础路径
BASE_DIR = Path(__file__).parent
UPLOAD_DIR = BASE_DIR / "uploads"
UPLOAD_DIR.mkdir(exist_ok=True)
RESULTS_DIR = BASE_DIR / "results"
RESULTS_DIR.mkdir(exist_ok=True)

# 服务配置
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "8000"))
WORKERS = int(os.getenv("WORKERS", "4"))

# 认证配置
API_KEY = os.getenv("API_KEY", "your-secret-api-key-here")
ENABLE_AUTH = os.getenv("ENABLE_AUTH", "true").lower() == "true"

# 任务队列配置
MAX_CONCURRENT_TASKS = int(os.getenv("MAX_CONCURRENT_TASKS", "10"))
TASK_TIMEOUT = int(os.getenv("TASK_TIMEOUT", "300"))  # 秒
RESULT_RETENTION_HOURS = int(os.getenv("RESULT_RETENTION_HOURS", "24"))

# Pandoc 配置
PANDOC_TIMEOUT = int(os.getenv("PANDOC_TIMEOUT", "60"))
MAX_FILE_SIZE = int(os.getenv("MAX_FILE_SIZE", "50")) * 1024 * 1024  # 50MB

# 清理配置
CLEANUP_INTERVAL_MINUTES = int(os.getenv("CLEANUP_INTERVAL_MINUTES", "30"))