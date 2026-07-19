# ========== 构建阶段 ==========
FROM python:3.10-slim as builder

WORKDIR /build

# 换 pip 源（加速 Python 包下载）
RUN pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple

COPY requirements.txt .
RUN pip install --upgrade pip
RUN pip install --no-cache-dir -r requirements.txt


# ========== 运行阶段 ==========
FROM python:3.10-slim

WORKDIR /app

# 删除可能存在的旧配置，重新创建 sources.list
RUN rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*.sources 2>/dev/null || true && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free" > /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free" >> /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian-security bookworm-security main contrib non-free" >> /etc/apt/sources.list

RUN apt-get update && apt-get install -y \
    ffmpeg \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 从构建阶段复制已安装的 Python 包
COPY --from=builder /usr/local/lib/python3.10/site-packages /usr/local/lib/python3.10/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# 复制项目代码（放在最后，利用缓存，代码变动不影响前面层）
COPY . .

EXPOSE 8001

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8001"]