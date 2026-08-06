# 05 — 生产构建栈通过 Caddy 形成同源启动闭环

Type: implementation
Status: resolved
Blocked by: 02 — 数据库临时中断后同一核心进程自动恢复就绪; 03 — 类型化启动握手返回可关联的扁平响应; 04 — 浏览器 Bootstrap 只在兼容握手后创建 Router

**What to build:** 让操作者使用正式生产构建产物启动 PostgreSQL、Go 和 Caddy，并从唯一公开入口加载 SPA、读取部署期 runtime config、完成同源启动握手。

## Acceptance criteria

- [x] PostgreSQL、Go 核心应用和 Caddy 均从正式生产构建产物启动。
- [x] Caddy 提供 SPA 和 `/api/v1` 同源代理，但不暴露 Management Listener。
- [x] 最小运维配置投影生成严格且不含 Secret 的 runtime config。
- [x] 浏览器从 Caddy 完成兼容握手并显示正常启动状态。
- [x] 生产容器满足非 root、只读根文件系统、最小 Capability 和 no-new-privileges 约束。

## 它让系统向前移动什么

Monitra 第一次形成真实的公开浏览器入口、生产 SPA、Go 核心应用和正式数据库依赖闭环。

## 当前事实和调用链

前序 Ticket 已建立各进程内行为，但尚无 OCI 镜像、Compose、Caddy、部署配置投影或正式同源拓扑。

本 Ticket 建立：

```text
最小运维配置校验与 Web runtime config 投影
→ PostgreSQL
→ Go 核心应用
→ Caddy
→ 浏览器加载生产 SPA
→ /api/v1 同源代理
→ 启动握手
```

## 唯一主要结果

一个正式生产构建栈可以通过 Caddy 完成浏览器到 Go 和 PostgreSQL的启动闭环。

## 输入和前置状态

- 前序 Ticket 的生产代码；
- 不含 Secret 正文的部署配置；
- 数据库 Secret 文件；
- immutable Release Identity；
- Docker Engine 和 Compose。

## 业务规则或待决问题

- 只引入当前 runtime config 所需的最小 Go 运维配置校验和投影行为，不扩展完整运维 CLI。
- Caddy 是唯一公开 Web 入口；Management Listener 不被代理或发布。
- SPA 使用生产静态构建，不运行生产 Node.js服务器。
- HTML 和 runtime config 必须重新验证；哈希资产可以不可变缓存。
- Caddy覆盖转发头；容器遵守总体工程基座的安全约束。

## 成功后的状态变化

三个正式服务按依赖关系启动；Caddy 对外提供 SPA 和同源 API；浏览器完成兼容握手并显示正常状态。

## 失败或拒绝行为

故意使 Go 核心应用不可用。Caddy不得因此暴露 Management Listener；SPA仍可从 Caddy 加载，但启动握手必须显示明确后端不可用状态。

## 预计修改范围

- Go 与 Web 多阶段生产镜像；
- Caddy静态资源和代理配置；
- Compose定义、Healthcheck和安全约束；
- 最小运维配置校验及 runtime config 投影；
- 根 Task 构建、启动、停止和诊断入口；
- 部署配置样例和 Secret 引用。

## 自动测试

1. 生产镜像可重复构建且以非 root 身份运行。
2. Caddy提供 SPA 深链接和 `/api/v1` 同源代理，但不提供 Management Listener。
3. runtime config 投影不包含 Secret，并在无效输入时失败。

## 运行演示

通过根 Task 构建并启动生产栈，从 Caddy 打开 SPA，展示握手结果、Compose健康状态和内部 Management 状态。

## 明确不做

- 正式 HTTPS；
- 完整运维 CLI 生命周期；
- Release Bundle、签名或离线安装；
- 自动 Migration；
- 业务服务、业务页面或生产集群。

## 完成证据

- 生产镜像摘要；
- Compose服务与健康状态；
- 浏览器成功页面；
- Caddy路由和Management隔离验证；
- 容器权限检查。
