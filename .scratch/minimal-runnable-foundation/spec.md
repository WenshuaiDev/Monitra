# Monitra 最小可运行工程基座

Status: ready-for-agent

## Problem Statement

Monitra 已经确认需要以同一套核心产品支持感知中心统一平台、业主级私有化部署和项目级私有化部署，并能够在离线环境中独立安装、升级和长期运行。当前仓库只有产品愿景、滚动推进方法和一个尚未形成实现的 Go Module 声明，没有应用代码、Web 应用、数据库实现、部署编排、测试或运行证据。

如果直接进入业务功能开发，语言和框架边界、进程生命周期、HTTP 契约、配置与 Secret、数据库可替换性、迁移权限、离线交付和质量门禁会被各个业务模块分别决定。这样会使三种部署形态逐渐分叉，也会让未来切换关系型数据库、加入时序数据库、缓存、消息系统或对象存储时被迫修改业务代码。

因此，Monitra 需要先建立一个不包含业务模型的最小可运行工程基座。这个基座必须形成真实的浏览器到运行服务再到正式关系型数据库的闭环，并把稳定的技术边界、显式 Composition、运行诊断、迁移和测试机制建立起来，但不得通过虚假业务页面、示例业务对象或为 E2E 特制的生产代码证明自身可用。

## Solution

建立一个可复制、显式装配、数据库可替换、可离线交付的模块化单体工程基座。

正式运行形态由浏览器、Caddy、一个 Go 核心应用进程和 PostgreSQL 组成。Caddy 提供 React SPA 静态资源并将同源版本化 API 代理到 Go 应用；Go 应用提供 Application Listener 和不经 Caddy 暴露的内部 Management Listener；PostgreSQL 作为第一版默认关系型数据库。独立的 Go 运维 CLI 负责部署前检查、配置投影、迁移和 Compose 生命周期操作。

工程采用单仓库、单根 Go Module 和显式 Composition。Foundation 只提供稳定通用的技术 Interface；Business Module 在未来出现时拥有自己的窄 Interface 和契约；Technical Adapter 实现这些 Interface；项目 Composition 逐项完成装配。当前基座只交付技术闭环和空应用外壳，不建立任何业务对象。

基座以生产构建运行栈作为最高层验收接缝：浏览器加载 Caddy 提供的 SPA，SPA 读取部署期运行配置，经 Foundation HttpClient 调用 Go 的版本化 HTTP JSON Interface，Go 返回 Release Identity、API 主版本和 `request_id`，数据库、健康检查、日志、指标、链路追踪和版本兼容状态均可被验证。数据库可替换性由同一套逻辑迁移、查询、事务和错误语义在真实 PostgreSQL 与 MySQL Adapter 上的共享一致性测试补充证明。

## User Stories

