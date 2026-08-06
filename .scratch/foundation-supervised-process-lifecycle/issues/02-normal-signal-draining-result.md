# 02 — 正常停止请求驱动 Runtime 有序 draining 并成功返回

Type: implementation
Status: ready-for-agent
Blocked by: 01 — Application Listener 停止接入并完成有界排空

**What to build:** 扩展现有 Foundation Runtime，使受控的 SIGINT 或 SIGTERM 停止原因进入同一条显式 draining 路径，先撤销 readiness，再排空 Application、停止 Management、关闭 PostgreSQL，并且只在全部步骤成功时返回正常结果。

## Acceptance criteria

- [ ] SIGINT 和 SIGTERM 都能被命令层转换为受控的正常停止原因，并进入现有 Runtime。
- [ ] draining 开始后 readiness 先变为失败，且不能因迟到的 PostgreSQL 恢复重新成功。
- [ ] Application 排空期间，只要 Management 尚未关闭，liveness 保持成功。
- [ ] 关闭顺序固定为 Application、Management、PostgreSQL。
- [ ] 正常停止且全部清理成功时 Runtime 返回成功。
- [ ] Runtime 保持一个外部生命周期 Interface，不引入通用组件注册或第二套关闭机制。

## 它让系统向前移动什么

Monitra 第一次把正常操作系统停止请求表达为明确、不可逆、可观察的进程状态变化，而不是 Context 取消后依靠无结果 defer 隐式清理。

## 当前事实和调用链

命令层当前使用操作系统信号取消 Context。Runtime 在监督循环看到 Context 完成后返回成功，并由延迟 shutdown 标记 not-ready、关闭两个 Listener 和 PostgreSQL。

当前顺序形状已经存在，但没有正式 draining 状态，Runtime 不能区分具体正常信号，shutdown 不能返回结果，也没有防止迟到 readiness 更新重新生效的状态不变量。

本 Ticket 建立：

```text
受控 SIGINT 或 SIGTERM 原因
→ Runtime 进入 draining
→ readiness = false
→ Application 有界排空
→ Management 停止
→ PostgreSQL 关闭
→ Runtime 返回成功
```

## 唯一主要结果

正常停止请求使 Runtime 按固定顺序完成一次成功 draining，并返回可映射为零退出码的正常结果。

## 输入和前置状态

- Ticket 01 的 Application 有界排空能力；
- 已启动的 Management、PostgreSQL 和 Application；
- 已经成功进入 ready 的 Runtime；
- 受控的 SIGINT 或 SIGTERM 停止原因。

## 业务规则或待决问题

- 命令层继续拥有操作系统信号注册，Runtime 不成为通用信号框架。
- 第一个正常停止触发使状态不可逆地进入 draining。
- readiness 的撤销发生在 Application 排空完成之前。
- PostgreSQL 探测或恢复不能使 draining 状态重新 ready。
- Management 在 Application 排空期间保持 live，并且只在 Application 被收束后停止。
- 未成功启动的资源跳过关闭，但其他已启动资源仍保持顺序。

## 成功后的状态变化

Runtime 从 ready 或 dependency_unavailable 进入 draining，再进入 releasing 和 stopped；readiness 从此保持失败；全部资源成功清理后 Runtime 返回正常结果。

## 失败或拒绝行为

在 draining 开始后故意产生一次 PostgreSQL 恢复或迟到的 ready 更新。Management readiness 必须持续返回失败，Runtime 不得重新进入 ready 或重新接受新的 Application 工作。

## 预计修改范围

- 命令层的受控停止原因映射；
- Foundation Runtime 的显式监督结果、draining 状态和清理顺序；
- Management readiness 协调；
- Runtime 行为测试和结构化时间线。

## 自动测试

1. 正常停止后 readiness 先失败，Application 排空期间 liveness 保持成功。
2. Application、Management、PostgreSQL 按固定顺序成功关闭，Runtime 返回成功。
3. draining 期间的 PostgreSQL 恢复不能重新建立 readiness。

## 运行演示

运行 Runtime 行为测试，输出 ready、draining live/not-ready、Application 结束、Management 结束、PostgreSQL 关闭和最终成功结果的时间线。

## 明确不做

- 清理超时或失败的最终非零结果；
- Listener 意外 Serve 失败；
- 真实操作系统子进程信号验收；
- Compose stop grace period；
- 通用生命周期参与者或资源扫描。

## 完成证据

- 健康状态观察序列；
- 资源关闭顺序日志；
- Runtime 正常结果；
- 迟到 readiness 更新被拒绝的测试；
- 无第二套 Runtime 或通用注册机制的代码审查。
