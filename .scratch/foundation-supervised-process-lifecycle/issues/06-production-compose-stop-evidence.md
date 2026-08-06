# 06 — 生产 Compose 在 grace period 内完成正常停止

Type: evidence
Status: ready-for-agent
Blocked by: 05 — 真实子进程证明信号类别、日志时间线和退出码

**What to build:** 在正式生产 Compose 拓扑中明确 core 的 SIGTERM 和 stop grace period，并使用正式生产镜像证明 ready 的核心容器能够在 grace period 内完成完整 draining、零退出且不依赖 Docker SIGKILL。

## Acceptance criteria

- [ ] 生产 Compose 明确声明 core 使用 SIGTERM 和 15 秒 stop grace period。
- [ ] stop grace period 大于应用内部 9 秒最坏关闭预算，测试观察期限大于 stop grace period。
- [ ] 正式 PostgreSQL、core和 Caddy启动后，core进入 ready并记录稳定容器 ID、PID和 RestartCount。
- [ ] 通过正式 Compose停止 core后，容器在 grace period内退出，ExitCode为0且 RestartCount不变。
- [ ] 验收拒绝 ExitCode 137、143、超时、容器重启或缺失生命周期日志。
- [ ] core日志按顺序包含 ready、draining、Application结果、Management结果、PostgreSQL关闭和最终成功。
- [ ] 测试控制全部位于外部编排，不进入生产二进制或生产镜像。

## 它让系统向前移动什么

Monitra 的正常退出从 Go 子进程证据提升为正式生产容器拓扑证据，证明 Docker 的停止预算和应用内部排空期限协调，日常停止不依赖强制杀死进程。

## 当前事实和调用链

当前生产 Compose 已经构建正式 core和 Web镜像，启动 PostgreSQL、core、runtime-config和 Caddy，并通过 core readiness Healthcheck建立生产启动闭环。

现有生产栈测试会停止 core以验证 Caddy返回后端不可用，但不检查停止信号、grace period、core退出码、关闭耗时或完整生命周期日志。当前 Compose 也没有为 core显式声明停止策略。

本 Ticket 建立：

```text
正式 Compose启动
→ PostgreSQL/core/Caddy ready
→ 记录容器身份和健康
→ Compose向 core发送 SIGTERM
→ 应用内部 draining
→ core在 grace period内 ExitCode 0
→ 验证无重启、无强制 kill、日志完整
```

## 唯一主要结果

正式生产 Compose能够在明确 grace period内正常停止 ready core，并取得零退出和完整生命周期证据。

## 输入和前置状态

- Ticket 05 已验证的正式二进制信号行为；
- 正式生产 Compose和生产镜像；
- 可读取的 core Management状态、容器身份、退出状态和日志；
- 15 秒 stop grace period和20秒测试观察期限。

## 业务规则或待决问题

- Compose 使用应用支持的 SIGTERM，不依赖默认值表达正式契约。
- 正常停止不允许 Docker达到 grace period后发送 SIGKILL。
- 容器 ExitCode 0、RestartCount不变和日志完整必须同时成立。
- 当前验收只证明单机正式生产拓扑，不承诺零停机、连接迁移或多副本协调。
- 测试项目使用独立 Compose项目名和可回收资源，不修改正式生产行为。

## 成功后的状态变化

core容器从 running/healthy进入 draining并在15秒内 stopped(ExitCode 0)；PostgreSQL和 Caddy的测试编排状态由外部验收负责清理。

## 失败或拒绝行为

验收故意把以下任一结果视为失败：退出码137或143、停止超过grace period、RestartCount变化、缺失 draining事件、资源关闭顺序错误或依赖Docker强制终止。

## 预计修改范围

- 生产 Compose的 core停止策略；
- 独立生产生命周期验收脚本或测试；
- 容器健康、身份、退出状态、耗时和日志观察；
- 根生产生命周期 Task入口；
- 失败清理和诊断输出。

## 自动测试

1. Compose配置验证应用内部预算、grace period和测试观察期限关系。
2. 正式 core容器从ready经SIGTERM在grace period内零退出且不重启。
3. 生命周期日志顺序完整，验收检查器拒绝强制kill或异常退出夹具结果。

## 运行演示

运行一个根生产生命周期 Task，输出生产镜像身份、core容器ID/PID/RestartCount、ready时间、停止时间、总耗时、ExitCode和有序日志事件。

## 明确不做

- 重构生产启动拓扑；
- Caddy连接迁移或优雅停止设计；
- PostgreSQL容器停止语义；
- Kubernetes、滚动升级或高可用；
- 生产故障注入接口。

## 完成证据

- 生效的 Compose停止策略；
- 正式生产镜像和容器身份；
- ready到停止的时间线；
- ExitCode 0、RestartCount不变和有限耗时；
- 无SIGKILL及无测试生产污染检查。
