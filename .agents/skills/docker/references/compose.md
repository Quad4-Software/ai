# Compose file reference

Compose is versionless. `version:` at the top of the file is obsolete,
emits a warning, and never selects a schema. Compose always validates
against the latest spec. The top-level `name:` field sets the project
name (otherwise the directory name or `-p`).

## Structure

```yaml
name: myapp

services:
  app:
    image: ghcr.io/example/app:1.2.3
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - app-data:/data
    env_file:
      - path: .env.app
        required: false
    environment:
      LOG_LEVEL: info
    depends_on:
      db:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M
    secrets:
      - db-password
    networks:
      - app-net

  db:
    image: postgres:17
    restart: unless-stopped
    volumes:
      - db-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 3s
      retries: 5
    secrets:
      - db-password
    networks:
      - app-net

networks:
  app-net:
    driver: bridge

volumes:
  app-data:
  db-data:

secrets:
  db-password:
    file: ./secrets/db-password
```

## Fields that matter

- **`image:`**: pin a tag or digest (`image@sha256:...`). `latest` drifts.
- **`build:`**: `context`, `dockerfile`, `args`, `target`. Since Compose
  v5, `docker compose build` delegates to buildx/Bake. Multi-platform
  builds use `platforms`.
- **`ports:`**: `"host:container"` or long form with `published`,
  `target`, `protocol`, `host_ip`. `host_ip: 127.0.0.1` keeps a publish
  off the LAN.
- **`volumes:`**: `"host:container[:ro]"` binds, named volumes, or
  `tmpfs`. Named volumes persist across `down`/`up`. `down -v` deletes
  them.
- **`environment:` / `env_file:`**: `environment` wins over `env_file`.
  `env_file` long form adds `required: false` and `format: raw`. `.env`
  in the project dir is for interpolation only and is not auto-loaded as
  container env. Precedence: shell > `--env-file` > `.env`.
- **`depends_on:`**: long form with `condition:
  service_started|service_healthy|service_completed_successfully`,
  `restart: true` (restart the dep if it dies during startup),
  `required: false`.
- **`healthcheck:`**: `test` (`NONE`, `CMD`, `CMD-SHELL`), `interval`,
  `timeout`, `retries`, `start_period`, `start_interval`.
- **`deploy:`**: only `resources.limits`/`reservations` (cpus, memory,
  pids) apply without Swarm. `replicas`, `placement`, `update_config`,
  `rollback_config`, `restart_policy` are Swarm-only and ignored by
  `docker compose up`.
- **`secrets:` / `configs:`**: long syntax `source`, `target`. Sources
  via `file:`, `environment:`, or `external: true`. Non-Swarm containers
  get secrets at `/run/secrets/<name>`, configs at `/<name>`.
- **`networks:`**: service-level list joins named networks. Top-level
  `networks:` defines them. `driver: bridge` is the default and supports
  embedded DNS. `internal: true` cuts outbound. `enable_ipv6: true` for
  IPv6.
- **`profiles:`**: per-service list. The service only starts under
  `--profile <name>` or `COMPOSE_PROFILES`. Use it for optional
  tooling/debug containers.
- **`include:`**: top-level list merging other compose files into this
  project.
- **`develop.watch:`**: `action: sync|sync+restart|rebuild` with
  `path`/`target`/`ignore`/`include`/`initial_sync`. Driven by
  `docker compose watch` or `up --watch`.
- **`logging:`**: `driver` (`json-file`, `local`, `syslog`, `journald`,
  `fluentd`) plus `options` (`max-size`, `max-file`, `compress`, `mode`,
  `max-buffer-size`).

## Commands

- `docker compose up -d`: create and start. Recreates containers whose
  config changed.
- `docker compose down`: stop and remove containers and networks. `-v`
  also drops named volumes (data risk).
- `docker compose ps`, `logs -f`, `exec <svc> sh`, `restart <svc>`,
  `pull`, `build`, `config` (renders the merged file).
- `docker compose watch`: develop.watch file sync.
- `docker compose run --rm <svc> <cmd>`: one-shot command in a service's
  image and network context.
- `docker compose up --scale <svc>=3`: only for services without
  `container_name` or fixed `ports`.

## Interpolation

`${VAR}`, `${VAR:-default}`, `${VAR:?error}` inside the file. Sources in
order: shell env, `--env-file`, `.env` in the project dir. Use
`$${VAR}` for a literal `$` that Compose should not expand.

## Common gotchas

- `depends_on` with `condition: service_healthy` waits for the dep's
  healthcheck to pass, but the healthcheck has to exist. Without one,
  `service_started` is the only reliable gate.
- A `ports:` publish on the default `bridge` network still works, but
  name resolution between services does not. Define a network.
- `container_name` disables `--scale` and breaks Compose's service
  discovery on some edge cases. Avoid it unless you need a fixed name
  for external tooling.
- `restart: always` restarts a container even after `docker stop`. Use
  `unless-stopped` for the usual "keep it running unless I stopped it"
  semantics.
- Compose v5.5.0 reconciles image digests: the first `up` after an
  upgrade may recreate containers once. `pull` honors `pull_policy`
  refresh windows.
