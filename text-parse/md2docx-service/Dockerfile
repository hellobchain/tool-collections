FROM pandoc/core:3.1-alpine

# 安装 Python
RUN apk add --no-cache python3 py3-pip

# 安装 uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

WORKDIR /app

# 创建虚拟环境（绕过 PEP 668 限制）
RUN uv venv

# 使用虚拟环境安装依赖
COPY requirements.txt .
RUN uv pip install -r requirements.txt \
    --index-url https://pypi.tuna.tsinghua.edu.cn/simple

COPY . .
RUN mkdir -p uploads results

EXPOSE 8002

# 使用虚拟环境中的 Python 运行
ENV PATH="/app/.venv/bin:$PATH"
# 关键：覆盖 ENTRYPOINT，不让 pandoc 当入口
ENTRYPOINT []
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8002"]