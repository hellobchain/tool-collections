# 1. JSON格式返回
curl -X POST \
  -H "X-API-Key: your-secret-api-key-change-me" \
  -F "file=@报告.pdf" \
  http://localhost:8001/convert

# 2. 纯文本返回（直接存为.md）
curl -X POST \
  -H "X-API-Key: your-secret-api-key-change-me" \
  -F "file=@报告.pdf" \
  http://localhost:8001/convert-raw > 报告.md