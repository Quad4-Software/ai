# k3s operations

Docs root: https://docs.k3s.io/

## Install

```bash
curl -sfL https://get.k3s.io | sh -s - server \
  --write-kubeconfig-mode 644 --disable=traefik
```

- Install via env vars, not args: `INSTALL_K3S_VERSION=v1.36.4+k3s1`
 (pin!), `INSTALL_K3S_EXEC="server --disable=traefik"`,
 `INSTALL_K3S_CHANNEL=stable|latest|v1.36`, `K3S_URL`, `K3S_TOKEN`.
- The script is fetched over HTTPS and executes as root. For pinned
 installs prefer downloading the binary + checksums from the GitHub
 release directly, or use the airgap path below.
- Uninstall: `/usr/local/bin/k3s-uninstall.sh` (server),
 `k3s-agent-uninstall.sh` (agent). `k3s-killall.sh` stops all
 containers first. Run it before wiping.

## Config file

`/etc/rancher/k3s/config.yaml` (or `config.yaml.d/*.yaml` drop-ins)
maps every CLI flag to YAML, kebab-case to the flag name:

```yaml
disable:
  - traefik
  - servicelb
flannel-backend: wireguard-native
cluster-cidr: 10.42.0.0/16
service-cidr: 10.43.0.0/16
```

Restart the k3s/k3s-agent systemd unit after edits.

## Datastore

| Mode | Flag | When |
| --- | --- | --- |
| sqlite (default) | none | single server only |
| embedded etcd | `--cluster-init` on first server | HA control plane, odd node count |
| external | `--datastore-endpoint=` | managed etcd/MySQL/Postgres |

- Adding servers to an etcd cluster: point them at the first server
 via K3S_URL + token. `k3s server --server https://first:6443`.
- etcd snapshots: `k3s etcd-snapshot save --name pre-upgrade`.
 Stored under `/var/lib/rancher/k3s/server/db/snapshots`, optional
 `--s3` upload. Restore: `k3s server --cluster-reset
 --cluster-reset-restore-path=<file>`.

## registries.yaml (containerd mirrors + auth)

`/etc/rancher/k3s/registries.yaml` is read by the bundled containerd:

```yaml
mirrors:
  docker.io:
    endpoint:
      - "https://registry.internal/v2/docker-hub"
      - "https://registry-1.docker.io"
configs:
  registry.internal:
    auth:
      username: ci
      password: ${REGISTRY_PASSWORD}   # env expansion in this file
    tls:
      ca_file: /etc/rancher/k3s/ca.crt
```

Changes need a k3s restart. Auth in configs applies to both pulls and
the embedded image store. For a pull-through cache, put your mirror
first and upstream last in endpoint order.

## Airgap

1. Download the release assets: `k3s`, `k3s-airgap-images-<arch>.tar.zst`
 (or .tar.gz), `install.sh`, and `sha256sum-<arch>.txt`. Verify sums.
2. Place images in `/var/lib/rancher/k3s/agent/images/` and the binary
 on PATH.
3. `INSTALL_K3S_SKIP_DOWNLOAD=true ./install.sh`.
4. Private registry still needed for app images. Wire it via
 registries.yaml.

## Upgrades and backups

- Upgrade one k8s minor at a time. `INSTALL_K3S_VERSION` pins the
 target. Re-run the script per node, servers first then agents.
- Rancher's system-upgrade-controller automates rolling upgrades via
 Plan CRDs.
- Backups: etcd snapshots (above) or sqlite file copy while stopped.
 `/var/lib/rancher/k3s/server/` holds datastore, certs, node-token,
 and creds - it is the crown jewel directory.

## Troubleshooting quick map

| Symptom | Check |
| --- | --- |
| `kubectl` fails from laptop | kubeconfig server is 127.0.0.1, rewrite to node IP, keep CA+client certs |
| Agent cannot join | 6443 reachability, wrong token, clock skew (certs are time-sensitive) |
| Pods no network | flannel iface choice on multi-homed hosts: `--flannel-iface` |
| Images pull slowly or fail | registries.yaml endpoint order. `crictl`/`k3s crictl images` to inspect |
| NodeNotReady after upgrade | cgroup driver drift or containerd template clobbered: check config.toml.tmpl |
| 403/forbidden on /mcp-style endpoints | not k3s. Check the app's own auth |

`k3s check-config` audits kernel and cgroup prerequisites. Logs:
`journalctl -u k3s` (server) or `-u k3s-agent`.
