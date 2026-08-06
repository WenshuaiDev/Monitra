# 02 — 数据库临时中断后同一核心进程自动恢复就绪

Type: implementation
Status: resolved
Blocked by: 01 — PostgreSQL 决定 Go 核心进程的首次启动结果

**What to build:** 让已经成功进入 ready 的核心进程在 PostgreSQL 临时中断时保持存活并转为 not ready，在数据库恢复后由原有单一连接池自动恢复连接，使同一进程在有限时间内重新 ready。

## Acceptance criteria

- [x] PostgreSQL中断后，liveness 保持成功而 readiness 在有限时间内失败。
- [x] PostgreSQL恢复后，原有连接池自动恢复，readiness 在有限时间内重新成功。
- [x] 数据库中断和恢复前后，核心进程 PID 或容器实例身份不变。
- [x] PostgreSQL持续不可用时，核心进程保持 live、not ready，不伪造恢复。
- [x] 恢复机制不创建第二连接池，也不透明重试业务语句。

## 它让系统向前移动什么

Monitra 可以区分不可恢复的首次启动失败和可恢复的运行期依赖中断，避免数据库短暂故障造成核心应用重启。

## 当前事实和调用链

Ticket 01 建立首次连接、单一连接池及 Management Listener，但尚未证明已经 ready 的进程如何处理数据库中断。

本 Ticket 建立：

```text
核心进程已 ready
→ PostgreSQL中断
→ 连接健康探测失败
→ readiness 失败、liveness 成功
→ PostgreSQL恢复
→ 同一连接池恢复连接
→ readiness 在有限时间内重新成功
```

## 唯一主要结果

PostgreSQL恢复后，同一 Go 核心进程无需重启即可自动恢复 readiness。

## 输入和前置状态

- Ticket 01 已完成；
- 核心进程至少成功进入过一次 ready；
- 可被停止和恢复的真实 PostgreSQL实例；
- 明确有限的健康探测和恢复观察期限。

## 业务规则或待决问题

- 运行期数据库中断不得触发核心进程退出或重启。
- liveness 不依赖数据库；readiness 必须反映当前数据库连接状态。
- 必须复用 Foundation 已有连接池，不得以重建池掩盖恢复问题。
- 连接恢复不是业务操作重试；当前前沿也不存在业务语句。

## 成功后的状态变化

数据库中断时 readiness 从成功变为失败；数据库恢复后 readiness 从失败重新变为成功；进程身份和 liveness 全程保持稳定。

## 失败或拒绝行为

让 PostgreSQL持续不可用并超过恢复观察窗口。核心进程必须继续存活且 liveness 成功，readiness 必须持续失败，不得伪造恢复或自行创建替代连接池。

## 预计修改范围

- Foundation 连接池健康探测与恢复监督；
- Runtime 状态协调；
- Management readiness；
- 真实 PostgreSQL和子进程集成测试。

## 自动测试

1. 中断 PostgreSQL 后 readiness 在有限时间内失败而 liveness 成功。
2. 恢复 PostgreSQL 后 readiness 在有限时间内重新成功。
3. 中断和恢复前后进程身份不变，且只存在一个连接池。

## 运行演示

启动核心进程至 ready，记录进程身份，停止 PostgreSQL并观察健康状态，再恢复数据库并证明同一进程自动恢复 ready。

## 明确不做

- 自动 Migration；
- 业务数据库请求的错误映射或透明重试；
- 多连接池、数据库故障转移或高可用；
- PostgreSQL容器自身的长期重启策略。

## 完成证据

- 中断与恢复时间线；
- 相同进程身份；
- liveness/readiness 响应序列；
- 连接池恢复日志；
- 集成测试结果。
