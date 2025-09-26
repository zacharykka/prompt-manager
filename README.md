# Prompt Manager 后端规划

## 项目愿景
- 建立统一的 Prompt 资产库，支撑多用户团队共享与治理。
- 提供版本化、可审计的 Prompt 生命周期管理，便于回溯与演进。
- 暴露标准化 API，后续可扩展管理后台、统计大屏及多模型适配。

## 系统架构概览
- **架构模式**：Gin + Clean Architecture（三层：Handler / Service / Repository）。
- **运行组件**：HTTP API 服务、PostgreSQL（生产）/SQLite（开发）、Redis 缓存与统计计数、可选日志聚合与指标上报。
- **部署形态**：容器化（Docker Compose/Helm），12-Factor 配置，通过 Viper 加载多环境设定。

## 核心功能模块
- Prompt 模板管理：创建、查询、标签、变量 Schema 校验、版本化。
- Prompt 版本控制：草稿/发布版、差异对比、历史追踪。
- Prompt 执行与统计：变量渲染、执行日志、按 Prompt/版本聚合统计。
- 用户与权限：支持多用户登录，按角色限制操作，记录审计信息。
- 可扩展模块（后续）：Prompt 套件组合、A/B 实验、可视化后台、模型切换策略。

## 技术栈与基础设施
- **语言 & 框架**：Go 1.22+、Gin、Clean Architecture、依赖注入（wire 或 fx）。
- **配置管理**：Viper（`config/default.yaml` + 环境变量），支持热加载。
- **数据库**：开发环境 SQLite，生产环境 PostgreSQL（pgx / sqlc / gorm）；迁移工具 golang-migrate；当前实现采用 `modernc.org/sqlite` 与 `github.com/jackc/pgx/v5` 驱动。
- **缓存**：Redis（go-redis），承担热点 Prompt 缓存、统计聚合、分布式锁、速率限制。
- **认证 & 授权**：JWT、API Key/HMAC、RBAC；中长期支持 OIDC。
- **测试 & CI**：Go test、golangci-lint、Docker Compose 集成测试（Postgres + Redis）；后续接入 CI/CD。
- **可观测性**：Zap/zerolog 结构化日志、Prometheus 指标、OpenTelemetry Trace（规划中）。

## 快速开始
1. 安装 Go 1.22+，并确保本地具备 SQLite 与 Redis 运行环境（开发阶段可使用容器）。
2. 克隆项目并进入目录：
   ```bash
   git clone https://github.com/zacharykka/prompt-manager.git
   cd prompt-manager
   ```
3. 执行数据库迁移（首次运行或结构更新后）：
   - SQLite（开发环境）：
     ```bash
     migrate -path db/migrations -database "sqlite3://$(pwd)/data/dev.db" up
     ```
   - PostgreSQL（生产环境示例）：
     ```bash
     migrate -path db/migrations -database "postgres://USER:PASSWORD@HOST:5432/prompt_manager?sslmode=disable" up
     ```
   若未安装 `migrate` CLI，可参阅 https://github.com/golang-migrate/migrate/tree/master/cmd/migrate
4. 安装依赖：
   ```bash
   GOCACHE="$(pwd)/.cache/go-build" GOENV="$(pwd)/.config/go/env" go mod tidy
   ```
5. 启动开发服务：
   ```bash
   make run
   ```
   默认读取 `config/default.yaml` 与 `config/development.yaml`，监听 `0.0.0.0:8080`。
6. 健康检查：访问 `GET http://localhost:8080/healthz`，可查看服务/数据库/Redis 状态。
7. 运行测试：
   ```bash
   make test
   ```
8. 默认管理员：可在 `config/<env>.yaml` 的 `seed.admin` 节点中指定初始管理员邮箱/密码/角色，或使用兼容环境变量 `PROMPT_MANAGER_INIT_ADMIN_*`。若配置留空则不会自动创建账号；部署后务必及时更新或禁用默认管理员。
9. 安全配置：运行前务必设置以下环境变量（可参考 `.env.example`）：
   - `PROMPT_MANAGER_AUTH_ACCESS_TOKEN_SECRET` / `PROMPT_MANAGER_AUTH_REFRESH_TOKEN_SECRET`（≥32 字符）
   - `PROMPT_MANAGER_AUTH_API_KEY_HASH_SECRET`
   - 可选：`PROMPT_MANAGER_INIT_ADMIN_EMAIL`、`PROMPT_MANAGER_INIT_ADMIN_PASSWORD`、`PROMPT_MANAGER_INIT_ADMIN_ROLE`（会覆盖配置文件中的种子设置）
