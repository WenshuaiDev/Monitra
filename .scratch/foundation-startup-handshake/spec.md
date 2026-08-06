# Monitra 第一个生产形态技术启动闭环

Status: ready-for-agent

## Problem Statement

Monitra 当前只有产品愿景、滚动推进方法、总体工程基座约束和一个尚未形成实现的 Go Module 声明，没有能够生产构建、启动或验收的应用。仓库中不存在 Go 应用入口、React SPA、OpenAPI、Caddy、Compose、PostgreSQL连接代码、测试或运行证据。

在进入第一条真实业务链路之前，Monitra 需要建立一个不包含业务语义的永久技术启动闭环，真实证明生产形态的浏览器入口、部署期配置、类型化 HTTP 契约、Go 运行时、PostgreSQL依赖、健康状态和请求追踪可以协同工作。

这个闭环不能依靠开发服务器、模拟业务、测试专用生产接口、隐藏故障开关或虚假业务页面成立，也不能把总体工程基座中的完整数据库中立性、Migration、可观测性、离线交付或运维能力一次性带入当前前沿。

## Solution

使用生产构建产物建立由浏览器、Caddy、一个 Go 核心应用和 PostgreSQL组成的正式启动拓扑。

Caddy 是唯一公开 Web 入口，负责提供 React SPA 静态资源，并把同源 `/api/v1` 请求代理到 Go Application Listener。Go 另行提供不经 Caddy 暴露的 Management Listener。PostgreSQL作为正式部署依赖启动，并由 Foundation 所有的单一有界连接池连接。

SPA 首先读取并严格校验部署期 runtime config。只有配置有效后，才创建 Foundation HttpClient，并让 OpenAPI 生成的 TypeScript 客户端通过该 HttpClient 执行启动握手。只有当握手返回的 API major 和 Release Identity 与浏览器期望兼容时，SPA 才创建 Router 并显示正常启动状态。

### 当前用户或操作者结果

- 部署操作者可以使用正式生产构建产物启动 PostgreSQL、Go 核心应用和 Caddy。
- 操作者可以通过 Management Listener 区分进程存活和应用就绪。
- 浏览器用户可以从 Caddy 加载 SPA，并在兼容闭环成立时看到明确的正常启动状态、Release Identity、API major 和本次握手的 `request_id`。
- runtime config 无效、后端不可达或版本不兼容时，浏览器显示明确失败原因，不显示看似正常的空应用外壳。

### 当前事实和目标调用链

当前仓库不存在可运行调用链。本前沿建立以下永久调用链：

```text
浏览器请求 Caddy
→ Caddy 返回生产构建的 SPA
→ Bootstrap Config Loader 读取同源部署期 runtime config
→ runtime config 完成严格校验
→ 创建 Foundation HttpClient
→ 创建 OpenAPI 生成的 TypeScript 客户端
→ 调用 /api/v1 下的启动握手
→ Caddy 同源代理到 Go Application Listener
→ Go 生成 request_id
→ Go 返回扁平启动握手响应
→ SPA 校验 API major 和 Release Identity
→ 校验成功后才创建 Router 并显示正常启动状态
```

正式运行链为：

```text
PostgreSQL 启动
→ Go Bootstrap 创建 Foundation 所有的单一连接池
→ 在有限期限内确认数据库可连接
→ 启动并监督 Application Listener 与 Management Listener
→ readiness 在当前必需依赖成立后变为成功
→ Caddy 对外提供 SPA 和 /api/v1 同源代理
```

## User Stories