1. As a Monitra developer, I want one authoritative repository structure, so that backend, frontend, contracts, deployment assets, operations tooling, and tests evolve together.
2. As a Monitra developer, I want one root Go Module, so that Go dependency and build boundaries remain simple during the modular-monolith phase.
3. As a Monitra developer, I want production Go code kept behind repository-internal boundaries, so that the copied Foundation does not accidentally become a public remote library contract.
4. As a Foundation maintainer, I want Foundation to expose only stable generic technical Interfaces, so that it does not absorb business semantics.
5. As a Business Module author, I want to own the narrow Interfaces and contracts required by my module, so that contracts follow their semantic owner.
6. As a project integrator, I want Composition to wire every module and adapter explicitly, so that runtime behavior can be understood without scanning, reflection, registration side effects, or a Service Locator.
7. As a deployment operator, I want the same release artifacts to support all three confirmed Deployment profiles, so that center, owner-private, and project-private installations do not become separate products.
8. As a private-deployment operator, I want Monitra to install and run without an online dependency on the center platform, so that isolated intranet deployments remain functional.
9. As a release operator, I want a self-contained offline Release Bundle, so that I can transfer, verify, install, and operate an exact release without internet access.
10. As a release operator, I want every image and artifact tied to one immutable Release Identity, so that I can determine exactly what version is running.
11. As a release operator, I want the Release Bundle cryptographically signed and verified by the operations CLI, so that altered or untrusted bundles are rejected.
12. As an operator, I want a self-contained Go operations CLI, so that the target server does not require Go, Node.js, pnpm, Python, Git, or Task.
13. As an operator, I want the operations CLI to validate the installed Docker Engine and Compose plugin, so that unsupported host tooling fails before changing the deployment.
14. As an operator, I want the operations CLI to invoke version-checked Docker Compose files, so that Monitra does not maintain a second orchestration engine.
15. As a developer, I want Task to be the sole public source and CI command interface, so that local and CI workflows use the same entry points.
16. As a developer, I want Go, Node.js, pnpm, Task, and generators pinned, so that builds do not drift silently.
17. As a browser user, I want the SPA to detect an incompatible or unavailable backend during startup, so that the application fails visibly instead of rendering a misleading shell.
18. As a frontend developer, I want strict TypeScript and one pnpm lockfile, so that transport and build errors are detected consistently.
19. As a frontend developer, I want Vite used only for development and production builds, so that production runtime does not depend on a Node.js server.
20. As a frontend developer, I want React Router Data Mode limited to routing and navigation lifecycle responsibilities, so that routing does not silently become the universal server-state cache.
21. As a frontend developer, I want one Foundation HttpClient, so that generated clients and handwritten technical calls share network, authentication, request correlation, and error handling policy.
22. As a frontend developer, I want runtime configuration loaded before Router and HttpClient initialization, so that one static build works across Deployment instances.
23. As a frontend developer, I want no Service Worker or PWA cache in the first version, so that offline installation is not confused with offline browser synchronization.
24. As an API consumer, I want a versioned HTTP JSON Interface described by OpenAPI, so that transport compatibility is explicit.
25. As an API consumer, I want every ordinary success and failure response to contain `code`, `message`, `data`, and `request_id`, so that response handling is stable.
26. As an API consumer, I want validation failures to expose optional top-level field errors, so that invalid fields can be presented without inventing a second error envelope.
27. As an API consumer, I want real HTTP status codes alongside stable semantic result codes, so that HTTP infrastructure and application clients both receive meaningful outcomes.
28. As a backend developer, I want Foundation HTTP Interfaces based on `net/http`, so that Business Modules do not depend on third-party Context, Handler, error, or Middleware types.
29. As a backend developer, I want Router replacement justified by measured request behavior and demonstrated feature gaps, so that framework churn does not drive the architecture.
30. As a frontend developer, I want OpenAPI to generate TypeScript transport types and a typed client only, so that generated code stays at the HTTP transport edge.
31. As a backend developer, I want contract tests to prove the Go implementation conforms to OpenAPI, so that server generation is not required for correctness.
32. As an operator, I want Caddy to be the only public Web entry point, so that SPA delivery, same-origin API routing, and forwarded-request trust have one boundary.
33. As a security operator, I want Caddy to overwrite forwarded headers and Go to trust them only from the configured immediate peer, so that clients cannot forge normalized source information.
34. As a security operator, I want non-root containers, read-only root filesystems, minimum capabilities, and no-new-privileges, so that a compromised component has a smaller host impact.
35. As a security operator, I want privileged mode, host network/PID, and Docker Socket access prohibited, so that the default deployment does not grant unnecessary host control.
36. As an operator, I want an explicit non-production HTTP mode and an operator-supplied HTTPS mode, so that formal deployments cannot silently downgrade transport security.
37. As an operator, I want typed configuration loaded once during Bootstrap and injected by Composition, so that modules do not read environment variables or files independently.
38. As an operator, I want strict YAML configuration with a declared Schema version, so that unknown, duplicate, or unsupported configuration fails before startup.
39. As an operator, I want configuration upgrades performed explicitly rather than rewritten during application startup, so that deployment changes remain reviewable.
40. As an operator, I want Secret bodies supplied through files and referenced by configuration, so that credentials are not embedded in ordinary settings.
41. As an operator, I want frontend runtime JSON projected from validated backend configuration, so that the browser receives only required public settings.
42. As an operator, I want Deployment profile and technical instance identity fixed at initialization, so that changing deployment meaning cannot happen as an ordinary runtime toggle.
43. As an operator, I want a Runtime supervisor for every long-lived component, so that unmanaged goroutines and silent component exits cannot leave a partially running process.
44. As an operator, I want bounded graceful shutdown and nonzero exit on critical component failure, so that Compose receives an accurate process outcome.
45. As an operator, I want separate liveness and readiness behavior, so that process death, startup, dependency outage, incompatibility, and draining are distinguishable.
46. As an operator, I want liveness independent of external services, so that a temporary database outage does not create a restart storm.
47. As an operator, I want readiness to include mandatory dependency and schema compatibility checks, so that traffic is accepted only when the release can operate safely.
48. As an operator, I want a running process to recover after a temporary database outage, so that a database restart does not require restarting the entire deployment.
49. As an operator, I want startup to fail after a bounded database connection deadline, so that an unusable deployment does not appear to start successfully.
50. As an operator, I want production logs emitted as structured JSON to stdout, so that deployment tooling can collect and retain them consistently.
51. As an operator, I want each inbound request assigned a server-generated cryptographically safe `request_id`, so that responses, context, and logs can be correlated without trusting the client.
52. As an operator, I want OpenTelemetry Trace and Metric export to be bounded and non-blocking, so that an unavailable telemetry destination does not stop business traffic.
53. As an operator, I want health, version, and metric endpoints on a separate internal Management Listener, so that operational interfaces are not accidentally exposed through Caddy.
54. As a security operator, I want nonzero HTTP timeouts and bounded request headers and bodies, so that the default server does not accept unbounded resource consumption.
55. As a database maintainer, I want PostgreSQL to be the default relational runtime without a TimescaleDB dependency, so that the initial deployment has one ordinary relational prerequisite.
56. As a Business Module author, I want structured typed statements and logical migrations instead of vendor SQL strings, so that business persistence remains independent of PostgreSQL and MySQL syntax.
57. As a Business Module author, I want database drivers, pools, vendor errors, and physical names hidden behind Foundation and Adapter boundaries, so that switching the relational database does not change my module.
58. As a database maintainer, I want real PostgreSQL and MySQL Adapters verified by one conformance suite, so that database neutrality is demonstrated rather than claimed.
59. As an operator, I want exactly one physical Database Source and one bounded connection pool in the first deployment, so that resource ownership and transaction behavior remain explicit.
60. As a Business Module author, I want to declare each required logical Namespace independently, so that Foundation does not decide namespace names, counts, or ownership.
61. As a project integrator, I want Composition to bind every logical Namespace to one physical Namespace explicitly, so that PostgreSQL Schema and MySQL Database differences remain adapter concerns.
62. As a database maintainer, I want queries and migrations to use the same injected Namespace reference, so that schema changes and runtime access cannot drift to different physical locations.
63. As a Business Module author, I want to coordinate explicitly across injected Namespaces within one physical transaction, so that cross-namespace behavior stays visible in the owning use case.
64. As a database maintainer, I want cross-Namespace foreign keys, views, triggers, procedures, cascades, and owner-spanning migrations prohibited, so that namespace ownership remains enforceable outside a specific database product.
65. As a Business Module author, I want to own transaction boundaries and retry decisions, so that Foundation cannot replay side effects without business knowledge.
66. As a Business Module author, I want only stable normalized database error categories, so that business behavior never branches on vendor codes, physical constraint names, or error text.
67. As a Business Module author, I want no generic Entity, CRUD Repository, soft-delete, audit-column, or optimistic-concurrency policy imposed by Foundation, so that technical convenience does not invent business rules.
68. As a database operator, I want migrations executed only by the Go operations CLI under a separate Migrator identity, so that the application Runtime identity has no DDL permission.
69. As a database operator, I want the core application to check schema compatibility without automatically migrating, so that startup cannot change persistent structure unexpectedly.
70. As a database operator, I want each logical Namespace to maintain its own immutable migration history and checksums, so that migration ownership follows Namespace ownership.
71. As a database operator, I want one exclusive migration lock for the physical Database Source, so that two CLI instances cannot race across Namespace migrations.
72. As a database operator, I want failed migrations to stop and report their actual state without automatic rollback or continuation, so that PostgreSQL and MySQL DDL differences are not hidden behind a false atomicity promise.
73. As a database operator, I want a failed migration to be safely rerunnable after correction, so that recovery does not depend on a universal Down migration.
74. As an operator, I want every Release to declare the exact expected Migration Head for each assembled Namespace, so that both older and newer incompatible database structures prevent startup.
75. As a data owner, I want PostgreSQL data stored in a labeled Docker named volume retained by normal stop, upgrade, and uninstall operations, so that routine lifecycle commands do not silently destroy state.
76. As a data owner, I want destructive data removal to require explicit target verification and confirmation, so that instance data cannot be deleted by an ambiguous command.
77. As a Go developer, I want standard-library tests, hand-written fakes, `httptest`, real subprocesses, and real database instances, so that tests observe behavior without a general mocking framework.
78. As a frontend developer, I want Vitest and React Testing Library for component and module behavior, so that frontend tests remain focused on rendered outcomes and user interaction.
79. As a quality maintainer, I want Playwright E2E isolated at repository root and executed against production-built containers, so that browser acceptance does not depend on development servers or production test hooks.
80. As a production maintainer, I want E2E tests limited to the browser, public HTTP behavior, and the formal operations CLI, so that production code contains no hidden test endpoints, controls, branches, or authentication bypasses.
81. As a quality maintainer, I want `task check` to be the sole authoritative full gate locally and in CI, so that completion has one repeatable definition.
82. As a quality maintainer, I want the startup handshake, incompatible-version path, dependency-outage path, and database adapter conformance covered automatically, so that the foundation proves both success and failure behavior.

