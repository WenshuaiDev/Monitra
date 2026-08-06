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