1. As a deployment operator, I want to start Caddy, the Go core application, and PostgreSQL from production artifacts, so that the first Monitra deployment shape is real rather than a development-only demonstration.
2. As a browser user, I want to load the SPA from Caddy, so that the public entry point matches the intended production topology.
3. As a browser user, I want startup configuration checked before the application Router exists, so that invalid deployment settings cannot expose a misleading application shell.
4. As a deployment operator, I want one immutable SPA build to consume deployment-time public configuration, so that the same Web artifact can be reused across Deployment instances.
5. As a frontend developer, I want one Foundation HttpClient created after runtime config validation, so that API calls share one network and failure policy.
6. As a frontend developer, I want the OpenAPI-generated TypeScript client to execute through Foundation HttpClient, so that generated transport code cannot establish a second network boundary.
7. As an API consumer, I want the startup handshake under `/api/v1`, so that transport compatibility has an explicit major version.
8. As an API consumer, I want ordinary responses to contain `code`, `message`, `data`, and `request_id`, so that startup success and failure use one stable envelope.
9. As a browser user, I want startup data to include the running Release Identity and API major, so that the browser can verify compatibility before showing the application.
10. As an operator, I want every inbound request to receive a server-generated request ID, so that a browser result can be correlated with structured logs.
11. As a security operator, I want client-provided request IDs ignored as authoritative IDs, so that clients cannot choose the correlation identity recorded by Monitra.
12. As a browser user, I want a clear normal startup state, so that I know the application shell represents a compatible running backend.
13. As a browser user, I want an invalid runtime config to produce a specific visible failure, so that configuration errors are not mistaken for an empty application.
14. As a browser user, I want an unavailable backend to produce a specific visible failure, so that an infrastructure outage is not hidden behind indefinite loading.
15. As a browser user, I want an incompatible API major to be rejected visibly, so that incompatible Web and API artifacts cannot run together silently.
16. As a browser user, I want an incompatible Release Identity to be rejected visibly, so that mixed release artifacts cannot present themselves as one release.
17. As an operator, I want liveness independent of PostgreSQL, so that a temporary database outage does not cause a restart storm.
18. As an operator, I want readiness to include the mandatory PostgreSQL dependency, so that traffic is accepted only when the current deployment can operate safely.
19. As an operator, I want the Go core process to exit nonzero when PostgreSQL cannot be reached within the initial startup deadline, so that an unusable deployment cannot appear to have started.
20. As an operator, I want a running Go process to remain alive when PostgreSQL is temporarily interrupted, so that a recoverable dependency outage does not require application restart.
21. As an operator, I want the single connection pool to recover automatically after PostgreSQL returns, so that readiness becomes successful again within a finite time on the same process.
22. As a security operator, I want the Management Listener kept behind the deployment network seam, so that Caddy does not expose operational endpoints publicly.
23. As a quality maintainer, I want Playwright to exercise production-built services, so that browser evidence does not depend on Vite or other development servers.
24. As a production maintainer, I want browser failures triggered through real configuration, service state, or incompatible artifacts, so that production code needs no E2E controls.
25. As a production maintainer, I want production images free of Playwright, browser binaries, test sources, and E2E dependencies, so that test infrastructure does not expand the production attack surface.
26. As a developer, I want one authoritative `task check` result, so that local and CI acceptance use the same definition of completion.

## Implementation Decisions

### Runtime and deployment

- The production topology contains Caddy, one Go core application process, and PostgreSQL. Infrastructure containers do not change the modular-monolith decision.
- Caddy is the only public Web entry point. It serves production SPA assets and proxies same-origin `/api/v1` requests to the internal Application Listener.
- The Management Listener is separate from the Application Listener and is not proxied by Caddy.
- The current acceptance may use an explicitly non-production HTTP mode. Formal HTTPS delivery remains outside this frontier.
- Production containers run as non-root with read-only root filesystems, minimum Linux capabilities, and no-new-privileges. Privileged mode, host network, host PID, and Docker Socket mounting are prohibited.
- The current frontier introduces only the minimum operations configuration validation and browser runtime-config projection required to start the production stack. It does not complete the operations CLI.

### Browser bootstrap and runtime config

- Runtime config is strict deployment-time JSON, separate from the immutable SPA build, non-cacheable, and limited to browser-safe settings.
- Runtime config is loaded and validated before Router and Foundation HttpClient initialization.
- The Bootstrap Config Loader is the only network read that precedes Foundation HttpClient. It reads deployment configuration rather than an application API.
- Invalid or unsupported runtime config stops Bootstrap before Router creation and produces a visible configuration failure.
- Foundation HttpClient is the only frontend application network execution interface. Generated and handwritten application calls cannot use an independent `fetch` policy.
- A successful handshake is checked against the expected API major and Release Identity before Router creation.
- The normal shell is a technical startup state only. It contains no business navigation, business data, fake domain objects, or demonstration workflow.

### HTTP JSON, OpenAPI, and request correlation

- The browser-facing Interface uses HTTP JSON under `/api/v1`.
- OpenAPI is authoritative for the handshake transport contract and generates TypeScript transport types and a typed client only.
- The generated client executes through an injected Foundation HttpClient interface and cannot establish its own network mechanism.
- Go Server, Handler, response policy, routing, and Composition remain handwritten and use `net/http` at the Foundation interface.
- Ordinary success and failure responses are flat objects containing `code`, `message`, `data`, and `request_id`. Failures set `data` to null.
- Successful handshake data contains at least the current immutable Release Identity and API major.
- Each inbound request receives a new cryptographically safe server-generated `request_id`. A client cannot select the authoritative value.
- The same request ID is carried in request context, response header, response body, and the corresponding structured request log. It remains distinct from a future Trace ID.
- Production logs use injected `slog` JSON loggers writing to stdout. There is no mutable application-global logger.

### PostgreSQL, startup, and health state

- PostgreSQL is a real dependency in the formal deployment topology but owns no business table, Business Namespace, or business Migration in this frontier.
- Foundation creates and owns exactly one bounded PostgreSQL connection pool.
- Initial connection has an explicit finite deadline. If PostgreSQL cannot be connected within that deadline, readiness never succeeds, resources are cleaned up within bounds, and the Go core process exits nonzero.
- The process cannot remain indefinitely alive in an initial not-ready state after the startup deadline.
- Liveness reports supervised process-loop health and does not depend on PostgreSQL.
- Readiness remains false during startup and includes the mandatory PostgreSQL connection state.
- After the process has entered ready at least once, a temporary PostgreSQL outage does not stop or restart the Go process. Liveness remains successful and readiness becomes unsuccessful within a finite time.
- The existing single connection pool performs controlled automatic connection recovery. It does not transparently retry submitted business statements.
- When PostgreSQL returns, readiness becomes successful again within a finite time on the same Go process and container instance.
- If PostgreSQL remains unavailable after a previously successful startup, the process remains alive and live while readiness remains unsuccessful.

