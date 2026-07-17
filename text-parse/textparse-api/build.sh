# 1. 构建镜像
docker build -t docling-service:v1.0.0 .

# 2. 使用docker run运行
docker run -d \
  --name docling-service \
  -p 5001:5001 \
  -v ./tmp:/tmp/docling \
  -v ./models:/root/.cache/docling \
  docling-service:latest

# 3. 使用docker-compose运行（推荐）
docker-compose up -d

# 4. 查看日志
docker-compose logs -f

# 5. 停止服务
docker-compose down