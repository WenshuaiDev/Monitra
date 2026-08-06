# 06 — Playwright 验证浏览器启动成功与明确失败

Type: evidence
Status: resolved
Blocked by: 05 — 生产构建栈通过 Caddy 形成同源启动闭环

**What to build:** 使用根目录 Playwright 和生产构建栈，重复证明浏览器正常启动以及 runtime config 无效、后端不可用、API major 不兼容和 Release Identity 不兼容时的明确失败状态。

## Acceptance criteria

- [x] Playwright 从 Caddy公开入口验证正常状态、Release Identity、API major 和非空 request_id。
- [x] 无效 runtime config 显示配置失败，不显示正常外壳。
- [x] Go 后端不可用显示后端失败，不显示正常外壳。
- [x] API major 与 Release Identity 不兼容分别显示准确失败状态。
- [x] 所有失败仅由真实配置、服务状态或不兼容生产产物触发，生产代码不存在 E2E 控制。

## 它让系统向前移动什么

当前技术闭环从“可以运行”提升为“真实浏览器能够重复证明成功和主要失败行为”。

## 当前事实和调用链

Ticket 05 已形成正式拓扑，但尚无根目录浏览器验收，也未证明失败时不会显示正常外壳。

Playwright 只通过以下链路观察系统：

```text
真实浏览器
→ Caddy公开入口
→ runtime config
→ /api/v1
→ 真实生产服务
```

## 唯一主要结果

根目录 Playwright 能够重复证明启动成功和全部规定的用户可见启动失败状态。

## 输入和前置状态

- Ticket 05 的生产栈；
- 有效和无效的部署期 runtime config；
- 可组合的兼容与不兼容生产构建产物；
- 可从外部停止的 Go 服务。

## 业务规则或待决问题

- 浏览器不得调用 Management Listener作为应用行为。
- 失败通过真实配置、服务停止或不兼容构建产物触发。
- 不得向生产代码添加 E2E 环境分支、测试端点、隐藏控制或故障开关。
- 每个失败断言同时证明正常外壳没有出现。

## 成功后的状态变化

Playwright成功验证页面显示正常状态、Release Identity、API major 和本次服务端握手的 request_id。

## 失败或拒绝行为

分别运行无效 runtime config、Go 后端不可用、API major 不兼容和 Release Identity 不兼容场景；每种情况必须显示准确失败原因，并确认正常启动外壳不存在。

## 预计修改范围

- 根浏览器验收工程；
- E2E生产栈编排和真实场景配置；
- 测试日志、截图、超时和失败清理；
- 根 Task E2E入口。

## 自动测试

1. 生产栈成功启动页面。
2. runtime config 无效和后端不可用页面。
3. API major 与 Release Identity 不兼容页面。

## 运行演示

运行根 Task 的生产 E2E 命令，生成成功与失败场景报告，并证明重复运行后不会遗留测试服务。

## 明确不做

- 开发服务器 E2E；
- 直接调用内部 Management Listener替代用户行为；
- E2E 专用生产镜像；
- 隐藏测试接口、认证绕过或生产故障注入。

## 完成证据

- Playwright报告；
- 每种状态的浏览器断言；
- 对应生产栈日志；
- 无残留测试服务的清理证据。