10. 请求体限制：可通过 `server.maxRequestBody` 设置单次请求体上限（默认 3MB），也可在环境变量 `PROMPT_MANAGER_SERVER_MAXREQUESTBODY` 中覆写。

## 使用 Docker 部署
1. 准备环境变量：
   ```bash
   cp .env.example .env
   ```
   根据需要修改访问令牌密钥、数据库口令等敏感信息。
2. 构建并启动服务（包含 PostgreSQL、Redis 与自动迁移）：
   ```bash
   docker compose up --build
   ```
   首次启动时 `migrate` 服务会自动运行数据库迁移并退出，随后 `app` 服务将依赖已完成的迁移继续启动。
3. 验证服务：
   - API: `http://localhost:8080/healthz`
   - PostgreSQL: `localhost:5432`
   - Redis: `localhost:6379`
4. 更新数据库结构时，可单独执行：
   ```bash
   docker compose run --rm migrate up
   ```
   若需回滚，可将 `up` 改为 `down 1` 等命令。
5. 停止并移除：
   ```bash
   docker compose down
   ```
   若需要清理数据卷，追加 `-v` 参数。

## 运行时依赖与初始化
- **数据库**：开发模式使用 `./data/dev.db`（自动创建）；生产模式需配置 PostgreSQL DSN，并调整连接池参数。
- **Redis**：用于缓存与健康检查，可通过 Docker 快速启动：
  ```bash
  docker run --rm -p 6379:6379 redis:7-alpine
  ```
- **配置文件**：可通过 `--config-dir` 指定目录，使用 `--env` 或环境变量 `PROMPT_MANAGER_ENV` 切换环境。
- **日志**：默认输出 JSON 到标准输出，级别由 `logging.level` 决定。
- **迁移执行**：推荐在 CI/CD 或启动脚本中调用 `migrate` CLI；也可将迁移步骤编排入 `Makefile`（例如新增 `make migrate`）。

## 当前可用 API
- `GET /healthz`：返回服务状态、环境信息以及数据库/Redis 的健康详情。
- `POST /api/v1/auth/register`：注册用户，字段 `email`、`password`、`role`（可选）。
- `POST /api/v1/auth/login`：使用 `email + password` 登录，返回访问令牌与刷新令牌。
- `POST /api/v1/auth/refresh`：提供刷新令牌换取新的访问/刷新令牌。
- `POST /api/v1/prompts`：创建 Prompt，可同时提交 `name`、`description`、`tags` 与初始 `body`，若提供正文会自动生成首个版本并设为已发布，同时将内容落入 `prompts.body` 字段。
- `GET /api/v1/prompts`：分页查询 Prompt 列表，支持 `limit`、`offset`、`search`（按名称模糊匹配），返回 `items` 与 `meta.total/limit/offset/hasMore`，并包含当前激活版本正文 `body` 便于前端展示概要。
- `GET /api/v1/prompts/{id}`：获取指定 Prompt 详情。
- `PUT /api/v1/prompts/{id}` / `PATCH /api/v1/prompts/{id}`：更新 Prompt 元数据。支持局部更新 `name`、`description`、`tags`；请求体必须至少包含一个字段，`name` 会自动 Trim 并验证非空，`tags` 接受 0~10 个字符串条目。
- `POST /api/v1/prompts/{id}/versions`：新增 Prompt 版本并可选设为激活。
- `GET /api/v1/prompts/{id}/versions`：查看 Prompt 版本列表。
- `POST /api/v1/prompts/{id}/versions/{versionId}/activate`：切换当前启用版本。
- `GET /api/v1/prompts/{id}/stats`：查看最近若干天（默认 7 天）的执行统计。
- `DELETE /api/v1/prompts/{id}`：软删除 Prompt（`status` 标记为 `deleted` 且记录 `deleted_at`），同时写入审计日志。操作完成后再次访问会返回 `404`。
- 其余业务 API 将在后续里程碑逐步实现。

### 认证流程说明
1. **注册**：管理员调用 `/api/v1/auth/register` 创建用户，密码以 bcrypt 哈希存储。
2. **登录**：用户凭凭证调用 `/api/v1/auth/login`，获得 `access_token` 与 `refresh_token`。
3. **访问受保护资源**：将 `Authorization: Bearer <access_token>` 加入请求头，`AuthGuard` 校验令牌并在上下文注入 `user_id`、`user_role`。
4. **权限**：写操作（创建/更新/删除 Prompt、激活版本）需要 `admin` 或 `editor` 角色，查看操作允许任意登录用户。删除操作会记录操作者信息至审计日志。
5. **刷新令牌**：在访问令牌即将过期时，调用 `/api/v1/auth/refresh` 补发新的令牌。
6. **统一错误格式**：所有认证相关接口返回 `code`、`message`、`details`，便于前端统一处理。

