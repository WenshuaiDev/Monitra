# Monitra 受监督的生产进程退出闭环

Status: ready-for-agent

## Problem Statement

Monitra 已经完成第一个生产形态技术启动闭环。生产构建的 PostgreSQL、Go 核心应用和 Caddy 可以启动；浏览器能够通过 Caddy 完成类型化启动握手；PostgreSQL 控制首次启动和 readiness；运行期数据库中断可以在同一连接池和同一进程中恢复。

当前退出路径还不是一个完整、可信的生产闭环。Runtime 已经拥有 Management Listener、Application Listener 和 PostgreSQL 连接池，也已经监督两个 Listener 的运行错误，但关闭实现仍存在以下缺口：

- Context 取消只被解释成无分类的正常返回，Runtime 不能表达具体停止原因；
- readiness 会在延迟执行的 shutdown 中变为失败，但没有正式 draining 状态和可观察时间线；
- Application 与 Management 分别使用固定的 Shutdown 期限，却没有整体关闭预算；
- HTTP Shutdown 的错误只写日志，不向 Runtime 或进程结果传播；
- Shutdown 超时后没有明确的强制收束语义；
- PostgreSQL 连接池关闭没有期限，也不能报告失败；
- 正常信号、Listener 故障和清理失败缺少一致的最终结果模型；
- 当前子进程测试只证明 SIGTERM 后有限时间内零退出，没有证明完整 draining 顺序；
- 生产 Compose 没有明确声明应用停止信号和 grace period，也没有证明 Docker 不需要强制杀死核心进程。

如果这些行为继续依靠隐式 defer、被吞掉的 Shutdown 错误和 Docker 默认值，操作者无法区分正常维护停止、关键组件失败、排空超时或资源清理失败，也无法确认停止期间是否仍有新请求进入、在途请求是否完成、健康状态是否按正确顺序变化。

## Solution

扩展现有 Foundation Runtime，使它继续作为唯一的生产进程生命周期 Interface，统一接收正常停止和关键组件失败，立即撤销 readiness，按明确顺序有界排空 Listener 和释放资源，并向主进程返回准确的成功或失败结果。

Application Listener 的关闭行为由现有 HTTP Server 模块深化：开始 Shutdown 时停止接受新连接和新请求，允许已经进入 Handler 的请求在有限期限内完成；达到期限后强制关闭仍在活动的连接，并返回规范化的超时结果。

Runtime 不再使用无法返回结果的延迟 shutdown。它显式汇总监督触发原因与每个清理阶段的结果，保证原始关键失败不会被后续清理成功覆盖，也保证某一清理阶段失败后仍继续尝试收束其余资源。

正式关闭顺序固定为：进入 draining 并撤销 readiness、排空 Application、停止 Management、关闭 PostgreSQL 连接池、记录最终结果并返回。正常 SIGINT/SIGTERM 且全部清理成功时零退出；关键 Listener 失败、排空超时或资源清理失败时非零退出。

生产 Compose 明确使用 SIGTERM，并提供大于应用内部最坏关闭预算的 stop grace period。真实子进程、真实 Listener 和正式生产 Compose 共同证明成功、关键失败和超时行为。生产代码不增加测试控制、慢请求端点或通用生命周期平台。

### 当前操作者结果

- 部署操作者可以启动正式生产构建的 Go 核心进程并等待其进入 ready。
- 操作者可以通过 SIGINT、SIGTERM 或生产 Compose 正常停止该进程。
- 操作者可以观察 readiness 先失败、Application 停止接入并排空、Management 后停、PostgreSQL 连接池释放以及最终零退出。
- 操作者可以从结构化日志确认退出原因、排空结果、资源关闭顺序和最终进程结果。
- Application 或 Management Listener 意外停止时，操作者可以观察 Runtime 对其余资源执行有界清理并非零退出。
- 请求超过排空期限或资源清理失败时，进程会在有限时间内结束并返回非零结果。
- 操作者可以区分正常维护停止、关键组件失败、排空超时和资源清理失败，而不依赖 Secret、请求正文或不受控底层错误文本。

## User Stories

