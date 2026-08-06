# 04 — Application 与 Management 意外停止被监督为关键失败

Type: implementation
Status: ready-for-agent
Blocked by: 02 — 正常停止请求驱动 Runtime 有序 draining 并成功返回; 03 — 排空超时或资源清理失败产生非零 Runtime 结果

**What to build:** 让已经由 Runtime 监督的 Application 和 Management Listener 在非正常 Shutdown 条件下意外停止时，被统一识别为关键组件失败；Runtime 对其余 Listener 和 PostgreSQL执行有界清理，并返回非零结果。

## Acceptance criteria

- [ ] Application Listener 的真实意外 Serve 失败触发 readiness 失败、其余资源有界清理和非零 Runtime 结果。
- [ ] Application 失败后，只要 Management 尚未停止，liveness 保持成功而 readiness 为失败。
- [ ] Management Listener 的真实意外 Serve 失败触发 Application 有界排空、PostgreSQL关闭和非零 Runtime 结果。
- [ ] Management 已失败时不伪造 liveness 可用，但 Runtime 仍监督并完成其余清理。
- [ ] 正常 HTTP Shutdown 不会被 Listener 错误通道重新报告为关键失败。
- [ ] Listener 失败通过真实 Listener 条件触发，不通过停止 PostgreSQL、生产 HTTP 控制或隐藏测试模式触发。

## 它让系统向前移动什么

Monitra 能够准确识别两个关键 HTTP Listener 的意外退出，避免核心进程在公开请求入口或运维健康入口已经丢失时继续以部分运行状态存活。

## 当前事实和调用链

现有 Runtime 已经选择两个 Listener 的错误通道。Application 或 Management 报告非正常 Serve 错误时，Runtime 返回不同的稳定错误。当前缺口不在于发现通道，而在于关键失败后的 draining、资源清理结果、健康状态时间线和非零最终结果尚未完整证明。

本 Ticket 扩展现有接缝：

```text
Application 或 Management 意外 Serve 失败
→ Runtime 记录关键组件类别
→ readiness 失败（Management仍可用时可观察）
→ 有界收束其余 Listener
→ 有界关闭 PostgreSQL
→ Runtime 返回失败
```

## 唯一主要结果

任一受监督 Listener 意外停止都会被视为关键失败，并使 Runtime 在有界清理其余资源后返回非零结果。

## 输入和前置状态

- Ticket 02 的正常 draining 路径；
- Ticket 03 的清理失败传播；
- 一个已经进入 ready 的 Runtime；
- 可以在测试所有权 seam 上产生真实 Serve 失败的 Application 和 Management Listener。

## 业务规则或待决问题

- Application 和 Management 都是关键组件，不进行自动重启。
- 关键失败原因为 `application_listener_failed` 或 `management_listener_failed` 等有限类别。
- Application 失败时，Management 在其正常关闭前继续提供 live/not-ready。
- Management 失败时，无法继续提供 liveness 是真实失败结果，不用替代端点掩盖。
- PostgreSQL临时不可用继续是可恢复依赖状态，不属于本 Ticket 的 Listener 失败。

## 成功后的状态变化

正常运行时两个 Listener 持续受监督；任一 Listener 意外失败后，Runtime 不可逆地进入失败 draining，完成有界清理并进入 stopped(failure)。

## 失败或拒绝行为

分别故意破坏真实 Application 和 Management Listener 的 Serve 条件。Runtime 必须拒绝把该事件当成正常 Shutdown 或数据库故障，并且两个场景最终都返回失败。

## 预计修改范围

- Runtime 关键触发原因和失败 draining；
- HTTP Server 错误通道与正常 Shutdown 区分；
- 真实 Listener 条件测试夹具；
- Management健康和资源清理观察；
- 关键失败结构化日志。

## 自动测试

1. Application Listener 意外失败后 Management为 live/not-ready，随后全部资源清理且 Runtime 失败。
2. Management Listener 意外失败后 Application被有界排空，PostgreSQL关闭且 Runtime 失败。
3. 正常 Shutdown 不产生额外关键 Listener 失败。

## 运行演示

运行两个真实 Listener 失败场景，展示触发组件、健康状态、其余资源清理、最终 Runtime 结果和有限总耗时。

## 明确不做

- Listener 自动重启、熔断或高可用；
- PostgreSQL故障模拟；
- 生产故障注入接口；
- 动态组件注册或通用 Supervisor；
- 真实操作系统子进程验收。

## 完成证据

- Application 和 Management 两类真实失败观察；
- Application失败期间的 live/not-ready 证据；
- 其余资源有界清理证据；
- 非零 Runtime 结果；
- 无生产测试控制和无底层错误泄露检查。
