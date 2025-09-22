# Prompt Manager 后端规划

## 项目愿景
- 建立统一的 Prompt 资产库，支撑多租户团队共享与治理。
- 提供版本化、可审计的 Prompt 生命周期管理，便于回溯与演进。
- 暴露标准化 API，后续可扩展管理后台、统计大屏及多模型适配。

## 系统架构概览
- **架构模式**：Gin + Clean Architecture（三层：Handler / Service / Repository）。
- **运行组件**：HTTP API 服务、PostgreSQL（生产）/SQLite（开发）、Redis 缓存与统计计数、可选日志聚合与指标上报。
- **部署形态**：容器化（Docker Compose/Helm），12-Factor 配置，通过 Viper 加载多环境设定。

## 核心功能模块
- Prompt 模板管理：创建、查询、标签、变量 Schema 校验、版本化。
- Prompt 版本控制：草稿/发布版、差异对比、历史追踪。
- Prompt 执行与统计：变量渲染、执行日志、按 Prompt/版本/租户聚合统计。
- 多租户治理：租户隔离、角色与权限、审计日志。
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
8. 默认管理员：首次启动会自动创建默认租户与管理员账号，凭据为 `tenant_id=default-tenant`、`email=admin`、`password=admin123`（部署后建议立即修改密码或关闭 `bootstrap.enabled`）。

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
- **默认管理员**：可通过 `bootstrap.*` 配置或环境变量 `PROMPT_MANAGER_BOOTSTRAP_*` 控制默认租户/管理员的启用、邮箱与密码；上线后建议关闭或改为自定义值。

## 当前可用 API
- `GET /healthz`：返回服务状态、环境信息以及数据库/Redis 的健康详情。
- `POST /api/v1/auth/register`：注册租户内用户，要求 `tenant_id`、`email`、`password`、`role`（可选）。
- `POST /api/v1/auth/login`：使用 `tenant_id + email + password` 登录，返回访问令牌与刷新令牌。
- `POST /api/v1/auth/refresh`：提供刷新令牌换取新的访问/刷新令牌。
- 其余业务 API 将在后续里程碑逐步实现。

### 认证流程说明
1. **注册**：管理员调用 `/api/v1/auth/register` 创建用户，密码会以 bcrypt 哈希存储。
2. **登录**：用户凭凭证调用 `/api/v1/auth/login`，获得 `access_token` 与 `refresh_token`。
3. **访问受保护资源**：将 `Authorization: Bearer <access_token>` 加入请求头，`AuthGuard` 中间件会校验令牌并在上下文注入 `tenant_id`、`user_id`、`user_role`。
4. **刷新令牌**：在访问令牌即将过期时，调用 `/api/v1/auth/refresh` 补发新的令牌。
5. **统一错误格式**：所有认证相关接口返回 `code`、`message`、`details`，便于前端统一处理。

## 多租户设计
- **隔离策略**：共享 Schema，在所有业务表引入 `tenant_id`（联合唯一索引确保租户内唯一性）。
- **上下文传播**：认证中间件解析 Token/API Key，写入 Gin Context → Service 层通过 `TenantAwareRepository` 强制过滤。
- **RBAC**：租户维度角色（Admin/Editor/Viewer），权限矩阵随租户配置扩展；审计表记录关键操作。
- **未来扩展**：若需数据库级隔离，可新增按租户分库的策略，保持 Repository 抽象层的驱动切换能力。

## 认证策略（推荐方案）
1. **自托管用户名/密码 + JWT/Refresh Token**
   - 密码哈希使用 bcrypt/argon2，登录颁发 Access Token（短期）+ Refresh Token（长期）。
   - Token Payload：`sub`, `tenant_id`, `user_id`, `role`, `exp`；Refresh Token 存储于 Redis，绑定 `device_id`。
   - Gin 中间件校验签名并注入租户上下文，Service/RBAC 层二次校验权限。
2. **机器对机器访问（API Key/HMAC）**
   - 针对内部服务或第三方集成，签发带 `tenant_id` 的 API Key，调用方通过 `X-API-KEY` 或 HMAC 头访问。
   - 后端校验 Key 的启用状态与租户绑定，支持速率限制与即时吊销。
3. **未来演进：对接 OIDC**
   - 预留 `/.well-known/jwks.json`、Scopes 设计，后续可切换至 Auth0/Keycloak；保持 Handler 层 Token 抽象，便于更换 IdP。

## 缓存一致性策略
- **Prompt 数据缓存**
  - Key 约定：`tenant:{tenant_id}:prompt:{prompt_id}:v{version}`，内容含 Prompt 元数据 + 渲染模板。
  - 更新流程：Service 执行“先删缓存 → 写数据库 → 成功后写新缓存”；写库失败时恢复旧缓存，避免脏读。
  - 缓存穿透：对不存在的资源写入短 TTL 空值；缓存击穿使用 Redis 分布式锁（`SETNX`）。
  - 版本切换：缓存键包含版本号，新旧版本并存，旧键设置短 TTL，确保正在执行的请求自然过期。
- **统计数据缓存**
  - 执行日志实时写库（`prompt_execution_logs`）并按需写 Redis Counter (`INCRBY`)；定时任务批量刷新聚合表。
  - 查询接口优先读 Redis 聚合结果，未命中时回源数据库并回填缓存（TTL 3-5 分钟 + 抖动）。
  - 零点归档：每日定时任务将 Redis Counter 落库，保证历史数据完整；异常时可回放日志表重建。
- **监控与降级**
  - Prometheus 记录缓存命中率、锁等待、失败率；Redis 故障时回退到直连数据库并打告警。

## 配置与环境
- `config/default.yaml`：基础配置（端口、日志级别、JWT secret 占位）。
- `config/development.yaml`：SQLite DSN、Redis 本地实例、调试级别日志。
- `config/production.yaml`：PostgreSQL 连接、Redis 集群、日志采样、限流阈值。
- Viper 加载顺序：默认文件 → 环境特定文件 → 环境变量（`PROMPT_MANAGER_*`）。
- 支持 `WATCH_CONFIG` 开关，实现配置热加载（刷新 Redis TTL、日志级别等）。

## 开发计划与里程碑
1. **Milestone 1：项目骨架 & 核心模型**
   - 初始化 Go module、Gin 服务、配置加载、日志/错误响应规范。
   - 实现多租户中间件、用户模型、基础认证（登录/刷新/注销）。
   - 完成 Prompt/PromptVersion 仓储接口与 SQLite/PG 兼容迁移。
   - ✅ 已交付：依赖容器、数据库/Redis 健康检查、请求日志、租户注入、标准化响应、README 文档更新。
2. **Milestone 2：业务能力完善**
   - Prompt CRUD、版本管理、变量 Schema 校验、渲染接口（含缓存）。
   - 执行日志写入与统计聚合表、基础统计 API + Redis 缓存。
   - RBAC 权限校验、审计日志、API Key 管理。
3. **Milestone 3：稳定性与观测**
   - 缓存一致性策略实现（锁、抖动、空值缓存），指标上报与报警。
   - Docker Compose 本地环境、集成测试套件、性能基准测试。
   - 预留 OIDC 接入抽象、编写前端接口（Swagger/OpenAPI）。
4. **Milestone 4（规划中）**
   - 管理后台 UI、A/B 实验、Prompt 套件、模型集成、SLO/SLI 建设。

## 后续协作建议
- 确认团队对 IdP、日志采集、部署平台的偏好，提前规划基础设施。
- 根据租户规模评估是否需要分库分表或引入事件流；当前设计保持灵活。
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
