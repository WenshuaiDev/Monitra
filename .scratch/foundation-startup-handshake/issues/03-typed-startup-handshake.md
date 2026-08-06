# 03 — 类型化启动握手返回可关联的扁平响应

Type: implementation
Status: resolved
Blocked by: 01 — PostgreSQL 决定 Go 核心进程的首次启动结果

**What to build:** 建立第一个版本化 HTTP 技术闭环，使调用方通过 OpenAPI 契约取得当前 Release Identity、API major 和服务端生成的 `request_id`，并能用该 ID 关联响应和结构化日志。

## Acceptance criteria

- [x] `/api/v1` 下存在由 OpenAPI 描述的启动握手。
- [x] 成功响应包含 `code`、`message`、`data`、`request_id`，且 `data` 至少包含 Release Identity 和 API major。
- [x] 同一服务端生成的 request ID 出现在响应头、响应体和对应结构化日志中。
- [x] 客户端提交的 request ID 不会成为服务端权威 request ID。
- [x] TypeScript 客户端可重复生成，并通过可注入的 Foundation HttpClient 执行器发起请求。

## 它让系统向前移动什么

Monitra 第一次拥有可以被浏览器类型化调用、可以判断版本兼容并可以关联日志的正式 Application Listener 行为。

## 当前事实和调用链

Ticket 01 只有 Management Listener，没有 Application Listener、OpenAPI、生成客户端或公开版本化请求。

本 Ticket 建立：

```text
GET /api/v1/<startup-handshake>
→ 服务端生成 request_id
→ handwritten net/http Handler
→ 扁平响应序列化
→ 响应头、响应体和 slog 日志关联
→ OpenAPI 契约验证
→ 生成 TypeScript transport 客户端
```

## 唯一主要结果

启动握手通过版本化 HTTP Interface 返回符合 OpenAPI 的扁平响应，并能用 request_id 关联响应和日志。

## 输入和前置状态

- Ticket 01 的运行进程和 Release Identity；
- 已确认的 `/api/v1` 主版本前缀；
- 当前 API major；
- 固定版本的 OpenAPI 验证和 TypeScript 生成工具。

## 业务规则或待决问题

- 普通响应包含 `code`、`message`、`data`、`request_id`；失败响应的 `data` 为 null。
- request_id 必须由服务端生成，不能信任客户端输入。
- Go Server、Handler 和路由保持手写，Foundation HTTP Interface 使用 `net/http`。
- 生成客户端不能内置独立网络、认证、关联或错误处理策略。

## 成功后的状态变化

调用方取得类型化成功响应；同一 request_id 出现在响应头、响应体和对应的一条结构化请求日志中。

## 失败或拒绝行为

客户端故意提交自定义 request ID。服务端必须忽略它并生成新的 request_id。对不支持的方法或路径使用真实 HTTP 失败状态和扁平失败响应。

## 预计修改范围

- Application Listener 和显式路由装配；
- Foundation HTTP 响应、request_id 和日志策略；
- 启动握手 Handler；
- OpenAPI 契约、验证和 TypeScript 生成；
- Go 契约测试。

## 自动测试

1. 成功响应字段、HTTP 状态和 OpenAPI 契约一致。
2. 客户端 request ID 不会成为服务端 request_id。
3. 响应头、响应体和结构化日志使用同一 request_id。

## 运行演示

启动 Go 核心进程，调用握手端点，并展示响应头、JSON 响应和对应的一条 slog JSON 日志。

## 明确不做

- Go Server、Handler、路由或业务代码生成；
- 业务语义结果码；
- Authentication、Session 或权限；
- Caddy、浏览器或完整 HTTP 错误分类体系。

## 完成证据

- 完整且可验证的 OpenAPI；
- 可重复生成命令及无漂移结果；
- Go 契约测试；
- 一组真实响应和日志关联证据。
