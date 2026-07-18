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


pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
| 镜像      | 地址                                         |
| ------- | ------------------------------------------ |
| **清华**  | `https://pypi.tuna.tsinghua.edu.cn/simple` |
| **阿里云** | `https://mirrors.aliyun.com/pypi/simple`   |
| **豆瓣**  | `https://pypi.doubanio.com/simple`         |
| **中科大** | `https://pypi.mirrors.ustc.edu.cn/simple`  |