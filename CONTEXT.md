# Prompt Manager Context Snapshot

## 后端概览
- 技术栈：Go 1.22+（Gin、Clean Architecture），SQLite(Dev)/PostgreSQL(Prod)，Redis。
- 配置：Viper 读取 `config/default.yaml` + 环境覆写；`seed.admin` 支持文件或 `PROMPT_MANAGER_INIT_ADMIN_*` 种子管理员。
- 认证：JWT Access/Refresh Token，API：`/auth/login|refresh|register`。
- Prompt API：
  - `POST /api/v1/prompts` 创建模板（name/description/tags/body），若携带 body 自动创建激活版本并落库 `prompts.body`。
  - `GET /api/v1/prompts` 支持 `limit/offset/search`，返回 `items` 与 `meta`（total/hasMore），记录当前正文 `body`。
  - 版本相关：`POST /prompts/{id}/versions`、`POST /prompts/{id}/versions/{versionId}/activate` 等。
- 数据库：迁移位于 `db/migrations`; 最新 `000002_add_prompt_body` 添加 `prompts.body` 列。
- 中间件：请求日志、限流、AuthGuard。
- 工具：`make run`/`make test`，或 `go run cmd/server/main.go --config-dir=config --env=development`。

## 前端概览（web/）
- 技术栈：Vite + React + TypeScript、Tailwind CSS、TanStack Query、Axios、Zustand、React Hook Form + Zod。
- 认证：登录页调用 `/auth/login`，Axios 拦截器自动附带 Access Token 并在 401 时刷新 Refresh Token。
- Prompt 列表：搜索、分页、骨架/空态、错误提示，展示标签与正文摘要；数据来源 `listPrompts` API。
- Prompt 创建：`CreatePromptModal` 表单可一次性填写 name/description/tags/body，成功后刷新列表。
- 目录结构：`src/app`（路由/Provider）、`features`（auth、prompts）、`components/ui`（Button/Input/Textarea/Badge）。
- 常用脚本：`npm install`、`npm run dev`、`npm run lint`、`npm run build`（需 Node ≥20.19，当前 20.13.1 会警告）。

## 环境与运行
- 开发依赖：本地 Redis (`docker run --rm -p 6379:6379 redis:7-alpine`)，SQLite 数据库位于 `data/dev.db`。
- Docker：`docker compose up --build` 会自动执行迁移；单独迁移 `docker compose run --rm migrate up`。
- 健康检查：`GET /healthz` 返回服务、数据库、Redis 状态。
- 默认管理员：通过 `seed.admin` 或环境变量配置，首次启动后应及时更换。
