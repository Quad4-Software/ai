---
name: docker
description: >
  This skill covers Docker Engine, the Compose plugin, and container
  operations. Use it for Engine and Compose versions and breaking changes,
  compose-file semantics, networking and DNS behavior, security boundaries
  (socket exposure, rootless, userns-remap, published-port filtering),
  logging and lifecycle, and update automation.
---

## When to use this skill

- You are writing or reviewing a `compose.yaml` or `Dockerfile`.
- You are deploying containers and need the networking, DNS, or
  port-publish behavior right.
- You are hardening a Docker host (socket access, rootless, firewalls).
- You are debugging why a port is reachable that a host firewall should
  block, or why name resolution fails on the default bridge.
- You are planning upgrades and need to know what breaks between Engine
  or Compose versions.

## How to use

1. Read this file for the version map, the compose-file semantics that
   matter, and the security model.
2. Load [references/compose.md](references/compose.md) for field-level
   Compose syntax, and [references/security.md](references/security.md)
   for hardening patterns and the socket/firewall details.
3. Fall back to https://docs.docker.com/engine/ and
   https://docs.docker.com/reference/compose-file/ for anything not
   covered here.

## Examples

- "Why is my unpublished container port reachable from the LAN?"
- "Write a compose file with a healthcheck, a secret, and a memory limit."
- "Explain why `docker compose up` ignores my `version:` field."
- "Set up rootless Docker and explain what stops working."
- "Review this compose file for socket exposure and privileged flags."

# Docker Engine and Compose

Docker is two products that version independently:

- **Docker Engine** (`dockerd` + `containerd` + `runc`) is the daemon that
  runs containers. Latest stable **v29.8.1** (Sept 2026). The v29 series
  launched Nov 2025. v28 and v27 are unmaintained. Only v25.0.x still
  gets LTS patches.
- **Docker Compose** (`docker compose`, a CLI plugin) is the multi-service
  orchestrator for a single host. Latest **v5.5.1** (Sept 2026). V5.0.0
  shipped Dec 2025 and skipped v3/v4 numbering to distance itself from
  the legacy file-version confusion.

Compose is not Swarm. `docker compose up` runs on one daemon. `deploy`
keys other than `resources.limits`/`reservations` are ignored without
Swarm mode.

## Engine v29 breaking changes

- **containerd image store is the default for fresh installs.** The old
  graphdriver path is deprecated. Daemons with `userns-remap` keep the
  legacy store for now. Existing installs are not migrated.
- **Minimum client API v1.44.** The daemon rejects clients older than
  Docker v25. `DOCKER_MIN_API_VERSION` overrides it for stragglers.
- **Experimental nftables backend** via `"firewall-backend": "nftables"`
  in `daemon.json`. Still experimental in v29.8.x, does not work in Swarm
  mode, and does not enable IP forwarding itself. iptables stays the
  default.
- **cgroup v1 deprecated** (supported until at least May 2029). Docker
  Content Trust removed. Go SDK split into `moby/moby/api` and
  `moby/moby/client`.

## Engine v28 breaking changes (Feb 2025)

- **Unpublished ports are dropped by default.** Per-bridge rules drop
  unsolicited inbound traffic to container IPs, so a port that is not in
  a `ports:` mapping is unreachable from outside even if the container
  listens on it. Opt out per daemon with `--ip-forward-no-drop` or per
  network with `gateway_mode_ipv4/ipv6: nat-unprotected`.
- IPv6 got saner defaults: `--ipv6` works without `fixed-cidr-v6` (auto
  ULA), IPv6-only networks via `docker network create --ipv6
  --ipv4=false`, random MACs on container interfaces.

## Compose semantics that matter

- **`version:` is obsolete.** Compose is versionless. The field is parsed
  only for backward compatibility, emits a warning, and never selects a
  schema. Drop it.
- `name:` sets the project name (else directory name or `-p`).
- `depends_on` long form waits on `service_started`, `service_healthy`,
  or `service_completed_successfully`. `restart: true` restarts a dep
  that dies during startup, and `required: false` tolerates a missing
  dep.
- `deploy.resources.limits`/`reservations` (cpus, memory, pids) **are**
  honored by `docker compose up` without Swarm. The other `deploy` keys
  (replicas, placement, update_config) are Swarm-only.
- `secrets:`/`configs:` land at `/run/secrets/<name>` and `/<name>` in
  non-Swarm containers. Long syntax: `source`, `target`, `file: ./path`,
  `environment: VAR`, or `external: true`.
- `env_file:` long form supports `required: false` and `format: raw` (no
  interpolation). `.env` in the project dir is for interpolation only,
  not automatic container env.