## Implementation Decisions

### Runtime and deployment

- The release target is Linux x86_64 on a single server or virtual machine. Development is supported on macOS arm64 and Linux x86_64. Windows and Linux ARM are not supported release targets.
- Runtime packaging uses OCI images and Docker Compose-compatible orchestration. Docker Engine and the Compose plugin are host prerequisites; Monitra validates but does not install or upgrade them.
- The first version has one core Go application process. Infrastructure containers do not change the modular-monolith decision. Process splitting requires a demonstrated isolation, dependency, or scale need.
- Center, owner-private, and project-private Deployment profiles use the same core images, release identity, migrations, and code. Profile differences are expressed through initialization settings and explicit Composition, not separate products or scattered conditionals.
- The first upgrade model uses planned downtime. Zero-downtime and rolling upgrades are not promised.
- Caddy is the public edge container. It serves production SPA assets and proxies the same-origin versioned API to the internal Go Application Listener.
- The Go process also exposes a separate internal Management Listener for liveness, readiness, release/version information, and metrics. Caddy does not proxy this listener.
- Non-production deployments may explicitly use HTTP. Formal deployments use operator-provided certificate and key files for HTTPS. There is no public ACME dependency and no silent HTTPS-to-HTTP downgrade.
- Containers run as non-root with read-only root filesystems, minimum Linux capabilities, and no-new-privileges. Privileged mode, host network, host PID, and Docker Socket mounting are prohibited unless a future exception is separately designed and approved.

