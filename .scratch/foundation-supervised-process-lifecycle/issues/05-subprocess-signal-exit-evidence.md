# 05 — 真实子进程证明信号类别、日志时间线和退出码

Type: evidence
Status: ready-for-agent
Blocked by: 03 — 排空超时或资源清理失败产生非零 Runtime 结果; 04 — Application 与 Management 意外停止被监督为关键失败

**What to build:** 使用真实操作系统子进程、正式核心二进制和真实 PostgreSQL，重复证明进入 ready 的进程分别接收 SIGINT 和 SIGTERM 后完成零退出，并证明受控 Runtime 或 Listener 失败能够稳定映射为非零进程结果。

## Acceptance criteria

- [ ] 正式构建的核心二进制在真实 PostgreSQL上进入 ready 后接收 SIGINT，并在 20 秒观察期限内零退出。
- [ ] 同一正式构建在进入 ready 后接收 SIGTERM，并在 20 秒观察期限内零退出。
- [ ] 两种正常信号的日志都包含 draining、Application停止接入、排空结果、Management停止、PostgreSQL关闭和最终成功，且顺序正确。
- [ ] 一个不修改生产行为的受控 Runtime或 Listener 失败子进程产生非零退出码。
- [ ] 子进程测试观察真实健康响应、日志和操作系统退出状态，不只断言函数返回或私有字段。
- [ ] 日志不包含 PostgreSQL Secret、请求正文或不受控底层错误文本。

## 它让系统向前移动什么

本前沿的 Runtime 语义被提升为真实操作系统进程证据，证明信号注册、主命令退出码映射和正式构建产物都与模块内行为一致。

## 当前事实和调用链

现有启动集成测试已经构建真实核心二进制、启动真实 PostgreSQL、等待 Management ready、发送真实 SIGTERM并确认进程在三秒内零退出。它还没有覆盖 SIGINT、draining 健康时间线、完整资源关闭日志或非零生命周期失败映射。

本 Ticket 扩展这条永久接缝：

```text
正式核心二进制
→ 真实 PostgreSQL
→ Management ready
→ 真实 SIGINT 或 SIGTERM
→ 观察 health、logs、Wait/ExitCode
→ 零退出

受控测试子进程生命周期失败
→ 非零退出
```

## 唯一主要结果

操作系统层面的进程退出状态准确表达正常信号成功和受控生命周期失败。

## 输入和前置状态

- Ticket 03、04 的最终 Runtime 结果语义；
- 正式构建的核心二进制；
- 真实 PostgreSQL实例和 Secret 文件；
- 独立的 Application 与 Management 地址；
- 真实 SIGINT、SIGTERM和有限观察期限。

## 业务规则或待决问题

- `Wait` 成功或 ExitCode 0 才表示正常信号完成。
- ExitCode 137、143 或超出观察期限都不是正常排空结果。
- 正常信号测试必须从 ready 开始，不以启动期间取消替代生产验收。
- 失败分支可以使用测试专属子进程夹具，但不能增加生产命令、生产环境开关或 HTTP 控制。
- PostgreSQL运行期中断不用于制造非零进程结果。

## 成功后的状态变化

正式子进程从 ready 经 draining 和 releasing 进入 stopped(success)，操作系统观察到零退出；受控失败子进程进入 stopped(failure)，操作系统观察到非零退出。

## 失败或拒绝行为

故意占用必要 Listener 地址，或运行只存在于测试范围的受控失败子进程夹具。进程必须返回非零；验收拒绝把启动或 Runtime 失败吞成零退出。

## 预计修改范围

- 现有真实 PostgreSQL子进程集成测试；
- SIGINT/SIGTERM测试矩阵；
- 健康状态、日志事件和退出码观察工具；
- 测试范围内的受控失败子进程夹具；
- 根生命周期集成测试入口。

## 自动测试

1. 正式核心二进制接收真实 SIGINT 后完整时间线和零退出。
2. 正式核心二进制接收真实 SIGTERM 后完整时间线和零退出。
3. 受控 Runtime或 Listener 失败子进程产生非零退出。

## 运行演示

运行根生命周期集成 Task，依次输出 SIGINT、SIGTERM 和失败场景的 ready 状态、信号时间、生命周期日志、总耗时和最终退出码。

## 明确不做

- 生产慢请求端点或故障控制；
- 通过停止 PostgreSQL制造关键失败；
- Compose stop grace period；
- 浏览器或 Caddy行为；
- 进程自动重启。

## 完成证据

- 两种真实信号及其零退出码；
- 受控失败的非零退出码；
- 完整有序结构化日志；
- 20 秒内有限完成时间；
- Secret和测试控制污染检查。