- `docker compose build` delegates to **buildx/Bake** since v5. The
  internal builder is gone.

Field-level detail: [references/compose.md](references/compose.md).

## Networking model

- Drivers: `bridge` (default, single host), `host`, `none`, `overlay`
  (Swarm only), `macvlan`/`ipvlan` (container gets its own L3 presence on
  the LAN).
- **Embedded DNS at 127.0.0.11 exists only on user-defined networks.**
  Containers on a `mynet:` network resolve each other by service name and
  alias. The default `bridge` network has no name resolution. `--link`
  is legacy. Put every multi-service project on a user-defined network.
- **Published ports bypass host firewall INPUT rules.** Docker DNATs in
  the `nat` table before `INPUT`/`FORWARD` apply, so ufw/firewalld INPUT
  rules do not filter published ports. Use the `DOCKER-USER` chain
  (evaluated before Docker's own chains) for source filtering, or bind
  publishes to `127.0.0.1:port:port`. Since Engine v28, unpublished
  ports are dropped anyway.
- IPv6: `enable_ipv6: true` on the network. ULA auto-assigns without an
  explicit subnet.

## Security model

- **`/var/run/docker.sock` = root-equivalent on the host.** dockerd runs
  as root and honors any API caller. A socket mount lets you create a
  `--privileged` container bind-mounting `/`. Do not mount it into
  untrusted containers. For Traefik/monitoring that legitimately needs
  the API, front it with `tecnativa/docker-socket-proxy` or
  `wollomatic/socket-proxy` and allowlist only the endpoints needed.
- **Remote API access:** TCP 2375 plaintext is "equivalent to root" and
  since Engine 27 requires an explicit `--tls=false` opt-out on
  non-localhost binds. The supported paths are TLS on 2376
  (`--tlsverify` + client certs) or `DOCKER_HOST=ssh://user@host`.
- **Rootless mode** runs the daemon as an unprivileged user. Needs
  `newuidmap`/`newgidmap` and `/etc/subuid`+`/etc/subgid` entries.
  Networking goes through RootlessKit (slirp4netns default). No ports
  <1024 without `cap_net_bind_service` on rootlesskit or the
  `ip_unprivileged_port_start` sysctl. Cgroup limits need cgroup v2 +
  systemd. No AppArmor, Swarm/overlay, SCTP, or checkpoint.
- **`userns-remap`** daemon option maps container root to unprivileged
  host UIDs while the daemon stays root. Containers under userns-remap
  keep the legacy image store in v29.
- Defaults: the builtin **seccomp** profile filters dangerous syscalls,
  and **AppArmor** `docker-default` applies on Debian/Ubuntu (SELinux on
  RHEL-likes). `no-new-privileges`, `cap_drop: ALL`, `read_only: true`,
  and a non-root `user:` are the standard hardening set.

Hardening patterns and socket-proxy configs:
[references/security.md](references/security.md).

## Lifecycle and ops

- **Restart policies:** `no` (default), `on-failure[:N]`, `always`,
  `unless-stopped`. `unless-stopped` survives daemon restarts but respects
  a manual `docker stop`.
- **Logging:** default `json-file` with no rotation. Set
  `log-driver: local` (ring-buffer binary, faster) or
  `log-opts: {max-size, max-file, compress}` to avoid disk fill.
  `mode: non-blocking` + `max-buffer-size` stops a chatty app from
  blocking on the log driver.
- **Healthchecks:** `docker ps`/`inspect` show `healthy`/`unhealthy`.
  Unhealthy does not restart the container by itself. Compose's
  `service_healthy` dep gates startup, and an autoheal pattern or an
  orchestrator handles restart.
- **Update automation:** Watchtower is **archived** (read-only since Dec
  2025, v1.7.1 final). Drop-in fork: `nickfedor/watchtower`. Actively
  maintained alternatives: **DIUN** (notify-only), **What's Up Docker**
  (`getwud/wud`, UI + optional auto-update). `ouroboros` is long dead.
- **Pin digests** (`image@sha256:...`) for reproducibility. Mutable tags
  drift. Renovate/Dependabot can PR digest bumps.
- **Pruning:** `docker system prune` clears stopped containers, unused
  networks, dangling images, and build cache. `-a` adds all unused
  images, `--volumes` adds anonymous volumes (data risk).

## Alternatives worth knowing

- **Podman** (v5.8.x): daemonless, rootless-first, drop-in CLI. Quadlet
  `.container` files generate systemd units.
- **nerdctl + containerd**: Docker-compatible CLI straight on containerd,
  good for k8s-adjacent hosts.
- **macOS/Windows dev:** OrbStack (fast proprietary), Colima (FOSS
  Lima-based), Rancher Desktop (containerd or moby backend).