1. As a deployment operator, I want the ready core process to accept SIGINT, so that an interactive maintenance stop completes normally.
2. As a deployment operator, I want the ready core process to accept SIGTERM, so that Compose can stop it through the standard container signal.
3. As a deployment operator, I want readiness to fail immediately when draining begins, so that no new traffic is intentionally routed to a stopping process.
4. As a deployment operator, I want liveness to remain successful while the supervised loop and Management Listener are still running, so that draining is not mistaken for a crashed process.
5. As an API caller, I want a request that has already entered the Application Listener to receive a finite opportunity to finish, so that normal maintenance does not unnecessarily discard completed work.
6. As an API caller, I want new connections or requests rejected after Application draining begins, so that the drain window cannot be extended by fresh work.
7. As a deployment operator, I want Application draining to have a fixed deadline, so that one stuck request cannot block shutdown forever.
8. As a deployment operator, I want timed-out Application connections forcibly closed after the drain deadline, so that the process can continue bounded cleanup.
9. As a deployment operator, I want Management to remain available until Application draining finishes or times out, so that liveness and readiness remain observable during the useful part of shutdown.
10. As a deployment operator, I want Management to stop only after Application has been settled, so that the observable shutdown order is unambiguous.
11. As a database operator, I want the Foundation-owned PostgreSQL pool closed after the HTTP Listeners, so that no request can begin using a resource that is already being released.
12. As a deployment operator, I want PostgreSQL pool closure to have a finite budget, so that leaked database work cannot keep the process alive indefinitely.
13. As a deployment operator, I want a successful SIGINT or SIGTERM drain to return exit code zero, so that orchestration recognizes a normal maintenance stop.
14. As a deployment operator, I want an unexpected Application Listener stop to be a critical failure, so that a partially functioning core process cannot appear healthy.
15. As a deployment operator, I want an unexpected Management Listener stop to be a critical failure, so that loss of operational health reporting cannot remain silent.
16. As a deployment operator, I want Runtime to clean up the remaining resources after a critical Listener failure, so that failure does not leave another Listener or pool running unmanaged.
17. As a deployment operator, I want critical Listener failure to produce a nonzero exit code, so that Compose receives an accurate process outcome.
18. As a deployment operator, I want Application drain timeout to produce a nonzero exit code, so that forced request termination is not reported as a clean stop.
19. As a deployment operator, I want Management shutdown or resource cleanup failure to produce a nonzero exit code, so that incomplete cleanup remains visible.
20. As a deployment operator, I want cleanup to continue after one stage fails, so that the process makes a bounded attempt to release every resource it owns.
21. As an operator reviewing logs, I want draining start and its trigger category recorded, so that I can distinguish signals from critical component failures.
22. As an operator reviewing logs, I want Application acceptance stop, drain completion or timeout, Management stop, PostgreSQL close and final result recorded in order, so that I can reconstruct the shutdown timeline.
23. As a security operator, I want lifecycle logs limited to controlled categories, so that Secret values, request bodies and arbitrary lower-level error text are not exposed.
24. As a quality maintainer, I want real subprocess tests to send actual SIGINT and SIGTERM, so that Context-only unit tests cannot falsely claim operating-system signal support.
25. As a quality maintainer, I want critical Listener failure triggered through real Listener conditions, so that PostgreSQL interruption or a hidden HTTP control is not used as a substitute.
26. As a quality maintainer, I want a controlled Handler to prove successful and timed-out drain behavior, so that production code needs no slow-request endpoint.
27. As a quality maintainer, I want at least one acceptance to run the formal production build, so that a test-only process is not the highest lifecycle proof.
28. As a deployment operator, I want Compose stop grace period coordinated with the internal shutdown budget, so that normal stopping does not depend on Docker SIGKILL.
29. As a quality maintainer, I want the production Compose acceptance to verify exit code, elapsed time, container identity and log order, so that a superficially stopped container cannot pass.
30. As a developer, I want the new lifecycle checks included in task check, so that local and CI completion continue to use one authoritative gate.
31. As a production maintainer, I want production images free of lifecycle test handlers, fixtures and controls, so that stronger tests do not expand the production attack surface.

## Implementation Decisions

### 当前事实和调用链

The current process loads typed configuration, registers SIGINT and SIGTERM through the command layer, and calls the single Foundation Runtime Interface. Runtime starts Management first, connects the Foundation-owned PostgreSQL pool, starts Application, marks readiness successful and then supervises Context cancellation plus both Listener error channels.

