# 03 — 排空超时或资源清理失败产生非零 Runtime 结果

Type: implementation
Status: ready-for-agent
Blocked by: 02 — 正常停止请求驱动 Runtime 有序 draining 并成功返回

**What to build:** 让 Application 排空超时、Management 停止失败或 PostgreSQL 关闭超时中的任一情况都改变 Runtime 的最终结果，同时继续有界尝试其余清理步骤，使进程不会无限等待或把不完整清理报告为正常停止。

## Acceptance criteria

- [ ] Application 排空超过 5 秒后被强制收束，Runtime 继续停止 Management 和 PostgreSQL并返回失败。
- [ ] Management Shutdown 失败时，Runtime 继续关闭 PostgreSQL并返回失败。
- [ ] PostgreSQL 连接池关闭具有 2 秒有限预算，超过期限时 Runtime 返回失败而不无限等待。
- [ ] Runtime 的整体内部关闭预算不超过 9 秒加有限调度余量。
- [ ] 原始触发原因和后续清理失败都被保留为受控类别，不打印不受控底层错误文本。
- [ ] 多个清理步骤同时失败时，所有仍可执行的清理步骤都被尝试。

## 它让系统向前移动什么

Monitra 的关闭结果第一次真实反映排空和资源释放是否完成，避免日志虽然报错但进程仍零退出，也避免一个卡住的资源永久阻塞停止。

## 当前事实和调用链

Ticket 02 建立正常 draining 和成功结果，但清理阶段尚未形成完整失败语义。当前 PostgreSQL Pool Close 没有 Context 或返回结果，现有 Runtime shutdown 也不能把 Listener Shutdown 错误向上返回。

本 Ticket 建立：

```text
draining
→ 某一清理阶段超时或失败
→ 记录受控失败类别
→ 继续后续清理
→ 汇总原始触发和清理结果
→ Runtime 在整体预算内返回失败
```

## 唯一主要结果

任何排空超时或资源清理失败都会产生有界、明确、非零的 Runtime 结果，而不会阻止其余资源的清理尝试。

## 输入和前置状态

- Ticket 02 的显式正常 draining 路径；
- 一个超过 Application 排空期限的受控 Handler；
- 一个可返回关闭失败的受控 Management Listener；
- 一个可保持关闭阻塞的受控 PostgreSQL 资源；
- 5 秒、2 秒、2 秒的分阶段预算和 9 秒整体预算。

## 业务规则或待决问题

- 清理是尽力而为但结果准确，不是遇到第一个错误立即跳过后续资源。
- Application 超时后必须先强制收束，才能进入 Management 阶段。
- Management 停止后才进入 PostgreSQL 关闭。
- PostgreSQL 底层 Close 即使缺少 Context，也不能控制主进程无限等待。
- Runtime 返回的错误类别保持稳定，日志不直接序列化底层错误正文。
- 正常停止触发后发生清理失败，最终结果仍为失败。

## 成功后的状态变化

全部阶段成功时行为与 Ticket 02 相同；任一阶段失败时 Runtime 仍进入 stopped，但最终结果为失败并包含受控原因类别。

## 失败或拒绝行为

分别故意阻塞 Application、让 Management 关闭失败、让 PostgreSQL Close 超过预算。每个场景都必须在有限时间内结束、继续执行仍可执行的后续清理，并返回失败。

## 预计修改范围

- Runtime 清理结果汇总和整体预算；
- Management 与 PostgreSQL 有界关闭调用；
- 规范化生命周期失败类别；
- 排空超时和资源失败行为测试；
- 安全结构化日志。

## 自动测试

1. Application 排空超时后强制收束并继续清理，Runtime 非零。
2. Management 停止失败后 PostgreSQL仍被关闭，Runtime 非零。
3. PostgreSQL Close 超时不会使 Runtime 超过整体预算，且多个失败不会被吞掉。

## 运行演示

运行三类受控失败场景，输出每个阶段的开始、结果、后续清理、总耗时和最终 Runtime 失败类别。

## 明确不做

- 操作系统进程退出码；
- Application 或 Management 意外 Serve 退出；
- Compose停止配置；
- PostgreSQL驱动替换或连接池抽象；
- 自动重试清理或后台清理任务。

## 完成证据

- 三类失败场景的有限耗时；
- 失败后仍执行后续清理的观察；
- 规范化 Runtime 失败结果；
- 结构化日志无 Secret、请求正文和底层错误文本；
- Go 行为测试结果。