### Language, technology stack, and repository structure

- Backend and operations tooling use Go. The browser application uses React with strict TypeScript.
- Vite is used for frontend development and builds; pnpm is the only JavaScript package manager and one lockfile governs the repository.
- The repository is a monorepo with one root Go Module. Production Go packages remain repository-internal; there is no root public package tree and no nested Foundation Go Module.
- The repository has explicit areas for the core application command, the operations CLI command, Go Foundation, Technical Adapters, project Composition, future Business Modules, the React application, OpenAPI composition, root-level E2E, deployment assets, documentation, local tracker material, and root build manifests.
- Foundation is reused across projects by copying and owning its source. It is not a centrally versioned runtime dependency.
- Contract follows its semantic owner. There is no global Contracts junk layer, universal Module interface, global module manifest, automatic scan, plugin registry, reflection-based discovery, `init()` registration, or Service Locator.
- Foundation provides stable generic technical Interfaces. A future Business Module may own narrow Interfaces and contract fragments. Technical Adapters implement the required Interfaces. Project Composition wires each dependency explicitly.
- No Business Module is created merely to demonstrate the foundation. Business module directories appear only when real business behavior is designed.

### Source, CI, operations CLI, and release bundle

- Task is the sole public source-development and CI command interface. A Makefile is not added as a second public interface.
- Formal target servers do not require source code, Git, Go, Node.js, pnpm, Task, or Python.
- The self-contained Go operations CLI performs deployment prechecks, configuration projection, migration execution, release verification, and formal lifecycle operations by invoking a version-checked `docker compose` CLI with versioned Compose files. It does not reimplement Compose through the Docker SDK.
- Go, Node.js, pnpm, Task, and code generators are exactly pinned. Docker Engine and Compose use a documented validated compatibility range. This Spec does not infer exact version numbers from unimplemented or incidental repository files.
- One immutable Release Identity binds the Web image, application image, migrations, configuration compatibility, offline bundle, and documentation. Images never use `latest` as a release contract.
- The offline Release Bundle contains OCI image archives, versioned Compose definitions, manifest and digests, configuration templates, the operations CLI, operator documentation, and a dependency-license list. It contains no Secret bodies or real deployment data.
- Formal Release Bundles are cryptographically signed. The Go operations CLI verifies the signature against a trusted public key. Unsigned bundles are accepted only through an explicit non-production path.