### GitHub OAuth 对接指南
> 目标：提供 GitHub 账号单点登录能力，降低团队成员首次接入成本。后端已内置完整的 OAuth 流程，可按如下步骤启用。

1. **创建 GitHub OAuth App**：访问 GitHub `Settings → Developer settings → OAuth Apps`，点击 `New OAuth App`。
   - 开发环境推荐配置：
     - `Homepage URL`: `http://localhost:8080`
     - `Authorization callback URL`: `http://localhost:8080/api/v1/auth/github/callback`
   - 生产环境需替换为公网域名并确保 HTTPS；可在 `config/production.yaml` 中调整回调地址。
2. **Scopes 选择**：首版仅需 `read:user` 与 `user:email`，用于读取基础资料与邮箱；如需团队/组织信息，可追加 `read:org`。
3. **后端流程约定**：
   - `GET /api/v1/auth/github/login`：根据配置生成 `state` 并 302 到 GitHub 授权页，可选携带 `redirect_uri` 作为登录完成后的回跳地址。
   - GitHub 成功授权后回调 `GET /api/v1/auth/github/callback?code=...&state=...`，后端会校验 `state`、交换 Access Token，并返回 JSON 结构 `{ "tokens": {...}, "user": {...}, "redirect_uri": "..." }`。
   - 若邮箱对应的本地账号不存在，会自动创建 `viewer` 角色的新用户并绑定 `user_identities` 映射；存在则完成绑定并更新最后登录时间。
4. **安全建议**：
   - `state` 由后端使用 JWT 签名并设置有效期，无需额外存储；若需更严的幂等控制，可结合 Redis 记录已消费的 state。
   - GitHub Access Token 仅用于一次性读取用户资料，不会持久化；如需调用更多 GitHub API，请结合后台作业或密钥管控策略。
   - 推荐在审计日志中记录第三方登录事件（含 `login`、`id`、`email`），并结合限流策略保护 `/github/login` 接口。

#### 环境变量配置
在 `.env` 或部署环境中新增以下变量，对应 `auth.github` 配置项：
```dotenv
PROMPT_MANAGER_AUTH_GITHUB_ENABLED=false
PROMPT_MANAGER_AUTH_GITHUB_CLIENTID="xxxx"
PROMPT_MANAGER_AUTH_GITHUB_CLIENTSECRET="xxxx"
PROMPT_MANAGER_AUTH_GITHUB_REDIRECTURL="http://localhost:8080/api/v1/auth/github/callback"
PROMPT_MANAGER_AUTH_GITHUB_SCOPES="read:user,user:email"
PROMPT_MANAGER_AUTH_GITHUB_ALLOWEDORGS="your-org-1,your-org-2"
PROMPT_MANAGER_AUTH_GITHUB_STATETTL=5m
```
- 需在生产环境通过安全渠道注入 `CLIENTSECRET` 等敏感配置。
- 若需要额外 Scopes，请在 `SCOPES` 中使用逗号分隔，并同步更新 GitHub 应用设置。

#### 调试与回归清单
- 本地模拟：`make run` 后访问 `http://localhost:8080/api/v1/auth/github/login`，确认 302 到 GitHub 授权页。
- 回调验证：使用测试账号完成授权后，应收到 JSON 响应并返回 `tokens`、`user` 信息；如传入 `redirect_uri` 会在响应中回显，供前端自行跳转。
- 失败场景：
  - `state` 无法解析或过期 → 返回 `400 OAUTH_STATE_INVALID`。
  - GitHub 未返回邮箱 → 返回 `400 OAUTH_EMAIL_MISSING`，需提示用户公开邮箱或在应用内补录。
  - 第三方组织校验失败 → 返回 `403 OAUTH_ORG_FORBIDDEN`。

## 认证策略（推荐方案）
1. **自托管用户名/密码 + JWT/Refresh Token**
   - 密码哈希使用 bcrypt/argon2，登录颁发 Access Token（短期）+ Refresh Token（长期）。
   - Token Payload：`sub`, `user_id`, `role`, `exp`；Refresh Token 可存储于 Redis 绑定 `device_id`。
   - Gin 中间件校验签名并注入用户上下文，Service/RBAC 层据此校验权限。
2. **机器对机器访问（API Key/HMAC）**
   - 针对内部服务或第三方集成，签发 API Key，调用方通过 `X-API-KEY` 或 HMAC 头访问。
   - 后端校验 Key 的启用状态并支持速率限制、即时吊销。