### Failure and rejection behavior

- Missing, unknown, malformed, or unsupported runtime config is rejected before HttpClient and Router initialization.
- Network failure, timeout, invalid JSON, or an unusable handshake response is presented as backend unavailable and does not render the normal shell.
- An API-major mismatch is presented as an API compatibility failure.
- A Release Identity mismatch is presented as a release compatibility failure.
- Compatibility failures are triggered in acceptance with real configuration or mismatched production artifacts, never through hidden production controls.
- A client-supplied request ID is ignored as the authoritative correlation identity.
- Initial PostgreSQL failure produces a nonzero core-process exit. Runtime PostgreSQL failure produces liveness-success/readiness-failure and permits automatic recovery without process restart.

### Explicit composition and module seams

- Bootstrap, runtime configuration, the Foundation connection pool, Foundation HttpClient, request-correlation policy, the Application Listener, and the Management Listener are explicitly assembled.
- There is no scan, reflection, `init()` registration, plugin registry, module manifest, or Service Locator.
- The primary acceptance seam is the production-built deployment observed by a real browser. Lower tests support this seam but do not replace it.

## Testing Decisions

- Good tests assert externally observable behavior at the highest practical seam. They do not assert private package layout, Router internals, SQL text, logger implementation details, or third-party framework behavior.
- The primary acceptance test runs a real browser against production-built Caddy, Go, and PostgreSQL services.
- Root Playwright acceptance proves the normal startup state and visible failures for invalid runtime config, unavailable backend, incompatible API major, and incompatible Release Identity.
- Failure scenarios use public browser behavior, real service state, deployment configuration, and real production artifact combinations. Production code contains no E2E branches, hidden endpoints, failure controls, or authentication bypasses.
- Go contract tests send requests through the real Handler seam and prove HTTP statuses, flat response fields, Release Identity, API major, safe failures, and request-ID correlation.
- Request-correlation tests prove that the response header, response body, and one structured log record contain the same server-generated ID, and that a client-provided value is not reused.
- React tests render the Bootstrap behavior and prove initialization order, normal output, and each visible failure state without asserting private Router mechanics.
- Real PostgreSQL and real subprocess tests prove initial success, bounded nonzero failure, liveness/readiness transitions, runtime outage, automatic recovery, finite readiness restoration, and unchanged process identity.
- Production-stack evidence repeats the database lifecycle test against the actual Compose topology and records process/container identity across the outage.
- Production-image inspection proves that Playwright, browser binaries, E2E source, test-only controls, and development servers are absent.
- Generated OpenAPI artifacts are reproducible and checked for drift.
- `task check` is the sole authoritative full gate locally and in CI. A partial green test cannot independently establish frontier completion.
- There is no existing code or test prior art in this repository. These tests intentionally establish the first permanent high-level seams and should be extended rather than paralleled later.

## Out of Scope

- Any business Namespace, table, Migration, model, page, route, record, fixture, workflow, authentication, authorization, Session, or business data scope.
- A complete database-neutral structured-statement model, Business Module persistence interface, PostgreSQL/MySQL shared Adapter conformance, or MySQL Adapter.
- Multiple Namespaces, cross-Namespace transactions, multiple physical Database Sources, database high availability, replication, or failover.
- Complete logical Migration support, Runtime/Migrator identity separation, Migration locking, checksums, failed-run recovery, or expected-head management.
- Transparent retry of submitted business statements or transactions.
- Complete OpenTelemetry receiver acceptance, a default Collector, or a complete operational metrics surface.
- The complete Go operations CLI, offline Release Bundle, Release Bundle signing, installation workflow, or formal release pipeline.
- Formal HTTPS delivery, public ACME, backup, restore, or production-data recovery.
- A complete UI component library, business navigation, Dashboard, sample page, or example business data.
- A general server-state cache, SSR, production Node.js server, Service Worker, PWA, or browser offline synchronization.
- Timeseries database, cache, message system, object storage, or empty abstractions for those systems.
- Tickets or implementation for any subsequent frontier.

## Further Notes

- This Spec is a small current-frontier slice of the confirmed minimal-runnable-foundation Spec. The broader Spec remains the design boundary but is not an instruction to implement all foundation capabilities now.
- The single overall acceptance result is: a real browser loads the production SPA from Caddy, validates deployment-time runtime config before Router and HttpClient creation, calls the versioned startup handshake through Foundation HttpClient and an OpenAPI-generated client, and shows a truthful success or failure state correlated with the Go process and PostgreSQL readiness.
- Exact handshake path naming beneath `/api/v1`, configuration field names, generator selection, finite timeout values, pool sizing, and the long-term Release Identity naming convention are local implementation parameters. They must be explicit and tested but do not block this frontier.
- The first permanent proof is technical. It must not be made more visually impressive by introducing fake business concepts.
- After this frontier completes, its real runtime evidence must be reviewed before selecting the next frontier.