Management liveness is independent of PostgreSQL. After the process has once become ready, a temporary PostgreSQL outage changes readiness to failure without terminating the process, and restoration on the existing pool changes readiness back to success. This behavior remains unchanged.

The current shutdown marks readiness unsuccessful, calls HTTP Shutdown for Application, calls HTTP Shutdown for Management, then closes PostgreSQL. The shutdown function cannot return an error, Listener shutdown errors are only logged, and PostgreSQL Close is not bounded. This implementation is extended rather than replaced.

### 进程状态与状态转换

The lifecycle uses the following semantic states:

```text
starting
├── startup succeeds → ready
├── normal signal → draining
└── startup or Management critical failure → draining(failed)

ready
├── PostgreSQL temporarily unavailable → dependency_unavailable
├── normal signal → draining
└── Listener critical failure → draining(failed)

dependency_unavailable
├── PostgreSQL restored → ready
├── normal signal → draining
└── Listener critical failure → draining(failed)

draining
→ releasing
→ stopped(success | failure)
```

- `starting`: Management is live, readiness is unsuccessful, and PostgreSQL or Application startup is incomplete.
- `ready`: both Listeners and PostgreSQL are available; liveness and readiness are successful.
- `dependency_unavailable`: PostgreSQL is temporarily unavailable after initial readiness; liveness succeeds, readiness fails, and Application remains supervised.
- `draining`: a normal signal or critical component failure has triggered irreversible shutdown; readiness has failed and Application settlement is in progress.
- `releasing`: Application is settled, Management is stopped, and owned non-listener resources are being released.
- `stopped`: all bounded cleanup attempts are complete and Runtime returns its final result.

Draining is irreversible. A late PostgreSQL recovery or readiness update cannot transition the process back to ready.

### 正常信号退出行为

- The command layer remains the owner of operating-system signal registration.
- SIGINT and SIGTERM are translated into controlled normal-stop reasons without leaking operating-system types through unrelated Foundation modules.
- Runtime acts on the first valid stop trigger and enters draining.
- Readiness becomes unsuccessful before Application draining can complete.
- Application Shutdown stops listener acceptance and waits for requests that already entered the Handler.
- Management remains live until Application completes or times out, then Management stops.
- PostgreSQL and any other already-started owned resources close after Management.
- Normal signal plus successful cleanup returns success and maps to exit code zero.
- A normal signal received during startup performs bounded cleanup of only the resources that actually started; successful cleanup is also a normal result, although the primary acceptance starts from ready.

### 关键组件失败行为

- Application Listener and Management Listener are both critical components.
- An unexpected Application Serve failure is categorized as `application_listener_failed`; readiness fails, Management remains live while available, remaining resources are cleaned up, and the final result is nonzero.
- An unexpected Management Serve failure is categorized as `management_listener_failed`; liveness is no longer promised because its Listener failed, Application is drained, remaining resources are cleaned up, and the final result is nonzero.
- Existing Management bind failure, Management startup Serve failure and Application bind failure remain nonzero startup failures with bounded cleanup of previously started resources.
- PostgreSQL runtime unavailability is not reclassified as critical component failure.
- Raw Listener errors are normalized before logging. The final process log does not emit arbitrary underlying error text.

### 排空超时和清理失败行为

The first implementation uses coordinated, finite policy values:

- Application graceful drain: 5 seconds;
- Management shutdown: 2 seconds;
- PostgreSQL and remaining resource cleanup budget: 2 seconds;
- maximum internal shutdown budget: 9 seconds;
- Compose stop grace period: 15 seconds;
- subprocess and Compose observation limit: 20 seconds.

These values are implementation parameters, not product decisions. They may be tuned using real runtime evidence as long as every value stays finite and the ordering `internal budget < Compose grace period < test observation limit` remains true.

- When Application graceful Shutdown reaches its deadline, the HTTP Server forcibly closes active connections and returns a normalized drain-timeout result.
- A failure in one cleanup stage does not skip later stages.
- Application timeout, Management shutdown failure, PostgreSQL close timeout or another owned-resource cleanup failure makes the final process result nonzero.
- The original critical trigger is retained when later cleanup also fails.
- PostgreSQL Close is wrapped in a finite Runtime-owned cleanup budget; timeout does not leave the main process waiting indefinitely.

