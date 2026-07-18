# 设置 API Key
export API_KEY="your-secret-key"

# 1. 同步转换 - 上传文件
curl -X POST "http://localhost:8002/md2docx/v1/convert/file" \
  -H "Authorization: Bearer $API_KEY" \
  -F "file=@document.md" \
  -o output.docx

# 2. 同步转换 - 直接文本
curl -X POST "http://localhost:8002/md2docx/v1/convert/text" \
  -H "Authorization: Bearer $API_KEY" \
  -F "markdown=# Hello World" \
  -F "filename=hello.docx" \
  -o hello.docx

# 3. 异步转换（大文件）
curl -X POST "http://localhost:8002/md2docx/v1/convert/async" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "markdown": "# Title\n\nContent here...",
    "filename": "output.docx",
    "extra_args": ["--toc"]
  }'

# 返回: {"task_id": "xxx", "status_url": "/tasks/xxx"}

# 4. 查询任务状态
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8002/md2docx/v1/tasks/xxx"

# 5. 实时进度（SSE）
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8002/md2docx/v1/tasks/xxx/progress"

# 6. 下载结果
curl -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8002/md2docx/v1/download/xxx/output.docx" \
  -o result.docx

# 7. 批量转换
curl -X POST "http://localhost:8002/md2docx/v1/convert/batch" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"markdown": "# Doc 1", "filename": "doc1.docx"},
      {"markdown": "# Doc 2", "filename": "doc2.docx"}
    ]
  }'

# 8. 健康检查
curl "http://localhost:8002/md2docx/v1/health"


| 功能     | 端点                          | 说明            |
| ------ | --------------------------- | ------------- |
| 同步文件转换 | `POST /md2docx/v1/convert/file`        | 小文件即时返回       |
| 同步文本转换 | `POST /md2docx/v1/convert/text`        | 直接提交文本        |
| 异步转换   | `POST /md2docx/v1/convert/async`       | 大文件/后台处理      |
| 批量转换   | `POST /md2docx/v1/convert/batch`       | 多文件批量处理       |
| 任务状态   | `GET /md2docx/v1/tasks/{id}`           | 查询进度和状态       |
| 实时进度   | `GET /md2docx/v1/tasks/{id}/progress`  | SSE 流式推送      |
| 取消任务   | `DELETE /md2docx/v1/tasks/{id}`        | 取消未完成任务       |
| 下载结果   | `GET /md2docx/v1/download/{id}/{name}` | 获取转换结果        |
| 健康检查   | `GET /md2docx/v1/health`               | 服务状态监控        |
| API 认证 | `Authorization: Bearer`     | 全局 API Key 验证 |

pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
| 镜像      | 地址                                         |
| ------- | ------------------------------------------ |
| **清华**  | `https://pypi.tuna.tsinghua.edu.cn/simple` |
| **阿里云** | `https://mirrors.aliyun.com/pypi/simple`   |
| **豆瓣**  | `https://pypi.doubanio.com/simple`         |
| **中科大** | `https://pypi.mirrors.ustc.edu.cn/simple`  |

# 安装 uv 
curl -LsSf https://astral.sh/uv/install.sh | sh

# 使用 uv 安装（自动使用缓存，速度极快）
uv pip install -r requirements.txt

# 或创建虚拟环境并安装
uv venv
uv pip install -r requirements.txt

uv pip install --system -r requirements.txt

docker build -t md2docx-service:v1.0.0 .
