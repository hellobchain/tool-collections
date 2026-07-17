# 安装依赖
pip install -r requirements.txt

# 复制环境变量配置
cp .env.example .env
# 编辑.env修改API_KEY

# 启动
python app.py

## 一、Swagger 自动文档（已内置）

启动服务后访问：
- **Swagger UI**：`http://localhost:8001/docs`
- **ReDoc**：`http://localhost:8001/redoc`

FastAPI 会自动从代码生成交互式文档，可直接在页面上测试接口。

---

## 二、Markdown 接口文档

### 基础信息

| 项目 | 内容 |
|------|------|
| **服务名称** | MarkItDown API |
| **版本** | v1.0.0 |
| **基础URL** | `http://localhost:8001` |
| **认证方式** | API Key（Header） |
| **字符编码** | UTF-8 |
| **响应格式** | JSON / Plain Text |

---

### 认证说明

所有业务接口需要在请求头中携带 API Key：

```
X-API-Key: your-super-secret-key-change-me
```

---

### 1. 健康检查

用于服务存活检测和监控。

**请求**
```
GET /health
```

**响应示例**
```json
{
  "status": "ok",
  "timestamp": "2026-07-14T10:30:00.123456"
}
```

---

### 2. 转换文档（JSON格式）

上传文件，返回 JSON 格式的 Markdown 内容。

**请求**
```
POST /convert
Headers:
  X-API-Key: your-api-key
  Content-Type: multipart/form-data
Body:
  file: <文件二进制>
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | File | 是 | 待转换文件，支持 PDF/DOCX/XLSX/PPTX/TXT/MD/JPG/PNG |

**响应参数**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | integer | 状态码，200表示成功 |
| filename | string | 原始文件名 |
| markdown | string | 转换后的 Markdown 内容 |
| size_mb | float | 文件大小（MB） |
| converted_at | string | 转换时间（ISO 8601） |

**成功响应示例**
```json
{
  "code": 200,
  "filename": "技术方案.pdf",
  "markdown": "# 技术方案\\n\\n## 概述\\n\\n这是转换后的内容...",
  "size_mb": 2.35,
  "converted_at": "2026-07-14T10:30:15.123456"
}
```

**失败响应示例**
```json
{
  "detail": "不支持 .exe 格式，支持: ['.pdf', '.docx', '.xlsx', '.pptx', '.txt', '.md', '.jpg', '.png']"
}
```

**调用示例**
```bash
curl -X POST \
  -H "X-API-Key: your-super-secret-key-change-me" \
  -F "file=@技术方案.pdf" \
  http://localhost:8001/convert
```

---

### 3. 转换文档（纯文本返回）

上传文件，直接返回纯文本 Markdown 内容，便于保存为 `.md` 文件。

**请求**
```
POST /convert-raw
Headers:
  X-API-Key: your-api-key
  Content-Type: multipart/form-data
Body:
  file: <文件二进制>
```

**请求参数**（同 `/convert`）

**响应**
- Content-Type: `text/plain; charset=utf-8`
- Body: 纯 Markdown 文本

**调用示例**
```bash
# 直接保存为 .md 文件
curl -X POST \
  -H "X-API-Key: your-super-secret-key-change-me" \
  -F "file=@技术方案.pdf" \
  http://localhost:8001/convert-raw \
  -o 技术方案.md
```

---

## 三、错误码说明

| HTTP状态码 | 说明 | 处理建议 |
|-----------|------|---------|
| 200 | 成功 | - |
| 400 | 不支持的文件格式 | 检查文件扩展名是否在允许列表 |
| 403 | API Key 无效或缺失 | 检查请求头是否正确携带 `X-API-Key` |
| 408 | 转换超时 | 文件过大或内容复杂，可调整 `CONVERT_TIMEOUT` |
| 413 | 文件大小超限 | 压缩文件或调整 `MAX_FILE_SIZE` |
| 429 | 请求频率超限 | 降低请求频率，当前限制见 `RATE_LIMIT` |
| 500 | 服务器内部错误 | 查看服务日志排查问题 |

---

## 四、配置参数说明

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `API_KEY` | `your-secret-api-key-change-me` | API 认证密钥（生产环境务必修改） |
| `RATE_LIMIT` | `10/minute` | 限流策略，格式：`次数/时间单位` |
| `MAX_FILE_SIZE` | `50` | 最大文件大小（MB） |
| `CONVERT_TIMEOUT` | `120` | 转换超时时间（秒） |
| `ENABLE_LLM` | `false` | 是否启用 LLM 图片描述 |
| `OPENAI_API_KEY` | - | OpenAI API Key（启用 LLM 时必填） |
| `OPENAI_MODEL` | `gpt-4o-mini` | LLM 模型名称 |

---

## 五、Postman 导入

可以用以下 OpenAPI 地址导入 Postman：
```
http://localhost:8001/openapi.json
```

或者直接使用 Swagger UI 的 "Export" 功能导出。

---

## 六、Python 调用示例

```python
import requests

# 配置
url = "http://localhost:8001/convert"
headers = {"X-API-Key": "your-super-secret-key-change-me"}

# 上传文件
with open("报告.pdf", "rb") as f:
    files = {"file": ("报告.pdf", f, "application/pdf")}
    response = requests.post(url, headers=headers, files=files)

# 处理结果
if response.status_code == 200:
    data = response.json()
    markdown_content = data["markdown"]
    # 保存为 .md 文件
    with open("报告.md", "w", encoding="utf-8") as f:
        f.write(markdown_content)
    print(f"转换成功！文件大小：{data['size_mb']}MB")
else:
    print(f"失败：{response.json()}")
```

---

## 七、JavaScript/Node.js 调用示例

```javascript
const fs = require('fs');
const FormData = require('form-data');
const axios = require('axios');

const form = new FormData();
form.append('file', fs.createReadStream('报告.pdf'));

axios.post('http://localhost:8001/convert', form, {
  headers: {
    ...form.getHeaders(),
    'X-API-Key': 'your-super-secret-key-change-me'
  }
})
.then(res => {
  fs.writeFileSync('报告.md', res.data.markdown);
  console.log('转换成功！');
})
.catch(err => console.error('失败：', err.response?.data));
```

---