### readiness 与 liveness 行为

| State | Liveness | Readiness |
| --- | --- | --- |
| starting | 200 | 503 |
| ready | 200 | 200 |
| PostgreSQL runtime unavailable | 200 | 503 |
| Application draining while Management is available | 200 | 503 |
| Application critical failure cleanup while Management is available | 200 | 503 |
| Management stopped or failed | unavailable | unavailable |
| process stopped | unavailable | unavailable |

Liveness means the supervised process loop and Management Listener are still functioning. It does not promise that Application still accepts work. No new management endpoint or externally controllable lifecycle state is introduced.

### 资源关闭顺序

All normal and failed termination paths share one explicit cleanup sequence:

```text
record controlled trigger
→ readiness unsuccessful
→ Application stops accepting new work
→ Application drains within its deadline
→ force-close Application on timeout
→ stop Management
→ close PostgreSQL pool within its deadline
→ aggregate final result
→ Runtime returns
→ command maps process exit code
```

Resources that never started are skipped. Resource ownership and order remain explicit; there is no scanning, dynamic registration, lifecycle participant catalogue, plugin mechanism or Service Locator.

### Runtime 与 HTTP Server 模块 seam

- `runtime.Run` remains the sole external process lifecycle Interface.
- HTTP Server deepens its existing Shutdown behavior to hide graceful waiting, deadline handling, forced connection closure and normalized results behind its current ownership seam.
- Runtime may use narrow implementation-private Listener and resource seams for its own tests. These seams are not exported as a general runtime framework.
- Production Application and Management adapters continue using the same HTTP Server implementation.
- PostgreSQL remains one Foundation-owned pool; the lifecycle work does not add a database abstraction, pool registry or second pool.

### 退出码和结构化日志语义

- Exit code `0`: SIGINT or SIGTERM and every cleanup stage succeeds.
- Exit code `1`: startup failure, critical Listener failure, drain timeout or resource cleanup failure.
- Exit code `2`: existing command usage error; unchanged.
- Docker signal exits such as 137 or 143 do not count as a clean stop.

Required structured events include:

- core process draining started, with `trigger=signal|critical_component` and a controlled reason;
- Application Listener stopped accepting requests;
- Application drain completed or timed out;
- Management Listener stopped or failed to stop;
- PostgreSQL connection pool closed or timed out;
- final core process result with `result=success|failure` and a controlled reason category.

Log event order follows actual state order. Lifecycle logs never contain Secret bodies, credential-bearing connection strings, request bodies, client-selected error text or arbitrary lower-level error strings.

### 运行与安全约束

- Only the existing Runtime and HTTP Server ownership seams are extended.
- Tests may use controlled Handler, Listener, clock or owned-resource adapters in test code at these seams.
- No production HTTP endpoint, startup-handshake parameter, environment branch, hidden command or failure switch is added for testing.
- PostgreSQL interruption is not used to simulate Listener failure.
- Subprocess tests send real operating-system signals.
- At least one acceptance runs the formal production core artifact.
- Production Compose explicitly uses SIGTERM and a stop grace period larger than the internal shutdown budget.
- Production containers retain non-root execution, read-only root filesystems, dropped capabilities and no-new-privileges.
- `task check` remains the only complete local and CI gate.

## Testing Decisions