3. **未来演进：对接 OIDC**
   - 预留 `/.well-known/jwks.json`、Scopes 设计，后续可切换至 Auth0/Keycloak；保持 Handler 层 Token 抽象，便于更换 IdP。

## 缓存一致性策略
- **Prompt 数据缓存**
  - Key 约定：`prompt:{prompt_id}:v{version}`，内容含 Prompt 元数据 + 渲染模板。
  - 更新流程：Service 执行“先删缓存 → 写数据库 → 成功后写新缓存”；写库失败时恢复旧缓存，避免脏读。
  - 缓存穿透：对不存在的资源写入短 TTL 空值；缓存击穿使用 Redis 分布式锁（`SETNX`）。
  - 版本切换：缓存键包含版本号，新旧版本并存，旧键设置短 TTL，确保正在执行的请求自然过期。
- **统计数据缓存**
  - 执行日志实时写库（`prompt_execution_logs`）并按需写 Redis Counter (`INCRBY`)；定时任务批量刷新聚合表。
  - 查询接口优先读 Redis 聚合结果，未命中时回源数据库并回填缓存（TTL 3-5 分钟 + 抖动）。
  - 零点归档：每日定时任务将 Redis Counter 落库，保证历史数据完整；异常时可回放日志表重建。
- **监控与降级**
  - Prometheus 记录缓存命中率、锁等待、失败率；Redis 故障时回退到直连数据库并打告警。

## 审计与软删除
- **软删除语义**：`prompts` 表新增 `status`（默认 `active`）与 `deleted_at` 字段，通过更新状态实现软删除，后续可按需恢复或定期物理清理。
- **恢复能力**：
  - `POST /prompts/{id}/restore` 可将软删除的 Prompt 重新激活，同时清空 `deleted_at` 并回写最新 `updated_at`。
  - `GET /prompts?includeDeleted=true` 可列出处于 `deleted` 状态的记录，便于回收站场景；未显式传参时默认仅返回 `active` 数据。
  - 非删除状态调用恢复接口会返回 `400 PROMPT_NOT_DELETED`，已恢复或不存在的记录则返回 `404 PROMPT_NOT_FOUND`。
- **审计日志**：`prompt_audit_logs` 表记录关键动作（当前实现覆盖删除与恢复），字段包含操作者、动作类型与可选上下文 `payload`，便于合规追踪。
- **Service 行为**：后端删除与恢复逻辑均会写入审计日志，若未来扩展更多操作，可沿用相同仓储接口快速落地。

## 配置与环境
- `config/default.yaml`：基础配置（端口、日志级别、JWT secret 占位）。
- `config/development.yaml`：SQLite DSN、Redis 本地实例、调试级别日志。
- `config/production.yaml`：PostgreSQL 连接、Redis 集群、日志采样、限流阈值。
- `seed.admin`：可在各环境配置文件中写入初始管理员邮箱/密码/角色，留空则跳过；同名环境变量 `PROMPT_MANAGER_INIT_ADMIN_*` 可临时覆盖。
- Viper 加载顺序：默认文件 → 环境特定文件 → 环境变量（`PROMPT_MANAGER_*`）。
- 支持 `WATCH_CONFIG` 开关，实现配置热加载（刷新 Redis TTL、日志级别等）。

## 开发计划与里程碑
1. **Milestone 1：项目骨架 & 核心模型**
   - 初始化 Go module、Gin 服务、配置加载、日志/错误响应规范。
   - 完成用户模型与基础认证（注册/登录/刷新）。
   - 完成 Prompt/PromptVersion 仓储接口与 SQLite/PG 兼容迁移。
   - ✅ 已交付：依赖容器、数据库/Redis 健康检查、请求日志、标准化响应、README 文档更新。
2. **Milestone 2：业务能力完善**
   - Prompt CRUD、版本管理、变量 Schema 校验、渲染接口（含缓存）。
   - 执行日志写入与统计聚合表、基础统计 API + Redis 缓存。
   - RBAC 权限校验、审计日志、API Key 管理。
   - ✅ **Prompt 版本控制增强（阶段一）**：
     - 已提供 Diff 接口（支持上一版本/当前激活版本比对），产出结构化差异结果。
     - 已扩展审计日志：记录 `prompt.version.created`、`prompt.version.activated` 等关键动作。
     - ✅ 服务层补齐相关测试（包括旧数据兼容、审计日志落库）。
   - 🔜 **Prompt 版本控制增强（阶段二）**：
     - 优化版本恢复/并发校验、缓存刷新策略。
     - 根据前端反馈进一步扩展接口（例如版本评论、审批流）。

