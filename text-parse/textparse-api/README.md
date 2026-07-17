# 同步转换
curl -X POST http://localhost:5001/v1/convert/file \
  -F "file=@document.pdf" \
  -F "to_formats=md,json" \
  -F "do_ocr=true"

# 异步转换（适合大文件）
curl -X POST http://localhost:5001/v1/convert/file/async \
  -F "file=@large_document.pdf" \
  -F "to_formats=md"

# 查询任务状态
curl http://localhost:5001/v1/status/{task_id}