# Docker security reference

## The socket is root

`/var/run/docker.sock` grants full host control. dockerd runs as root and
honors any caller, so a socket mount lets the container create a
`--privileged` sibling bind-mounting `/`. This is the single most
important Docker security fact.

- Never mount the socket into a container you do not fully trust.
- For tools that legitimately need the API (Traefik, some monitoring),
  front it with a socket proxy that allowlists endpoints:
  - `tecnativa/docker-socket-proxy`: HAProxy-based, per-endpoint env
    allowlist (`CONTAINERS=1`, `IMAGES=1`, etc.).
  - `wollomatic/socket-proxy`: scratch Go image, regex allowlists per
    method, IP allowlist, secure defaults.
- The proxy itself is a privileged service. Bind it to localhost or an
  internal network only.

## Remote API access

- TCP 2375 plaintext = unauthenticated root over the network. Since
  Engine 27, binding a non-localhost TCP socket without TLS requires an
  explicit `--tls=false`/`--tlsverify=false` opt-out or the daemon fails
  at startup. `tcp://localhost` is exempt.
- The supported paths are TLS on 2376 (`--tlsverify` + a CA + client
  certs) or `DOCKER_HOST=ssh://user@host`, which tunnels the API over
  SSH with no new port.
- Do not put `daemon.json` `hosts` entries on a public interface without
  TLS.

## User namespaces

Two ways to stop container-root from being host-root:

- **Rootless mode** runs dockerd itself as an unprivileged user. Install
  with `dockerd-rootless-setuptool.sh` after setting up `newuidmap`,
  `newgidmap`, and >=65,536 entries in `/etc/subuid` and `/etc/subgid`.
  Trade-offs:
  - Networking goes through RootlessKit (slirp4netns default, pasta or
    vpnkit alternatives). Source IPs do not propagate to `-p` by default.
  - No ports below 1024 unless `rootlesskit` gets `cap_net_bind_service`
    or the `net.ipv4.ip_unprivileged_port_start` sysctl is lowered.
  - `--cpus`, `--memory`, `--pids-limit` are silently ignored without
    cgroup v2 + systemd with delegated controllers.
  - No AppArmor, no Swarm/overlay networks, no SCTP, no checkpoint.
  - overlay2 storage needs kernel >=5.11, else it falls back to
    fuse-overlayfs.
- **`userns-remap`** keeps dockerd as root but maps container UIDs into
  unprivileged host ranges via subuid. Trade-off: v29 keeps these
  daemons on the legacy image store, and sharing volumes between
  remapped and non-remapped containers needs UID-aware host paths.

## Mandatory-access defaults

- **seccomp**: the builtin profile blocks ~44 dangerous syscalls
  (`clone` with certain flags, `mount`, `kexec_load`, etc.).
  `--security-opt seccomp=unconfined` disables it. Do not.
- **AppArmor**: `docker-default` profile on Debian/Ubuntu. Not available
  under rootless. `apparmor=unconfined` disables.
- **SELinux**: labeling on RHEL-likes. `:z` (shared) or `:Z` (private) on
  volume mounts relabels for container access.

## Hardening a service

```yaml
services:
  app:
    image: ghcr.io/example/app:1.2.3
    user: "1000:1000"
    read_only: true
    tmpfs:
      - /tmp
      - /run
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE   # only if the app binds <1024
    security_opt:
      - no-new-privileges:true
    pids_limit: 200
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M
```

- `cap_drop: ALL` then `cap_add` only what is actually needed. Most web
  apps need nothing, or `NET_BIND_SERVICE` if they must bind a low port.
- `read_only: true` plus `tmpfs` for `/tmp` and `/run` removes the
  writable filesystem a container escape would need.
- `no-new-privileges` stops setuid binaries inside the container from
  gaining rights.
- `user:` drops to a non-root UID. Match it to the image's expected user
  or a host UID that owns the volumes.
- `pids_limit` and `deploy.resources.limits` bound fork bombs and memory
  exhaustion.

## Firewall behavior

- Published ports DNAT in the `nat` table before `INPUT`/`FORWARD`
  apply, so ufw/firewalld INPUT rules do not see them. Filter published
  traffic in the `DOCKER-USER` chain (evaluated before Docker's own
  chains) or bind the publish to `127.0.0.1`.
- Since Engine 28, unpublished ports are dropped by per-bridge rules.
  Before 28, a container listening on an unpublished port was reachable
  from the LAN through the bridge.
- The `firewall-backend: nftables` daemon option is experimental in v29
  and does not work in Swarm mode. iptables remains the default.

## Image supply chain

- Pin digests (`image@sha256:...`) for anything that matters. Mutable tags
  can be re-pushed.
- Prefer minimal bases: `-slim`, `alpine`, `gcr.io/distroless/...`, or
  `scratch` + a static binary. Every package in the image is attack
  surface and update burden.
- `docker scout cves <image>` lists known CVEs against the Hub's
  advisory feed.
- Do not bake secrets into layers (`ENV`, `COPY` of `.env`, `ARG` used
  in a layer that persists). Build-time `ARG` values land in image
  history. Use `--secret` mounts in buildx or Compose `secrets:` at
  runtime.