---

## 版本接口与 Diff 契约（最新）

为统一前后端契约，版本相关接口已全部采用 snake_case 字段；前端对应解析已同步更新。

- 列表版本：`GET /api/v1/prompts/:id/versions`
  - 响应 `items[]` 字段：
    - `id`, `version_number`, `status`, `created_by`, `created_at`, 以及可选 `variables_schema`、`metadata`（历史数据可能为空）。

- 创建版本：`POST /api/v1/prompts/:id/versions`
  - 请求：`body`（必填）、`status`（可选，默认 `published`）、`variables_schema`、`metadata`、`activate`（布尔）。
  - 审计：写入 `prompt.version.created`（payload 含 `version_id`、`version_number`、`status`、`activated_inline`）。

- 激活版本：`POST /api/v1/prompts/:id/versions/:versionId/activate`
  - 行为：更新 `prompts.active_version_id` 与 `prompts.body` 快照。
  - 审计：写入 `prompt.version.activated`（payload 含 `version_id`、`version_number`）。

- 版本 Diff：`GET /api/v1/prompts/:id/versions/:versionId/diff?compareTo=previous|active` 或 `?targetVersionId=xxx`
  - 响应示例（仅展示字段结构）：
    ```json
    {
      "diff": {
        "prompt_id": "p_123",
        "base":   {"id":"v2","version_number":2,"created_by":"user@x.com","created_at":"2025-09-24T21:42:00Z","status":"published"},
        "target": {"id":"v1","version_number":1,"created_by":null,"created_at":"2025-09-23T15:37:00Z","status":"published"},
        "body": [{"type":"insert|delete|equal","text":"..."}],
        "variables_schema": {"changes": [{"key":"foo","type":"modified","left":"bar","right":"baz"}]},
        "metadata": {"changes": [{"key":"k","type":"added","right":"1"}]}
      }
    }
    ```
  - 说明：
    - 文本差异为片段数组，`type` 取值 `insert|delete|equal`；前端根据类型高亮。
    - JSON 字段差异为键值级变化（`added|removed|modified`），值以字符串形式给出；后续可扩展更深层级 diff。

> 兼容性注意：早期接口的驼峰字段（例如 `versionNumber`、`variablesSchema`）已废弃，不再返回。

---

## 版本列表分页与状态筛选（后端）

为适配前端分页与筛选能力，版本列表接口支持如下参数与返回：

- 路由：`GET /api/v1/prompts/:id/versions`
- Query 参数：
  - `status`（可选）：`draft|published|archived`，省略表示全部。
  - `limit`（可选，默认 50）：单页条数。
  - `offset`（可选，默认 0）：偏移量。
- 响应：
  - `data.items`: 版本数组。
  - `data.meta`: `{ limit, offset, has_more }` 用于前端是否展示“下一页”。

实现要点：
- Service 新增 `ListPromptVersionsEx`，内部使用 `limit+1` 拉取并裁剪，计算 `has_more`。
- Repository 新增 `ListByPromptAndStatus`，在 SQL 层按 `status` 过滤。

3. **Milestone 3：稳定性与观测**
   - 缓存一致性策略实现（锁、抖动、空值缓存），指标上报与报警。
   - Docker Compose 本地环境、集成测试套件、性能基准测试。
   - 预留 OIDC 接入抽象、编写前端接口（Swagger/OpenAPI）。
4. **Milestone 4（规划中）**
   - 管理后台 UI、A/B 实验、Prompt 套件、模型集成、SLO/SLI 建设。

## 后续协作建议
- 确认团队对 IdP、日志采集、部署平台的偏好，提前规划基础设施。
- 如需横向扩展，可评估引入事件流或服务化抽象，目前设计保持灵活。
- 逐步编写 ADR（Architecture Decision Record），将关键决策沉淀，便于新成员快速上手。

## Git 提交与发布建议
- **分步提交**：按照功能模块拆分提交，例如“项目骨架初始化”“依赖容器与健康检查”“完善文档”。
- **示例流程**：
  1. `git add cmd/server internal pkg config README.md`
  2. `git commit -m "feat: setup service bootstrap"`
  3. `git commit -m "feat: add infra container and healthcheck"`
  4. `git commit -m "docs: update README"`
  5. `git push origin <branch>`
- **PR 建议**：在提交 PR 时列出变更范围、测试结果（如 `make test`）、配置改动及对 README 的更新，以便代码审查与部署。

---
以上规划将随着实现迭代持续更新，欢迎在 Issue 中记录讨论或补充需求。
