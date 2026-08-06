# 04 — 浏览器 Bootstrap 只在兼容握手后创建 Router

Type: implementation
Status: resolved
Blocked by: 03 — 类型化启动握手返回可关联的扁平响应

**What to build:** 建立 React SPA 的永久启动状态机，使有效配置和兼容握手进入正常技术外壳；配置无效、后端不可用或版本不兼容时，在 Router 创建前显示准确失败状态。

## Acceptance criteria

- [x] runtime config 在 Foundation HttpClient 和 Router 创建前完成加载与严格校验。
- [x] 有效配置通过 Foundation HttpClient 和生成客户端执行启动握手。
- [x] 只有 API major 与 Release Identity 均兼容时才创建 Router并显示正常状态。
- [x] 正常状态显示 Release Identity、API major 和非空 request_id。
- [x] 无效配置、后端不可用、API major 不兼容和 Release Identity 不兼容分别显示明确失败状态，且不显示正常外壳。

## 它让系统向前移动什么

Monitra 第一次拥有不会在部署配置或后端不兼容时伪装正常的浏览器启动行为。

## 当前事实和调用链

Ticket 03 提供 OpenAPI 和生成客户端，但当前没有 Web 工程、runtime config、Foundation HttpClient、Router 或页面。

本 Ticket 建立：

```text
读取 runtime config
→ 严格校验
→ 创建 Foundation HttpClient
→ 创建生成客户端
→ 执行启动握手
→ 校验 API major 和 Release Identity
→ 成功后创建 Router
→ 显示正常启动状态
```

## 唯一主要结果

浏览器只有在 runtime config 有效且启动握手兼容时才创建 Router 并显示正常状态。

## 输入和前置状态

- Ticket 03 的 OpenAPI 与生成客户端；
- 部署期 runtime config Schema；
- 成功、不可用和不兼容的握手结果。

## 业务规则或待决问题

- runtime config 必须先于 Router 和 HttpClient。
- Bootstrap Config Loader 只负责读取部署配置，不成为第二个应用网络接口。
- 所有 API 请求通过 Foundation HttpClient。
- 失败状态必须明确、可访问，并且不能渲染正常外壳。
- 当前页面只表达技术启动状态，不引入业务导航或数据。

## 成功后的状态变化

Bootstrap 从 loading 进入 ready，创建一次 Router，并显示正常启动状态及握手身份信息。

## 失败或拒绝行为

分别提供无效 runtime config、不可达后端、API major 不兼容和 Release Identity 不兼容结果。每种情况都必须进入对应失败状态，并且不得创建正常 Router。

## 预计修改范围

- React、Vite、严格 TypeScript 和 pnpm 工程基线；
- Web Bootstrap 与 runtime config 校验；
- Foundation HttpClient 和生成客户端 Adapter；
- Router 与最小技术状态页面；
- React行为测试。

## 自动测试

1. runtime config 在 HttpClient 和 Router 之前加载，成功后 Router 只创建一次。
2. 无效配置和后端不可用显示各自失败状态。
3. API major 或 Release Identity 不兼容时拒绝正常启动。

## 运行演示

使用生产 Web 构建依次提供有效和无效配置，展示正常状态与四类失败状态；该演示不替代后续 Caddy 生产栈验收。

## 明确不做

- 业务页面、业务导航或业务数据；
- 完整 UI 组件库；
- TanStack Query、Redux、Zustand；
- Service Worker、PWA、SSR；
- E2E 专用生产分支或故障注入接口。

## 完成证据

- Bootstrap 执行顺序测试；
- 成功和失败渲染测试；
- TypeScript检查；
- 生产 Web 构建结果。
