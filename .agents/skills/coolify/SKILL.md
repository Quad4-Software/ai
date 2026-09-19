---
name: coolify
description: >
  This skill covers Coolify, the self-hosted PaaS. Use it for
  architecture (control plane vs managed servers), install and upgrade
  flows, the Projects/Environments/Resources model, build packs, the
  Traefik proxy, GitHub App and deploy-key integrations, preview
  deployments, API v1, backups, Sentinel metrics, and the v4->v5
  roadmap.
---

## When to use this skill

- You are deploying or operating Coolify, or choosing between it and
  Dokploy, CapRover, Dokku, Kamal, or plain compose.
- You need the control-plane layout, SSH-based management model, or
  resource hierarchy.
- You are wiring git push deploys, PR previews, or the API.
- You are planning backups, multi-server deploys, or the v4->v5
  migration.
- You are debugging why a deployment, proxy route, or preview URL
  failed.

## How to use

1. Read this file for the architecture, resource model, deploy flow,
   and ops surface.
2. Fall back to https://coolify.io/docs for field-level detail and
   https://github.com/coollabsio/coolify/releases for release notes.

## Examples

- "Walk me through what happens when I push to a repo Coolify watches."
- "Set up PR preview deployments with a wildcard domain."
- "Explain why my Docker Compose app does not get rolling updates."
- "Back up the Coolify panel and every managed database to S3."
- "Compare Coolify to Dokploy for a multi-node setup."

# Coolify

Coolify is a self-hosted, open-source alternative to Heroku/Netlify/
Vercel/Railway, also offered as managed Coolify Cloud
(app.coolify.io). Latest stable **v4.3.23** (Sept 2026). The 4.x line
went stable April 2026 after about two years of betas. V5.x is in early
development as a Rust rewrite (`coold`) with native clustering.

## Architecture

The control plane is four containers on the management host:

- `coolify`: Laravel app (Livewire 3 UI), dashboard + API, queued jobs
  via Horizon + scheduler.
- `coolify-db`: PostgreSQL 15 for config and state.
- `coolify-redis`: Redis 7 for queues/cache.
- `coolify-realtime`: Soketi for dashboard websockets and the browser
  terminal.

Management is **SSH**, not an agent fabric. Coolify connects to each
managed server over SSH and runs Docker commands. Each managed server
gets a `coolify-proxy` container (the reverse proxy) and optionally a
`coolify-sentinel` container (metrics agent that pushes status to the
panel over HTTPS with a generated token). A managed server only needs
SSH + Docker.

## Install

`curl -fsSL https://cdn.coollabs.io/coolify/install.sh | sudo bash`

The script:

1. Installs curl, wget, git, jq, openssl.
2. Installs Docker Engine 24+ via get.docker.com.
3. Creates `/data/coolify/{source,ssh,applications,databases,backups,
   services,proxy,...}`.
4. Generates an ed25519 SSH key and adds it to root's
   `authorized_keys` so localhost is itself a managed server.
5. Downloads `docker-compose.yml`, `docker-compose.prod.yml`, `.env`,
   `upgrade.sh` from cdn.coollabs.io and runs `docker compose up -d`.

Auto-install supports Ubuntu LTS (20.04/22.04/24.04). Other distros use
the manual install. Minimums: 2 CPU, 2 GB RAM, 30 GB disk, Docker 24+.

## Resource model

`Projects -> Environments (production, staging) -> Resources`. Each
resource deploys to a `Server -> Destination` (a Docker network. The
default is `coolify`).

- **Applications:** git repos or images. Build packs: Nixpacks,
  Railpack (beta, v4.1+), Static, Dockerfile, Docker Compose (empty or
  repo-based), Docker Image (pre-built from a registry). Sources:
  public git URL, deploy key (any provider), or GitHub App.
- **Databases:** PostgreSQL, MySQL, MariaDB, MongoDB, Redis,
  Dragonfly, KeyDB, ClickHouse. Scheduled DB-aware backups exist for
  Postgres/MySQL/MariaDB/Mongo/ClickHouse, not for Redis/Dragonfly/
  KeyDB.
- **Services:** ~200+ pre-configured Compose templates (n8n,
  Plausible, Vaultwarden, Nextcloud, Uptime Kuma). Any Compose stack
  runs as "Docker Compose Empty".
- **Scheduled tasks:** cron commands inside a resource's container,
  60-36000 s timeout, execution history kept.
- **Shared variables:** env vars at team/project/environment/server
  scope, referenced as `{{team.NAME}}`, `{{project.NAME}}`,
  `{{environment.NAME}}`, `{{server.NAME}}`.

## Networking and proxy

One `coolify-proxy` per server. **Traefik is the default**. Caddy is an
integrated but less-documented alternative. "Custom (None)" lets you
bring your own. Automatic Let's Encrypt via the proxy, routing driven
by generated Docker labels. Wildcard domains are supported and required
for PR previews (e.g. `*.preview.example.com`).

