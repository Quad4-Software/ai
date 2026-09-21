---
name: k3s
description: >
  This skill covers k3s, the lightweight CNCF-certified Kubernetes
  distribution from SUSE/Rancher, tracking upstream Kubernetes
  (v1.37.0+k3s1 current, Sept 2026). Use it for install flags and env
  vars, server/agent topology, the bundled components (containerd,
  flannel, servicelb, traefik, local-path-provisioner, helm-controller),
  datastore choices (sqlite vs embedded etcd vs external), registries.yaml
  mirrors, airgap installs, snapshots, and hardening.
---

## When to use this skill

- You are installing or upgrading a k3s node or cluster.
- You are choosing between embedded etcd, sqlite, and an external
  datastore.
- You are configuring private registry mirrors, airgap images, or
  disabling bundled components.
- You are debugging the kubeconfig, flannel, traefik, or the klipper
  load balancer.

## How to use

1. Read this file for the architecture and defaults.
2. Load [references/ops.md](references/ops.md) for install flags,
   config file, datastore, registries, airgap, backup, and upgrade
   procedures.
3. Fall back to https://docs.k3s.io/ for field-level detail.

## Examples

- "Install a 3-server HA k3s cluster with embedded etcd."
- "Point k3s at our internal registry mirror and disable traefik."
- "Why does kubectl on my laptop fail after a k3s install?"
- "Airgap-install k3s on an isolated node."

# k3s

## What it is

A single ~70 MB binary wrapping upstream Kubernetes plus the batteries
most clusters actually run: containerd (2.x), flannel CNI, CoreDNS,
servicelb (klipper LoadBalancer), traefik ingress, local-path-provisioner,
metrics-server, helm-controller, and a network policy controller. CNCF
certified, so manifests behave the same as upstream.

Versions track upstream Kubernetes directly: **v1.37.0+k3s1** is the
current release line (Sept 2026). The `+k3sN` suffix is the packaging
revision. Upgrade one minor at a time. Do not skip.

## Topology

- `k3s server` - control plane (+ default worker). Multiple servers
  give HA.
- `k3s agent` - worker, joins via `K3S_URL=https://server:6443` and
  `K3S_TOKEN` from `/var/lib/rancher/k3s/server/node-token` on the
  server.
- Single-server default datastore is **sqlite**. Two or more servers
  need **embedded etcd** (`--cluster-init` on the first) or an external
  datastore (`--datastore-endpoint` for etcd/MySQL/Postgres).
- `--docker` is a hard error (dockershim died in k8s 1.24). k3s always
  runs its bundled containerd.

## Defaults worth changing

- **traefik and servicelb are on by default.** Disable with
  `--disable=traefik --disable=servicelb` if you run your own ingress/
  LB. Same mechanism disables servicelb, coredns, metrics-server,
  local-storage, helm-controller.
- **kubeconfig** lands at `/etc/rancher/k3s/k3s.yaml`, mode 600 root.
  `K3S_KUBECONFIG_MODE=644` on install, or copy it and fix the
  `127.0.0.1` server address to the node's IP for remote kubectl.
- **flannel backend** defaults to vxlan. `--flannel-backend=wireguard-
  native`, `host-gw`, or `none` (bring your own CNI, then disable the
  built-in policy controller too: `--disable-network-policy`).
- **Ports:** 6443 apiserver (agents + kubectl), 8472/udp flannel vxlan,
  51820/udp wireguard-ipv4 (51821 ipv6), 10250 kubelet, 2379/2380 etcd
  (embedded), 9345 for new-server join. NodePort range 30000-32767
  handled by servicelb via host ports.
- **containerd config** is managed by k3s. Edit via
  `/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl`, never
  the rendered config.toml. Registry mirrors and auth live in
  `/etc/rancher/k3s/registries.yaml`.
- **Config file** `/etc/rancher/k3s/config.yaml` maps flags to keys
  (kebab to camelCase). Prefer it over editing systemd units.

## Security posture

- CIS benchmark: k3s ships hardened-by-default versus upstream (fewer
  ports, restricted component perms). The docs carry a CIS hardening
  guide and a self-assessment per release.
- Secrets encryption at rest: `k3s secrets-encrypt` (AES-CBC default,
  supports rotation and disable).
- SELinux supported via `--selinux` or the packaged policy. Rootless mode
  exists but is experimental.
- The node-token is a cluster join credential: treat
  `/var/lib/rancher/k3s/server/node-token` as secret, and rotate via
  `k3s token rotate`.
- Default CNI is not encrypted (vxlan plaintext). Use wireguard backend
  or an external CNI where east-west traffic needs protection.

## Ecosystem position

- vs **kind/minikube**: k3s is a real cluster (persistent, multi-node,
  systemd), not a dev-only sandbox.
- vs **k0s/MicroK8s**: similar single-binary class. k3s is the most
  resource-light and airgap-friendly. MicroK8s uses snaps. k0s keeps a
  purer upstream layout.
- vs **RKE2**: RKE2 is Rancher's hardened/FIPS-focused sibling. Pick it
  for compliance regimes, k3s for footprint.

## Sources

- Docs: https://docs.k3s.io/
- Releases: https://github.com/k3s-io/k3s/releases
- Install script: https://get.k3s.io
