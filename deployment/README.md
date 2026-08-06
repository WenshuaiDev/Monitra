# Production startup loop

The production Compose topology publishes only Caddy on loopback. PostgreSQL,
the core Application Listener, and the Management Listener remain on the
deployment network.

First-time local setup:

```sh
cp deployment/production.env.example deployment/production.env
printf 'replace-with-a-local-password\n' > deployment/secrets/postgresql-password
chmod 600 deployment/secrets/postgresql-password
```

Set a deployment-specific immutable `MONITRA_RELEASE_IDENTITY` in
`deployment/production.env`, then use the root Tasks:

```sh
task production:up
task production:diagnose
task production:down
```

`production:down` preserves the PostgreSQL and runtime-config volumes. Set
`MONITRA_PRODUCTION_ENV_FILE` to use an env file at another path. The PostgreSQL
password is referenced as a Compose Secret file and must never be committed.

## Browser startup acceptance

Run all browser-visible startup scenarios against isolated production-built
services with:

```sh
task test:e2e
```

The Task installs the pinned Playwright Chromium runtime, builds the production
core and Caddy images, and verifies the compatible startup state plus invalid
runtime config, unavailable core, API-major mismatch, and Release Identity
mismatch. Each scenario adds a browser screenshot, Compose state, and timestamped
service logs to `playwright-report/index.html`. The isolated Compose project and
its volumes are removed whether the suite passes or fails.

## Authoritative quality gate

`task check` is the sole complete acceptance entry point for this frontier. It
runs formatting, Go analysis and tests, OpenAPI generation drift detection,
TypeScript checks and Web tests, the production Web build, real PostgreSQL
integration, production-stack checks, Playwright browser acceptance, the
production database lifecycle, and production-image purity inspection.

Run it from the repository root after installing the locked dependencies:

```sh
pnpm install --frozen-lockfile
task check
```

The runner stops at the first failed child gate and returns that failure to the
caller. Its contract test uses an independent Task fixture to prove both the
complete ordered step list and nonzero propagation.

The final image inspection exports the filesystems of
`monitra-core:ticket06-test` and `monitra-web:ticket06-test`, reports their image
IDs, file counts, and installed Alpine packages, then rejects Playwright,
browser runtimes, E2E/test source, test controls, Node package trees, and
development servers. A checked-in contaminated manifest proves that the same
checker rejects forbidden files and dependencies independently of the real
images.

GitHub Actions performs tool and dependency setup, then invokes the same
`task check`; it does not maintain a second acceptance command list.