- Good tests assert externally observable behavior at the highest practical seam: health responses, request results, connection acceptance, structured event order, elapsed time and process exit status. They do not assert private fields or merely count method calls.
- HTTP Server tests use a real TCP Listener and a controlled test Handler. One request enters before shutdown, a later request is refused, the first request either finishes after controlled release or is forcibly closed at the deadline.
- Runtime behavior tests exercise the `runtime.Run` Interface with real Listener conditions or narrow implementation-private adapters at the real production ownership seam. They observe Management health, logs, cleanup order and Runtime result.
- Application and Management unexpected-stop tests trigger actual Listener Serve failure conditions. They do not stop PostgreSQL as a substitute.
- Subprocess tests build a real core binary, start real PostgreSQL, wait for ready, send real SIGINT and SIGTERM and verify zero exit plus the complete lifecycle log.
- A controlled test-only subprocess fixture may verify nonzero mapping for a Runtime or Listener failure, but the production binary contains no test control.
- The existing startup integration tests are prior art for building the core process, sending SIGTERM, observing health and reading structured logs. They are extended to verify full lifecycle outcomes rather than replaced with a parallel harness.
- The existing production database-lifecycle script is prior art for isolated Compose projects, health polling, container identity, restart count, bounded waits and timestamped logs. A lifecycle acceptance follows the same external orchestration style.
- Production Compose acceptance starts the formal stack, records container identity and readiness, stops core through Compose, and verifies elapsed time below the grace period, exit code zero, unchanged restart count, absence of forced-kill results and ordered lifecycle logs.
- Failure acceptance covers Application drain timeout, Management cleanup failure, PostgreSQL close timeout, Application Listener failure and Management Listener failure.
- Production artifact inspection continues to reject test handlers, test commands, failure controls, Playwright, browser binaries and E2E source from production images.
- The authoritative gate contract is updated so every new lifecycle gate runs in the expected order and any child failure is propagated.

### 验收证据

1. A real HTTP request completes during graceful drain, while a new request cannot enter after drain begins.
2. A deliberately blocked request exceeds the drain deadline, is forcibly closed, and produces a normalized timeout result in finite time.
3. Runtime exposes live/not-ready while Application drains and Management is still available.
4. Runtime closes Application, then Management, then PostgreSQL, and returns success for a normal stop.
5. Application and Management unexpected Serve failure each trigger bounded cleanup and a nonzero Runtime result.
6. Application timeout, Management shutdown failure and PostgreSQL close timeout each continue remaining cleanup and produce a nonzero result.
7. Real SIGINT and SIGTERM sent to a ready core subprocess both produce exit code zero and the required log timeline.
8. The formal production Compose stack stops core within the configured grace period with exit code zero, no restart and no forced kill.
9. `task check` includes all new lifecycle checks, propagates their failures and continues to verify production-image purity.

## Out of Scope

- Redesigning the completed startup handshake, browser Bootstrap, generated OpenAPI client or Caddy startup loop.
- Treating temporary PostgreSQL runtime interruption as a process failure.
- OpenTelemetry Trace, Metric, OTLP export or a full operational metrics surface.
- Database Namespace, structured statement model, PostgreSQL/MySQL Adapter conformance or MySQL Adapter.
- Migration execution, Runtime/Migrator identity separation, migration locks, checksums, failure rerun or expected-head management.
- Complete Go operations CLI, deployment initialization or configuration migration.
- Offline Release Bundle, signing, formal release pipeline, HTTPS, backup or restore.
- Generic background tasks, workers, schedulers, queues or retry frameworks.
- Generic lifecycle participants, dynamic component registration, plugins, Service Locator or a runtime component catalogue.
- Authentication, Authorization, Session, business data scope or any business model, API, page, record or workflow.
- Timeseries database, cache, message system or object storage abstraction.
- Kubernetes, high availability, rolling deployment or zero-downtime upgrade.
- A production slow-request endpoint, failure injection endpoint, hidden control, test mode or lifecycle bypass.
- Tickets or implementation for any later frontier.

## Further Notes

- The single overall acceptance result is: a formally built core process reaches ready, then a normal SIGINT or SIGTERM makes readiness fail before Application drain, keeps Management live while useful, settles Application within a finite deadline, stops Management, releases PostgreSQL and exits zero; critical Listener failure or drain/cleanup failure performs the same bounded cleanup and exits nonzero.
- The broader minimal-runnable-foundation Spec remains the engineering boundary but is not an instruction to implement OpenTelemetry, migrations, database neutrality, offline delivery or other later capabilities in this frontier.
- The existing startup frontier is complete and must not be redesigned or decomposed again.
- The exact lifecycle event wording may be adjusted during implementation, but event categories, required fields, safe logging and observable order are part of this Spec.
- The current broad foundation Spec still has a stale tracker status and an outdated statement that the repository has no implementation. That documentation issue does not invalidate existing code or block this frontier.
- Future long-lived resources, streaming requests, uploads or background jobs will need their own real production consumers before being added to the shutdown sequence.
- Whether lifecycle timeouts later become deployment configuration remains a future operational question. Fixed, finite, tested Foundation policy is sufficient for this frontier.
