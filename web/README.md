# Prompt Manager Web 前端规划

## 项目愿景
- 构建轻量、响应迅速的后台管理界面，覆盖 Prompt 的增删改查、版本管理与统计能力。
- 与现有 Go 后端 API 无缝集成，提供安全、可扩展的前端基础设施。
- 通过模块化设计便于后续快速迭代，如权限扩展、API Key 管理、智能分析等。

## 业务范围概述
- **身份认证**：登录、令牌刷新、角色识别（Admin/Editor/Viewer）。
- **Prompt 列表**：支持名称搜索、分页、标签展示、当前正文概览，并提供快速进入编辑或创建流程；回收站视图可查看软删除记录并执行一键恢复。
- **Prompt 详情**：基础元信息、当前激活版本、关联统计与审计日志。
- **版本管理**：版本历史浏览、Diff 对比、激活/回滚、版本备注。
- **编辑流程**：创建/编辑 Prompt，支持草稿、字段校验、变量说明。
- **可选扩展**：使用数据统计、操作日志、API Key 管理面板。

## 技术选型（轻量化优先）
- **构建与语言**：Vite + React + TypeScript，享受快速热更新与 Tree-Shaking。
- **样式系统**：Tailwind CSS 搭配 Headless UI/Radix Primitives（按需引用，保持瘦身），或适度引入轻量组件库。
- **路由管理**：React Router v7 数据路由，支持懒加载和错误边界。
- **服务器状态**：TanStack Query 处理远程数据、缓存、乐观更新。
- **本地状态**：React Hooks 为主，必要时使用 Zustand 构建轻量全局 Store。
- **表单与校验**：React Hook Form + Zod，提升复杂表单的一致性与性能。
- **网络层**：Axios + 自定义拦截器（自动注入 Access Token、失败时串行刷新 Refresh Token 并重放请求）。
- **测试**：Vitest + React Testing Library，保证关键交互稳定性。
- **其他**：如需图表，优先采用 ECharts 单文件或 Recharts 按需加载版。

## 目录结构建议
```
web/
  src/
    app/                # App shell、全局路由、布局组件
    pages/              # 页面级组件（PromptList、PromptDetail 等）
    features/
      prompt/           # Prompt 领域模块（组件、hooks、api、types）
      auth/             # 鉴权模块
    components/         # 可复用基础组件与高级 UI 组合
    libs/               # axios/fetch 封装、queryClient、auth helpers
    stores/             # Zustand 等全局状态
    styles/             # Tailwind 配置、全局样式
    assets/             # 静态资源
  public/               # 静态文件、favicon
  index.html
  package.json
  vite.config.ts
```

## 状态与数据管理
- **鉴权流程**：启动阶段读取安全存储中的 Refresh Token；若存在则请求刷新接口，成功后再渲染受保护路由。
- **后端交互**：统一在 `features/*/api.ts` 中封装接口，与 TanStack Query 结合，实现缓存、重试、乐观更新。
- **错误处理**：请求失败时抛出带错误码的业务异常，由拦截器转成统一的 Toast 消息；401 自动触发刷新或退出登录流程。
- **权限控制**：在路由和组件层使用 `RoleGuard` 进行双重校验，按钮级别根据角色禁用或隐藏。
- **表单草稿**：编辑弹窗使用 Zustand/Context 保存临时输入，关闭时给出丢失警告。

## 页面与组件拆分
1. **Auth 模块**：登录页、忘记密码占位、鉴权路由守卫。
2. **Dashboard/PromptList**：
   - 搜索框、分页控制、空态提示、错误回退与骨架屏；
   - 列表项显示标签、正文摘要、最近更新人与时间；
   - 顶部按钮打开“新建 Prompt”弹窗（可一次性创建初始内容）。
3. **PromptDetail**：
   - 信息卡片展示基础属性。
   - Tab 切换 "版本历史"、"统计"、"审计" 等面板。
4. **PromptEditor / CreatePromptModal**：
   - 目前提供 Modal 形式的新建表单（名称、描述、标签、内容），后续可扩展变量说明与版本编辑能力；
   - 表单校验、请求错误提示、成功后自动刷新 Prompt 列表。
5. **VersionHistory**：
   - 列表 + Diff 视图，支持激活/回滚确认流程。
6. **统计/报告（可选）**：
   - 使用轻量图表展示调用量、成功率等指标。

## 实施路线图
1. **项目初始化**：创建 Vite + React + TS 工程，配置 ESLint/Prettier、Tailwind、路径别名、环境变量管理。
2. **基础设施**：搭建路由骨架、Layout、QueryClient、AuthProvider、错误边界与全局消息组件。
3. **认证模块**：完成登录页面、Token 存储与自动刷新逻辑，保护受限路由。
4. **Prompt 列表迭代**：实现列表视图、搜索过滤、分页、空态/加载态、正文预览。
5. **详情 + 版本管理**：完成详情页面、版本 Tab、Diff、激活/回滚交互。
6. **增删改流程**：构建 PromptEditor，落地创建、编辑、删除的 Mutation 流程；处理乐观更新与表单校验；回收站页支持软删除恢复与提示反馈。
7. **增强体验**：加入权限提示、字段帮助信息、批量操作、统计面板等增量功能。
8. **质量保障**：补齐关键交互单元测试、集成测试，配置 CI，接入 Bundle Analyzer 做性能体检。
9. **部署策略**：输出构建产物，通过 Vercel/Netlify/S3+CloudFront 等方案上线；结合后端环境变量配置跨域与鉴权。

## 后续思考
- 与后端团队确认 OpenAPI（或手工维护接口契约），必要时接入代码生成减少重复定义。
- 预留模块化扩展（例如 API Key、工作流审批），确保路由与状态不会耦合过度。
- 针对移动端或嵌入式场景，考虑 Tailwind 响应式设计与 PWA 支持。
- 监控与可观测性：接入前端日志上报/性能指标（Sentry、LogRocket 等轻量方案）。

该规划可作为后续工程实施、任务拆解与沟通的基础文档。

---

## 快速开始

```bash
# 安装依赖（推荐 pnpm，也可改用 npm install）
pnpm install

# 本地开发
pnpm run dev

# 构建产物
pnpm run build

# 预览构建
pnpm run preview
```

默认使用 Vite 提供的 React + TypeScript 模板，后续可根据规划引入 Tailwind CSS、TanStack Query 等依赖。
