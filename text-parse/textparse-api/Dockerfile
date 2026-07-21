# ========== 构建阶段 ==========
FROM python:3.14-slim AS builder

WORKDIR /build

RUN pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple

COPY requirements.txt .
# RUN pip install --no-cache-dir -r requirements.txt
RUN pip install --no-cache-dir -r requirements.txt --extra-index-url https://download.pytorch.org/whl/cpu



# ========== 运行阶段 ==========
FROM python:3.14-slim

WORKDIR /app

# 删除可能存在的旧配置，重新创建 sources.list
RUN rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*.sources 2>/dev/null || true && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free" > /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free" >> /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian-security bookworm-security main contrib non-free" >> /etc/apt/sources.list

# 安装 libreoffice-writer
RUN apt-get update && apt-get install -y --no-install-recommends \
    libreoffice-writer \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

# 从构建阶段复制已安装的 Python 包
COPY --from=builder /usr/local/lib/python3.14/site-packages /usr/local/lib/python3.14/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

COPY . .

EXPOSE 5001

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "5001"]