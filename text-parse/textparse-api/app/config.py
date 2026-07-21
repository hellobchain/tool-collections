import os
from pathlib import Path

# 服务配置
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "5001"))

# 认证配置
API_KEY = os.getenv("API_KEY", "your-secret-api-key-here")
API_KEY_HEADER = os.getenv("API_KEY_HEADER", "Authorization")
ENABLE_AUTH = os.getenv("ENABLE_AUTH", "true").lower() == "true"
