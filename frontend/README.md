这是 **百宝箱** 的前端项目，技术栈为 **Vue 2 + Element UI + Vue Router + Vuex + Axios**。

### 项目概要

| 方面 | 说明 |
|---|---|
| **定位** | AI 辅助生成周报的 SPA |
| **路由** | `/login`（登录/注册）、`/`（周报工作台，需登录） |
| **后端** | Go 服务器，地址 `http://localhost:8000`，通过 proxy 转发 |
| **核心流程** | 收集碎片 → AI 生成草稿 → 手动润色 → 定稿归档 |
| **模块** | `auth`（登录态）、`weekly`（周报状态机）、`ui`（弹窗控制） |
| **特色功能** | 碎片管理、上期遗留确认、三种叙述风格（攻坚/协作/稳健）、AI 生成、定稿锁定 |

### 目录结构

```
src/
├── main.js          入口
├── App.vue          根组件
├── router/          路由 + 导航守卫
├── store/           Vuex 模块 (auth, weekly, ui)
├── api/             接口封装 (axios + 拦截器)
├── views/
│   ├── Login.vue    登录/注册页
│   └── WeeklyReport.vue  主工作台
└── components/
    └── CarryoverDialog.vue  遗留项确认弹窗
```

开发服务器 8080，运行 `npm run serve` 即可启动。


### 环境配置文件

| 文件 | 模式 | 用途 |
|---|---|---|
| `.env` | — | 默认（开发时 `npm run serve`） |
| `.env.development` | `development` | 开发环境 Docker 构建 |
| `.env.staging` | `staging` | 测试环境 |
| `.env.production` | `production` | 生产环境 |

### Docker 构建命令

```bash
# 开发环境
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 测试环境
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# 生产环境
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### 机制说明

- `docker-compose.yml` 作为基座，包含 postgres + backend，不包含 frontend
- 环境 compose 文件通过 `BUILD_ENV` build arg 控制前端构建模式
- Dockerfile 中 `npm run build -- --mode ${BUILD_ENV}` 会根据模式加载对应的 `.env.{mode}` 文件
- 生产环境暴露 `443` 端口（需自行配置 SSL），开发和测试暴露 `80`