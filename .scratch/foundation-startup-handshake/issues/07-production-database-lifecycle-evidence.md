# 07 — 生产栈证明数据库首次失败及运行期自动恢复

Type: evidence
Status: resolved
Blocked by: 05 — 生产构建栈通过 Caddy 形成同源启动闭环

**What to build:** 在正式 Compose 运行形态中证明首次数据库连接失败必然使 Go 核心进程非零退出，并证明已经 ready 的核心进程能够在数据库中断和恢复期间保持同一实例、自动重新 ready。

## Acceptance criteria

- [x] 首次启动时 PostgreSQL不可连接，Go 核心进程在限定时间后非零退出且从未 ready。
- [x] 正常运行后停止 PostgreSQL，核心进程不退出、不重启，liveness 成功、readiness 失败。
- [x] 恢复 PostgreSQL后，同一连接池自动恢复，readiness 在有限时间内重新成功。
- [x] 中断前后 Go 核心进程 PID 或容器实例 ID 不变。
- [x] 所有数据库控制都位于外部测试编排，生产代码无测试专用逻辑。

## 它让系统向前移动什么

将 Go 集成接缝中的数据库状态机提升为正式生产容器拓扑中的运行证据，排除 Compose重启策略或第二连接池掩盖问题。

## 当前事实和调用链

Ticket 01、02 已实现并测试数据库状态机，Ticket 05 提供生产栈；尚缺正式容器级证据。

本 Ticket 验收：

```text
Compose启动或停止 PostgreSQL
→ Go单一连接池
→ Management liveness/readiness
→ 核心进程退出码或容器实例状态
```

## 唯一主要结果

正式生产栈能重复证明数据库首次不可用导致非零退出，以及运行期中断恢复不需要重启核心进程。

## 输入和前置状态

- Ticket 05 的生产镜像与 Compose；
- 可控制启停的 PostgreSQL容器；
- 可读取的核心进程身份、健康响应和日志。

## 业务规则或待决问题

- 首次连接超时必须非零退出，不得保持无限 not ready。
- 运行期中断不退出或重启核心进程。
- 恢复后 readiness 必须在有限时间内重新成功。
- 验收控制仅存在于外部编排，不进入生产二进制或镜像运行逻辑。

## 成功后的状态变化

运行期证据必须完整呈现：

```text
ready
→ PostgreSQL中断
→ live + not ready
→ PostgreSQL恢复
→ 同一核心进程重新 ready
```

## 失败或拒绝行为

在首次启动前保持 PostgreSQL不可连接。Go 核心容器必须在限定时间后以非零状态退出，且不得出现任何 ready 记录。

## 预计修改范围

- 根生产栈生命周期验收脚本或测试；
- Compose外部测试编排；
- Management轮询和进程身份记录；
- Task验收入口。

## 自动测试

1. 首次数据库不可连接时核心进程非零退出且从未 ready。
2. 运行期中断时 liveness 成功、readiness 失败、核心进程未重启。
3. 数据库恢复后同一进程在有限时间内重新 ready。

## 运行演示

运行一个根 Task 命令，输出首次启动失败结果以及运行期中断、恢复的时间线、进程身份和健康状态。

## 明确不做

- 浏览器故障注入；
- 数据库高可用、复制或故障切换；
- 业务语句重试；
- Migration、业务数据或数据恢复；
- 修改生产代码以控制数据库状态。

## 完成证据

- 两个场景的命令记录和退出码；
- liveness/readiness时间线；
- 中断前后相同的进程或容器身份；
- PostgreSQL恢复后的连接日志；
- 无测试专用生产逻辑检查。
