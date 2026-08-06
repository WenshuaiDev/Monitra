# 01 — Application Listener 停止接入并完成有界排空

Type: implementation
Status: ready-for-agent
Blocked by: None — can start immediately

**What to build:** 深化现有 Foundation HTTP Server 的正式关闭行为，使 Application Listener 开始关闭后立即停止接受新连接或新请求，允许已经进入 Handler 的请求在有限期限内完成，并在期限耗尽时强制收束仍在活动的连接并返回规范化失败。

## Acceptance criteria

- [ ] Application Shutdown 开始后，新的连接或请求不能再进入 Handler。
- [ ] 已经进入 Handler 的请求在排空期限内完成时，调用方取得正常请求结果，Shutdown 返回成功。
- [ ] 已进入 Handler 的请求超过排空期限时，活动连接被强制关闭，Shutdown 在有限时间内返回规范化超时失败。
- [ ] Application 排空期限明确为 5 秒，并由不依赖私有字段的行为测试覆盖。
- [ ] 生产 HTTP Interface 不增加慢请求参数、测试端点、故障开关或环境分支。

## 它让系统向前移动什么

Monitra 第一次拥有可被 Runtime 可靠使用的 Application 排空行为：关闭接缝既不会立即丢弃所有在途请求，也不会因一个卡住的请求无限等待。

## 当前事实和调用链

当前 Foundation HTTP Server 在真实 TCP Listener 上运行标准库 HTTP Server，并把非正常 Serve 错误报告到有缓冲的错误通道。Application 和 Management 都复用这个 Server。

当前 Shutdown 只直接调用标准库 graceful Shutdown。调用方负责创建期限，但 Server 本身不负责超时后的强制关闭，也不提供规范化的排空结果。Runtime 当前会吞掉 Shutdown 错误。

本 Ticket 只深化 HTTP Server 已经拥有的关闭责任：

```text
真实请求进入 Handler
→ 调用 Shutdown
→ Listener 停止接受新工作
→ 在途 Handler 在期限内完成
→ Shutdown 成功

或

→ Handler 超过期限
→ 强制关闭活动连接
→ Shutdown 返回规范化超时
```

## 唯一主要结果

Application Listener 能够停止接受新工作，并在成功完成或超时强制收束之间给出有限、明确的排空结果。

## 输入和前置状态

- 现有 Foundation HTTP Server 和 Application Listener；
- 一个已经进入 Handler 的真实 TCP 请求；
- 一个 5 秒排空期限；
- 测试代码中的受控 Handler，用于释放或持续阻塞请求。

## 业务规则或待决问题

- 正常关闭导致的标准 ServerClosed 结果不能被报告为意外 Listener 失败。
- Shutdown 开始后不得允许新工作延长排空窗口。
- 到达期限后必须主动强制收束，不能仅返回 Context 错误并留下活动请求。
- 返回给 Runtime 的错误必须是稳定类别，不暴露任意网络或 Handler 错误正文。
- Management 的具体停止顺序不在本 Ticket 决定；它由 Runtime 后续 Ticket 使用同一 Server 能力。

## 成功后的状态变化

Application 从接受请求的运行状态进入不再接受新工作的关闭状态；所有已进入请求在期限内结束后，Server 完成关闭并返回成功。

## 失败或拒绝行为

测试 Handler 故意保持阻塞超过排空期限。Server 必须在期限后强制关闭相关连接，向调用方返回规范化排空超时，并且测试不能无限等待。

## 预计修改范围

- Foundation HTTP Server 的关闭实现和规范化关闭结果；
- Application Listener 关闭行为测试；
- 真实 TCP 请求与受控 Handler 测试夹具。

## 自动测试

1. 在途请求于排空期限内完成，Shutdown 成功，关闭后的新请求不能进入。
2. 在途请求超过期限后被强制关闭，Shutdown 返回规范化超时。
3. 正常 Shutdown 不会向 Listener 错误通道报告关键 Serve 失败。

## 运行演示

运行一个 Go 行为测试：先让真实请求进入受控 Handler，再触发 Shutdown；分别释放 Handler 和保持阻塞，输出原请求结果、新请求拒绝结果、关闭耗时和规范化结果。

## 明确不做

- Runtime 状态和资源关闭顺序；
- Management liveness/readiness；
- 进程信号、退出码和 Compose；
- 业务 Handler、流式请求或上传接口；
- 通用 Listener 插件或生命周期注册。

## 完成证据

- 真实 TCP 请求的成功排空结果；
- draining 后新请求被拒绝的观察；
- 超时强制关闭和有限耗时；
- Go 测试结果；
- 生产 HTTP Interface 无测试控制的差异检查。