### Configuration and Secret handling

- Backend configuration is strict constrained YAML. Unknown and duplicate fields, YAML anchors, aliases, custom tags, and unsupported syntax are rejected.
- Configuration has an explicit Schema version. Upgrade precheck detects incompatible versions; configuration migration is an explicit operator action; application startup never rewrites the configuration file.
- Bootstrap loads and validates configuration once, produces typed immutable settings, and passes settings through explicit Composition. Foundation, Adapters, and future Business Modules do not read environment variables or configuration files independently.
- Secret bodies are provided through files. Ordinary configuration and environment variables contain only Secret paths or references. The first version rotates Secrets through a controlled restart.
- The Deployment profile and immutable technical instance identity are set at instance initialization. Changing a Deployment profile is not an ordinary configuration edit and requires a future migration design.
- The instance identity scopes Compose project naming, labels, and managed directories. Destructive operations resolve targets through the release manifest and exact labels rather than broad path or name matching.
- Frontend runtime configuration is strict, non-cacheable JSON projected by the Go operations CLI from validated YAML. It contains only browser-safe settings and is validated before Router or HttpClient initialization.
- One production frontend build is reused across deployments. Hashed static assets may be cached immutably; the HTML entry document and runtime configuration are revalidated.
- The first version is same-origin and does not enable CORS. Split frontend/backend deployment requires a future design covering allowed origins, credentials, cookies, CSRF, and authentication.
- There is no dynamic feature-flag system. Available capabilities arise from explicit Composition and startup configuration; the browser receives actual backend capability availability rather than inferring it from scattered Deployment-profile checks.

### Go runtime, HTTP, logging, health, and observability

- Foundation Runtime supervises every long-lived component. A critical component exit causes bounded shutdown and a nonzero process result. Unmanaged production goroutines are prohibited.
- Graceful shutdown has explicit finite drain deadlines and stops accepting new work before waiting for in-flight work.
- Liveness reports process-loop health without requiring external dependencies. Readiness remains false during startup and draining and checks mandatory dependencies and schema compatibility.
- Initial database connection has a bounded startup deadline. Failure to connect or a schema incompatibility prevents the application from becoming ready and terminates startup.
- After successful startup, a temporary database outage does not fail liveness or immediately terminate the process. Readiness fails, database-dependent requests return a stable temporary-unavailability result, and the single connection pool reconnects with controlled backoff.
- Foundation's public HTTP Interface uses `net/http`. A compatible Router or framework may be used internally only when a complete request benchmark and real feature gap justify it. Third-party Context, Handler, error, and Middleware types never enter Business Module Interfaces.
- The HTTP server uses explicit nonzero safe timeouts and header/body bounds. A future upload or streaming endpoint must explicitly relax only the necessary limits.
- Caddy overwrites forwarded headers. Go uses normalized proxy information only when the immediate peer matches the explicitly trusted Caddy boundary.
- Foundation creates a new cryptographically safe `request_id` for every inbound request. Clients cannot choose it. It is propagated through request context, response header, response body, and structured logs. It remains distinct from OpenTelemetry Trace ID.
- Production logs use `log/slog` JSON output to stdout. Loggers are injected; there is no application-global mutable logger. Deployment tooling owns log collection and retention.
- OpenTelemetry provides Trace and Metric signals. OpenTelemetry types do not enter Business Module Interfaces, and slog remains the logging API.
- An OpenTelemetry Collector is not part of the default runtime. Acceptance must prove real OTLP export to an actual receiver. Exporter outage, queue pressure, and shutdown are bounded and do not block application traffic indefinitely.

### HTTP JSON and OpenAPI

- The public browser-facing API uses HTTP JSON. GraphQL and browser-facing gRPC are not introduced.
- API major version is carried in the URL beginning with `/api/v1`. Compatible changes stay within the major version; breaking changes require a new major version. The first release does not promise simultaneous support for multiple majors.
- OpenAPI is the authority for the HTTP transport contract. Foundation owns a common transport fragment; future Business Modules own their paths, schemas, and semantic result codes; API Composition references fragments explicitly and publishes one complete document.
- The first version generates only TypeScript transport types and a typed client. Generated calls execute through the Foundation HttpClient and cannot establish their own network, authentication, correlation, or error-handling mechanisms.
- Go Server, Handler, use case, domain type, persistence type, route, and Composition code are handwritten. Go contract tests prove conformance to the composed OpenAPI document. Go transport DTO generation may be reconsidered only after real mechanical repetition appears and must remain limited to the HTTP transport edge.
- Ordinary success and failure responses are flat objects containing `code`, `message`, `data`, and `request_id`. Validation failures may additionally contain top-level `field_errors`.
- Failure responses always set `data` to null. `field_errors` is absent from success responses and appears only for validation failure. There is no outer `error` object and no RFC 9457 response shape.
- Responses use real HTTP status codes. Foundation owns the common response shape, safe fallback, request correlation, and serialization. A Business Module owns its semantic result codes, business data, and HTTP mapping.
- Semantic result codes are stable uppercase strings prefixed by their owner. They are part of the API contract. There is no runtime global code registry; composed OpenAPI validation checks uniqueness.

