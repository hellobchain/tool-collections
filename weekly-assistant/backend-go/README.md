# 1. 进入后端目录
cd backend-go

# 2. 下载依赖
go mod download

# 3. 修改.env配置（数据库、DeepSeek Key等）

# 4. 创建PostgreSQL数据库
createdb weekly_assistant

# 5. 启动服务
go run cmd/main.go




## 项目概览：weekly-assistant 后端 (Go)

这是一个**周报AI助手**的 Go 后端项目，基于 **Gin + GORM + PostgreSQL** 构建。

### 目录结构

```
backend-go/
├── cmd/server/main.go        # 入口：路由注册、启动服务
├── internal/
│   ├── config/config.go      # 配置加载（环境变量/.env）
│   ├── database/database.go  # 数据库初始化 + 自动迁移
│   ├── models/models.go      # 数据模型 + DTO
│   ├── auth/jwt.go           # JWT 认证 & bcrypt 密码
│   ├── middleware/auth.go    # Gin 中间件（token 校验）
│   ├── handlers/
│   │   ├── auth.go           # 注册/登录
│   │   ├── week.go           # 周报状态/继承/生成/归档
│   │   └── fragment.go       # 碎片增删
│   └── services/
│       ├── week.go           # 周次工具函数
│       └── llm.go            # DeepSeek API 调用 + 降级方案
├── .env                      # 环境配置模板
├── go.mod / go.sum           # 依赖管理
└── run.sh / run.bat          # 启动脚本
```

### 数据模型 (4 张表)

| 表 | 用途 |
|---|---|
| `users` | 用户注册登录 |
| `fragments` | 周报碎片（手动添加/继承） |
| `weekly_reports` | 周报历史（定稿归档） |
| `goals` | 季度目标池 |

### API 路由

- **公开**: `POST /api/auth/register`, `POST /api/auth/login`
- **需认证**: 
  - `GET /api/week/status` — 本周状态（碎片、继承、定稿）
  - `POST /api/week/carryover/confirm` — 确认继承事项
  - `POST /api/week/generate` — 调用 LLM 生成草稿
  - `POST /api/week/finalize` — 定稿归档
  - `POST /api/fragments` / `DELETE /api/fragments/:id` — 碎片管理

### 核心流程

1. 用户添加 **碎片**（记录本周工作点滴）
2. 从上期周报自动提取 **继承事项**
3. 调用 **DeepSeek API** 将碎片整理为结构化周报草稿
4. 用户编辑后 **定稿归档**，同时提取下周待办项

### 技术栈

- **Gin** — HTTP 框架
- **GORM + PostgreSQL** — 数据库
- **golang-jwt** — JWT 认证
- **bcrypt** — 密码加密
- **DeepSeek API** — LLM 生成