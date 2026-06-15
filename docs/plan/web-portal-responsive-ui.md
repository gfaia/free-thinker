# Plan: 给 Web Portal 前端增加响应式 UI

## Context

用户希望给现有 web portal 前端“套一个 UI”，并且能适配桌面 Web 和移动端。当前 `web/portal` 已经是 Vue 3 + Vite + Vue Router + Element Plus 的只读数据浏览前端，但 UI 仍比较基础：`App.vue` 只有横向头部菜单，`Queries.vue` 和 `Articles.vue` 依赖桌面表格与 inline 表单，`ArticleDetail.vue` 的详情布局固定为两列。目标是在不改动后端 API 的前提下，复用现有 Element Plus 和数据流，为门户增加更完整的响应式应用外壳、移动端导航和移动端友好的列表/详情布局。

## Recommended Approach

### 1. 改造应用外壳

修改 `web/portal/src/App.vue`：

- 保留 Element Plus，新增一个响应式 shell：
  - 桌面端：sticky 顶部 header、品牌名、横向导航、居中的主内容区域。
  - 移动端：紧凑 header、汉堡按钮、`el-drawer` 垂直导航。
- 在脚本中定义导航项（`/queries`、`/articles`），用当前 route 高亮菜单。
- 点击移动端导航后关闭 drawer。
- 在 `App.vue` scoped CSS 中加入根布局和全局 reset（通过 `:global`），例如 body margin reset、`box-sizing: border-box`、主区域最大宽度与移动端 padding。
- 推荐断点：`max-width: 767px` 为移动端。

### 2. 让 Query Tasks 页面适配移动端

修改 `web/portal/src/pages/Queries.vue`：

- 保留现有 `listQueries()` 数据加载逻辑和过滤字段。
- 将 inline filter 改造成可换行/可堆叠布局：桌面端横向，移动端单列、控件 100% 宽度。
- 移除 select 上的内联宽度，改用 CSS 控制。
- 桌面端继续使用现有 `el-table`，外层加横向滚动容器作为兜底。
- 移动端新增 card/list 视图，显示 keyword、platform、status、last_run、id 等关键信息；用 CSS 在移动端隐藏表格、显示卡片。
- 保留 loading、error、empty 状态。

### 3. 让 Articles 页面适配移动端

修改 `web/portal/src/pages/Articles.vue`：

- 保留现有 `listArticles()`、`open(row)`、过滤参数和桌面表格行为。
- 将 source/query/limit/offset 过滤区改为响应式布局：桌面端横向，移动端单列。
- 桌面端继续使用 `el-table` 和 row-click。
- 移动端新增文章卡片列表，显示 title、source、query_keyword、author、created_at、原文链接和明确的“详情”按钮。
- 原文链接使用 `@click.stop`，避免误触发详情跳转。
- 保留 total、loading、error 和空数据表现。

### 4. 让 Article Detail 页面适配移动端

修改 `web/portal/src/pages/ArticleDetail.vue`：

- 保留现有 `getArticle()` / `getArticleContent()` 数据加载逻辑。
- 增加移动端判断（可在本文件用 `window.matchMedia('(max-width: 767px)')` + `ref` / lifecycle 监听），将 `el-descriptions` 的 `column` 动态绑定为桌面 2 列、移动端 1 列。
- 详情 header 在窄屏下改为纵向堆叠，标题和链接允许换行。
- 优化 summary 和 raw content 的换行、最大宽度、移动端 padding/font-size，避免长内容撑破布局。

## Critical Files

- `web/portal/src/App.vue` — responsive shell、桌面/移动导航、全局页面布局。
- `web/portal/src/pages/Queries.vue` — 查询任务过滤区、桌面表格和移动卡片视图。
- `web/portal/src/pages/Articles.vue` — 文章过滤区、桌面表格和移动卡片视图。
- `web/portal/src/pages/ArticleDetail.vue` — 详情页响应式 descriptions、header、raw content。
- `web/portal/src/main.ts` — 目前只需复用现有路由和 Element Plus 引入，预计不必修改。

## Notes / Non-goals

- 不引入 Tailwind、Sass 或新的 UI 依赖；继续复用 Element Plus。
- 不改变 Go 后端 API、Vite proxy 或数据模型。
- 暂不处理 Go 后端是否生产环境直接 serve Vue `dist` 的问题；本次聚焦前端 UI 适配。
- 不重构成大量新组件，避免扩大范围；若后续页面增多，再抽取共享 Layout/Form/List 组件。

## Verification

1. 前端类型检查和构建：

   ```bash
   cd /Users/tianfei.gu/gfaia/projects/free-thinker/web/portal
   npm run build
   ```

2. 本地开发联调（需要 Go portal API 在 `127.0.0.1:8080`，Vite 已代理 `/api`）：

   ```bash
   cd /Users/tianfei.gu/gfaia/projects/free-thinker/web/portal
   npm run dev -- --host 127.0.0.1
   ```

3. 浏览器手动检查：

   - `/queries`：桌面表格正常；移动端显示查询任务卡片；过滤和刷新可用。
   - `/articles`：桌面表格和行点击正常；移动端显示文章卡片；详情按钮和外链行为正确。
   - `/articles/:id`：桌面详情两列；移动端详情一列；summary/raw content 不撑破屏幕。
   - 设备宽度建议覆盖 375px、414px、768px、1024px、1440px。

4. 生产预览可选验证：

   ```bash
   cd /Users/tianfei.gu/gfaia/projects/free-thinker/web/portal
   npm run preview -- --host 127.0.0.1
   ```