### React application foundation

- The primary UI is a client-rendered React SPA with no SSR or production Node.js server.
- React Router Data Mode owns route matching, navigation lifecycle, route Loader/Action, Pending behavior, and Error Boundaries. It is not the default universal server-state cache.
- The minimal foundation uses bootstrap root loading and local component state only. It does not introduce TanStack Query, Redux, Zustand, or another global/server-state library. A future choice must separate Router lifecycle data, page state, HttpClient policy, and shared server cache ownership.
- Foundation HttpClient is the only frontend network execution boundary. Business code and generated clients do not scatter direct `fetch` calls.
- The minimum permanent frontend/backend closure is a startup handshake and empty application shell. It proves runtime configuration, HttpClient, Caddy proxying, Go HTTP behavior, API compatibility, Release Identity, and request correlation. It does not present fake business pages or data.
- The Web Foundation contains only technical capabilities. Future React Business Modules own their pages, routes, interactions, and narrow Interfaces. Web Composition assembles them explicitly.
- Frontend and backend module boundaries are independent. They connect through the composed OpenAPI contract and do not require mirrored directory or module structures.
- The minimal foundation does not include a full UI component library. It includes only the minimum styles, design tokens, responsive behavior, and accessibility needed by the empty shell and visible failure states.
- No Service Worker, PWA installation, or browser offline-data cache is included.

### Relational persistence foundation

- The default runtime uses PostgreSQL without requiring TimescaleDB or another PostgreSQL extension.
- The release runtime contains one physical Database Source and one Foundation-owned bounded connection pool. Future Business Modules and Adapters cannot construct independent pools.
- The connection pool has explicit limits, connection lifetimes, wait behavior, readiness behavior, and metrics. Exact numeric defaults are implementation parameters constrained by finite safe behavior, not new product choices in this Spec.
- A real MySQL Adapter and shared database conformance tests are part of foundation acceptance, but MySQL does not run in the default Monitra deployment.
- The Business Module persistence boundary is database-neutral. It exposes no `database/sql`, driver objects, placeholder syntax, executable vendor SQL strings, vendor errors, or physical database names.
- Foundation provides typed structured statements, projections, codecs, executor and transaction primitives, logical migration execution, Namespace references, health, and normalized results. Adapters compile statements, map dialects and types, bind runtime values, execute operations, and normalize errors.
- Foundation does not provide a generic Entity model, universal CRUD Repository, default optimistic concurrency, soft deletion, or audit columns. These are business decisions when a real use case requires them.
- There is exactly one physical Source in the first deployment. Multiple independent Sources, PostgreSQL and MySQL coexistence, read/write splitting, sharding, distributed transactions, and automatic failover are not included.
- One Source may contain multiple logical Namespaces. PostgreSQL maps them to Schemas in one database; MySQL maps them to Databases/Schemas on one server.
- Foundation owns only Namespace identity/reference primitives, validation and parsing, migration execution, and database-dialect mapping. It does not predeclare a namespace catalogue or decide names, counts, ownership, or module allocation.
- A future Business Module declares each required logical Namespace and migration set. Composition binds each declaration explicitly to the Source and a physical Namespace. Multiple Namespace requirements are injected individually and never inferred or discovered.
- Structured queries and migrations use the same injected Namespace reference. Hard-coded PostgreSQL Schema, MySQL Database, or other physical qualified names are prohibited.
- Cross-Namespace database foreign keys, views, materialized views, triggers, procedures, implicit cascades, and migrations owned by another module are prohibited.
- A Business Module application use case may explicitly coordinate multiple injected Namespaces within the same physical transaction. The use case owns the transaction boundary. Foundation executes begin, commit, rollback, and error normalization without hidden nested transactions, savepoints, or whole-callback retries.
- Foundation never transparently retries a submitted business statement, transaction, or migration. It may restore pool connectivity. Retry choice belongs to the semantic owner and must account for idempotency and external effects.
- Adapters map database failures to a finite stable taxonomy such as unavailable, timeout, conflict, constraint violation, canceled, and unknown. Driver codes, vendor error strings, and physical constraint names remain internal diagnostic details. Business-specific constraint handling uses owner-defined logical identities.

