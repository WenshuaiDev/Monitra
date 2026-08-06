# 01 — PostgreSQL 决定 Go 核心进程的首次启动结果

Type: implementation
Status: ready-for-agent
Blocked by: None — can start immediately

**What to build:** 建立第一个真实 Go 生产进程入口，使应用只在 Foundation 所有的 PostgreSQL 连接池成功连接后进入 ready；首次连接超过限定时间仍未成功时，核心进程完成有界清理并非零退出。

## Acceptance criteria

- [ ] 真实 PostgreSQL可用时，核心进程在有限时间内从 not ready 进入 ready。
- [ ] PostgreSQL不可连接时，readiness 从未成功，核心进程在启动期限结束后非零退出。
- [ ] 启动等待和失败期间，liveness 与 readiness 表达不同含义，结构化日志不泄露 Secret。
- [ ] 生产运行只创建一个由 Foundation 所有的有界关系数据库连接池。
- [ ] 根目录存在可重复执行的 Task 测试和演示入口。

## 它让系统向前移动什么

Monitra 第一次拥有能够根据正式数据库依赖给出真实启动结果的核心进程，而不是一个与部署依赖无关的空程序。

## 当前事实和调用链

当前只有 Go Module 声明，没有 Go 包、应用命令、类型化配置、连接池、Management Listener、Taskfile 或测试。

本 Ticket 建立：

```text
生产配置与 Secret 文件引用
→ Go Bootstrap
→ Foundation 创建唯一 PostgreSQL连接池
→ 在有限期限内验证连接
→ 启动并监督 Management Listener
→ readiness 成功

或

→ 连接期限耗尽
→ 有界清理
→ 核心进程非零退出
```

## 唯一主要结果

Go 核心进程的首次启动结果真实反映 PostgreSQL 是否在限定时间内可用。

## 输入和前置状态

- 可用或不可用的真实 PostgreSQL；
- 只通过文件引用提供的数据库 Secret；
- 有限的启动连接期限；
- Release Identity、监听地址和连接参数等最小类型化配置。

## 业务规则或待决问题

- 只有 Foundation 可以创建关系数据库连接池。
- liveness 不依赖 PostgreSQL；readiness 在连接成功前保持失败。
- 首次连接期限结束后，不允许核心进程无限等待或长期保持 not ready。
- 驱动、池容量和具体有限超时是实现参数，不是新的产品决策，但必须固定、可解释并被测试覆盖。

## 成功后的状态变化

Foundation 持有一个有界连接池；核心进程持续运行；liveness 成功；readiness 从失败变为成功。

## 失败或拒绝行为

故意配置一个不可连接的 PostgreSQL 地址。达到启动期限后，核心进程必须非零退出，readiness 从未成功，日志明确说明依赖启动失败但不输出 Secret 正文。

## 预计修改范围

- 根开发命令与工具版本约束；
- Go 核心应用入口和显式 Composition；
- 最小类型化配置及 Secret 文件读取；
- Foundation Runtime、PostgreSQL连接池和 Management Listener；
- Go 单元、真实 PostgreSQL集成和真实子进程测试。

## 自动测试

1. 真实 PostgreSQL可用时，进程启动并最终 ready。
2. PostgreSQL不可连接时，进程在限定时间内非零退出且从未 ready。
3. 启动等待期间 liveness 与 readiness 的外部行为正确。

## 运行演示

通过根 Task 启动测试 PostgreSQL和核心进程并查询 Management Listener；随后使用不可连接地址重复运行，展示 readiness 从未成功和非零退出码。

## 明确不做

- Application Listener 和启动握手；
- 成功启动后的数据库中断恢复；
- 业务表、Namespace 和 Migration；
- MySQL、Caddy、SPA 或 Playwright；
- 完整运维 CLI。

## 完成证据

- Task 命令及退出码；
- liveness/readiness 响应；
- 真实 PostgreSQL连接证据；
- 不含 Secret 正文的结构化启动日志；
- Go 测试结果。