Panel ports: `8000/tcp` dashboard, `6001/tcp` realtime, `6002/tcp` web
terminal, `22/tcp` SSH, `80/443` for proxied traffic and ACME. Once the
dashboard sits behind a domain, 8000/6001/6002 can close publicly.
Traefik versions supported in v4.3.21+: 3.7.13 / 3.6.25 / 2.11.57.

## Deploy flow

- **Git push triggers:** the GitHub App (manifest-flow setup on
  github.com, manual for GH Enterprise) handles repo selection,
  webhooks, PR events, and commit statuses. Deploy keys + manual
  webhooks cover GitLab/Bitbucket/Gitea/any git host.
- **PR previews:** each PR gets its own URL (template with
  `{{random}}`/`{{pr_id}}`), auto-deleted on merge/close, separate
  preview env vars, optional PR comments (needs GitHub App PR
  read/write).
- **Rolling updates:** for Nixpacks/Railpack/Static/Dockerfile/
  Docker-Image apps, Coolify starts the new container alongside the
  old, waits for health check, swaps, removes. Near-zero-downtime but
  not guaranteed. **Not supported for Docker Compose apps.**
- **Health checks:** dashboard-configured HTTP/command checks (need
  curl/wget in the image), Dockerfile `HEALTHCHECK`, or Compose
  `healthcheck`. Traefik drops unhealthy containers from routing.
- **Build servers:** a server flagged "build server" runs builds,
  pushes the image to a registry, and the deployment server pulls.
  Requires matching CPU arch and a registry.
- **API v1:** base `{coolify-url}/api/v1`, Bearer tokens from Keys &
  Tokens -> API tokens, team-scoped with `read`, `read:sensitive`,
  `write`, `deploy`, `root` permissions. Endpoint groups:
  `applications`, `deployments`, `deploy` (by uuid/tag, `?force=true`,
  PR ids), `services`, `databases`, `servers`, `projects`,
  `environments`, `teams`, `security`, `s3`. `/api/health` is
  unauthenticated.
- **CLI:** official `coolify-cli` (Go, MIT), latest v1.7.0. Contexts,
  `app`, `database backup`, `deploy`, `s3`, `resource list`, JSON
  output for CI.
- **Notifications:** email (SMTP/Resend/system), Discord, Telegram,
  Slack + Mattermost, Pushover, generic webhook. Per-channel event
  selection.

## Ops

- **Backups:** cron-scheduled database dumps (per-database selection,
  retention, parallel gzip), optional upload to any S3-compatible
  storage. Volume/file backups and a scheduled backup of the Coolify
  instance itself exist.
- **Logs:** deployment logs (per-deployment UUID, streamable,
  API-accessible since v4.3.23) and container logs in the dashboard.
- **Sentinel:** `coolify-sentinel` (Rust) reports server + container
  health to the panel, optional CPU/memory metrics (10 s default, 7-day
  retention, SQLite on the server), optional traffic analytics from
  proxy access logs. Metrics are unavailable for Compose/service
  deployments.
- **Updates:** automatic (cron-checked), semi-automatic ("Upgrade"
  button), or manual (re-run install/`upgrade.sh`). Logs in
  `/data/coolify/source/upgrade-*.log`.
- **Multi-server:** add SSH-reachable Docker hosts, deploy resources
  per-server. An external LB is recommended for cross-server HA.
- **Swarm is deprecated.** It was never fully implemented and will be
  removed in v5. Existing v4 Swarm deployments keep working, but do
  not create new ones. v5 replaces it with native Compose replicas
  plus Coolify's own scaling.
- **Kubernetes:** no K8s destination in current v4. An experimental K8s
  destination shipped briefly in early v4 betas and was dropped.
  Maintainers point at v5's own scaling instead.

## Security model

- Email/password auth, optional OAuth/SSO, per-user TOTP 2FA with
  recovery codes, team/role model.
- Team-scoped API tokens with IP restrictions and expiry.
- Encrypted secret storage and webhook signature verification.
- Panel-to-server communication is SSH (key-based), not mTLS.
  Sentinel-to-panel is HTTPS + token.

## Limitations

- **Single control node.** The panel is one Docker Compose stack and
  `coolify-db` is a single Postgres instance. No native HA for the
  panel itself. Protect `/data/coolify`, the `.env`, and the SSH keys.
  Take off-box backups. If the panel is down, running apps keep
  running but deploys and management stop.
- No built-in orchestrator failover today. Swarm is going away and v5's
  scaling is not out.

## Alternatives

- **Dokploy**: closest competitor: Traefik-based, multi-node via
  Docker Swarm, polished UI.
- **CapRover**: older one-click PaaS on Docker Swarm + nginx. Mature,
  less actively modernized.
- **Dokku**: minimalist single-host git-push PaaS. CLI-first, no GUI.
- **Portainer**: container management UI, not a PaaS (no git-build
  pipeline). Pair with your own CI.
- **Kamal 2**: 37signals' CLI deploy tool: zero-downtime container
  deploys to plain VMs over SSH with kamal-proxy. No dashboard or
  multi-tenant layer.
- **Plain `docker compose`**: pick it for few services on one host
  when you do not need git-push builds, PR previews, RBAC, or a UI.