### Migration and database identity

- The Go operations CLI is the only migration executor. It connects with an independent Migrator Secret and role that has required DDL permissions.
- The core application uses a separate Runtime Secret and role with minimum DML permissions. It is not a superuser or schema owner and cannot perform DDL or automatic migrations.
- Deployment maintenance follows precheck, migration, application startup, and readiness verification. Application startup checks compatibility only.
- Each explicitly assembled logical Namespace stores its own migration history in its physical Namespace. Records include stable Migration identity, order, and content checksum. Modifying an already applied migration is rejected.
- Migration execution acquires an exclusive lock for the entire physical Database Source. Failure to obtain the lock fails explicitly without background waiting or hidden retries.
- The cross-database Migration Contract does not promise transactional DDL atomicity. An Adapter may use transactions where supported, but logical migrations cannot depend on that guarantee.
- Each migration supports preflight inspection and safe rerun after a partially applied failure. A migration is recorded complete only after all of its steps succeed. Failure stops the plan, reports the actual state, and neither automatically rolls back nor executes later migrations.
- Migrations are forward-only and immutable. There is no general Down migration mechanism. Rollback is allowed only when the older application is explicitly compatible with the current schema; whole-state restore is a later production-hardening capability.
- The Release Manifest declares the exact expected Migration Head for every assembled logical Namespace. A runtime database that is either behind or ahead is incompatible; the application refuses startup and does not become ready.
- PostgreSQL persistent data uses a labeled Docker named volume. Normal stop, upgrade, and uninstall operations retain it. Destructive removal requires exact instance/volume resolution, explicit confirmation, and a dedicated operation.

## Testing Decisions

- Good tests assert observable behavior at the highest practical seam. They do not assert private package structure, SQL text, Router internals, logger implementation details, or third-party framework behavior.
- The primary acceptance seam is the production-built deployment: a real browser loads the SPA from Caddy, reads runtime configuration, executes the generated typed call through Foundation HttpClient, reaches the Go Application Listener through the proxy, and observes the flat versioned response and server-generated `request_id`.
- The same acceptance environment proves visible behavior when runtime configuration is invalid, the API major or Release Identity is incompatible, the application is unavailable, the database is unavailable, or the schema head is incompatible.
- Playwright E2E lives in a root-level test-only area and runs against production-built Caddy, Go, and PostgreSQL Compose services. E2E artifacts and dependencies do not enter production images.
- E2E uses only browser behavior, public HTTP, and the formal Go operations CLI. Production code contains no E2E environment branch, hidden test endpoint, invisible control, authentication bypass, or otherwise unused test hook.
- Go unit and module tests use the standard testing package, hand-written fakes, `httptest`, and real subprocesses as appropriate. A general mocking framework, SQLite substitute, SQLMock, or map-backed fake database is not used to claim persistence correctness.
- React component and module behavior is tested with Vitest and React Testing Library through rendered output and user interaction.
- The composed OpenAPI document is validated for reference completeness and unique semantic codes. TypeScript transport generation is reproducible from the committed contract.
- Go HTTP contract tests send requests to the real Handler boundary and prove status codes, flat response fields, validation field errors, failure `data: null`, request correlation, safe fallback behavior, and conformance to OpenAPI.
- Shared database conformance tests execute the same logical Namespace declarations, migrations, structured statements, projections, transactions, error normalization, compatibility checks, and migration checksum behavior against real PostgreSQL and real MySQL.
- Database conformance tests prove that no Business Module-facing test or fixture changes when the Adapter changes. They also prove that Runtime identities cannot execute DDL and Migrator identities can execute the required migration path.
- Migration acceptance covers Source-level lock contention, modified applied migrations, exact-head checks, partial DDL failure reporting, safe rerun, and refusal to continue to later migrations.
- Process acceptance uses real subprocesses and operating-system signals to prove startup failure, liveness/readiness transitions, bounded graceful drain, critical-component exit behavior, temporary database outage, recovery, and final process exit status.
- Observability acceptance proves structured slog output with request correlation, real OTLP Trace and Metric export to a receiver, bounded exporter failure, and Management Listener isolation from Caddy.
- Release acceptance verifies immutable identities and digests across artifacts, signature rejection and acceptance paths, offline installation without network access, Compose version precheck, non-root/read-only container settings, and production-image absence of development and E2E tooling.
- Frontend build acceptance proves that one static build consumes different valid runtime configurations, rejects invalid public configuration before Router startup, revalidates HTML/runtime JSON, and does not install a Service Worker.
- `task check` is the sole authoritative full gate and CI invokes the same command. Partial tasks may support local iteration but never establish completion independently.
- The current repository contains no implementation or tests, so there is no existing test prior art to reuse. These are intentionally the first permanent high-level seams. Future tests should extend them rather than create parallel acceptance harnesses.

