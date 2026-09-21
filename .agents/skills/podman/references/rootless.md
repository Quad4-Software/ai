# Rootless Podman

Rootless is the default posture, not a mode you bolt on. Each user owns
storage (`~/.local/share/containers`), config (`~/.config/containers`),
and a socket (`/run/user/$UID/podman/podman.sock`).

## Requirements

- `newuidmap`/`newgidmap` installed and `/etc/subuid`, `/etc/subgid`
 entries granting each user a UID range (shadow-utils handles this on
 modern distros).
- cgroup v2 + systemd for resource limits (mandatory since v6).
- `loginctl enable-linger <user>` for services that outlive the login
 session.

## Networking: pasta

Pasta is the only rootless network backend in v6 (slirp4netns removed).
Differences that bite:

- Pasta gives the container the **same address as the host interface**
 by default, not a NAT alias. `podman run` port publishing maps onto
 the host's address.
- `-p 8080:80` on a port <1024 fails unless
 `net.ipv4.ip_unprivileged_port_start` is lowered or pasta gets
 CAP_NET_BIND_SERVICE.
- ICMP/ping needs `net.ipv4.ping_group_range` covering the user's
 groups.
- `--network bridge` uses netavark + aardvark-dns (v2 since Podman 6)
 for inter-container DNS on user-defined networks, same as rootful.
- Pasta copies host resolv.conf/addresses more faithfully than
 slirp4netns did. VPN and split-DNS setups behave differently. Check
 `pasta(1)` options via `containers.conf` `[network] pasta_options`.

## Storage

- Default driver is `overlay` on fuse-overlayfs for rootless (native
 overlayfs works where the kernel allows user-mount overlay, which is
 most modern distros).
- `podman system migrate` handles backend migrations. BoltDB to SQLite
 must run on 5.8 before a v6 upgrade.
- `podman system reset` wipes all rootless state (images, volumes,
 networks). Destructive.

## The API socket

```bash
systemctl --user enable --now podman.socket
export DOCKER_HOST=unix:///run/user/$UID/podman/podman.sock
```

Docker-compatible clients (docker-compose, Testcontainers, some IDE
plugins) work against it. It is per-user and unprivileged, so it is
safer than /var/run/docker.sock, but it still grants control of that
user's containers and storage - treat it as that user's root.

## Common failures

| Symptom | Cause |
| --- | --- |
| `cannot set up namespace` / subuid errors | missing /etc/subuid range, or stale `/etc/nsswitch.conf` sssd order |
| Port <1024 bind fails | ip_unprivileged_port_start sysctl |
| No outbound DNS in container | pasta picked up a broken host resolver, check containers.conf |
| `there might not be enough IDs` | subuid range too small for image's UID spread |
| Volume mount permission denied | keep-id (`--userns=keep-id`) or :U volume flag needed for rootless UID mapping |

## Rootless vs rootful at a glance

- Rootless: per-user, no daemon, unprivileged ports only, no host
 netns manipulation, limited resource control without delegation.
- Rootful (`sudo podman` or system Quadlets): full netavark features,
 macvlan, all ports, but the socket is still root-equivalent per user
 namespace scope. Prefer system Quadlets over `sudo podman` daemons.