## Out of Scope

- Any business domain model, aggregate, entity, value object, business table, business migration, use case, page, route, sample record, fixture domain, or synthetic product workflow.
- Authentication, login, user, account, role, permission, and business data-authorization design. These are designed at the first real protected business frontier rather than represented by placeholders.
- A generic Business Module interface, module manifest, runtime plugin system, automatic module discovery, or global contract catalogue.
- A generic background job, scheduler, worker, task, queue-consumer, outbox, or retry framework.
- Timeseries database, cache, message system, and object storage runtime components. These are confirmed future infrastructure directions but are neither deployed nor represented by empty abstraction layers in this foundation.
- TimescaleDB or another PostgreSQL extension dependency.
- Multiple physical Database Sources, mixed PostgreSQL/MySQL operation in one deployment, cross-source transactions, distributed transactions, sharding, replication, read/write splitting, automatic failover, or database high availability.
- A generic CRUD Repository, base Entity, universal optimistic concurrency, default soft deletion, default audit columns, or business retry policy.
- Frontend/backend split-origin deployment, CORS policy, cookie authentication, CSRF design, or public browser credentials.
- A general server-state cache, TanStack Query, Redux, Zustand, or another application-wide state framework.
- Server-side rendering, a production Node.js server, Service Worker, PWA installation, browser offline synchronization, mobile application, or desktop application.
- A complete UI component library, business navigation, dashboard, sample page, demo business data, or design-system productization.
- GraphQL, browser gRPC, Go server generation, Handler generation, route generation, use-case generation, persistence generation, Composition generation, or business/domain code generation.
- Kubernetes, multi-node deployment, high availability, zero-downtime deployment, rolling upgrade, Linux ARM, Windows, or native host-process installation.
- Public ACME certificate automation, wildcard CORS, forwarded-header trust from arbitrary proxies, or Docker Socket control from a container.
- Automatic runtime configuration rewriting, automatic schema migration during application startup, general Down migrations, transparent database-operation retries, or silent HTTP security-limit relaxation.
- A default OpenTelemetry Collector in every deployment.
- Machine-readable SBOM and release vulnerability-scan evidence.
- Backup and restore implementation or production-data recovery acceptance. These are the final production-hardening activities after business functionality is complete and before product V1 delivery.
- Tickets, implementation sequencing, or production code. This Spec records the confirmed foundation boundary only.

## Further Notes

- `docs/inspire.md` remains the product vision authority. This Spec only defines the current technical foundation frontier and must not be used to settle unresolved product or domain questions.
- The engineering baseline is not product V1 production readiness. Until backup and restore hardening is completed, foundation-stage persistent data is treated as disposable development or non-production data.
- The existing Go Module declaration does not independently establish an exact supported Go toolchain version. The implementation must pin verified tool versions without treating incidental current file contents as a previously confirmed choice.
- Numeric defaults for HTTP deadlines, body/header limits, connection-pool sizing, backoff, drain windows, and telemetry queues must be finite, explicit, configurable where operationally necessary, and covered by behavior tests. This Spec intentionally does not invent unconfirmed numbers.
- Exact HTTP endpoint paths beyond the confirmed `/api/v1` major prefix, exact CLI subcommand names, physical Namespace names, and configuration key names remain implementation naming details. They must follow the ownership and behavior constraints above and must not introduce new product semantics.
- Backup was intentionally removed from the minimal foundation despite being part of Monitra's long-term operating goal. Its design and acceptance occur after business completion and before formal V1 delivery.
- The first permanent vertical proof is technical: runtime configuration, SPA startup, Foundation HttpClient, Caddy, Go HTTP, Release Identity, request correlation, database compatibility, and operational signals. It must not be expanded with fake business concepts merely to make the demonstration look richer.
