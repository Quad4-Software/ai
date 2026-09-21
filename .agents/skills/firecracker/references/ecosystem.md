# MicroVM ecosystem

## VMM alternatives

- **Cloud Hypervisor** (Linux Foundation, active): the full-featured
 Rust VMM. KVM + MSHV, PCI/VFIO passthrough, live migration, CPU and
 memory hotplug, Windows guests, virtio-fs. Pick it when Firecracker
 minimalism loses a requirement.
- **libkrun / krunvm / muvm** (active): library VMM, KVM on Linux and
 HVF on macOS. virtio-gpu (venus) makes it the GPU-microVM path, and
 TSI socket impersonation. Used by crun/Podman integration and muvm
 for gaming on small-page kernels.
- **crosvm** (Google, active): ChromeOS VMM, per-device jailed
 processes via minijail, virtio-gpu/audio/wayland.
- **QEMU microvm machine** (active): Firecracker-inspired minimal
 machine type inside QEMU, virtio-mmio, no PCI/ACPI.
- **Kata Containers**: Firecracker backend exists in the Rust runtime
 (Kata 4.x) but is second class - bundled FC is stale and has known
 bugs. Officially supported hypervisors are QEMU, Cloud Hypervisor,
 Dragonball.

## Orchestration over Firecracker

- **flintlock** (liquidmetal-dev, active): gRPC microVM lifecycle
 service, containerd-backed, Cluster API provider for k8s-node
 microVMs. Community owned post-Weaveworks.
- **firecracker-containerd**: effectively maintenance mode, still on
 containerd 1.7. Works but check health before committing.
- **microvm.nix** (active): declarative NixOS microVMs across
 firecracker, cloud-hypervisor, qemu, crosvm, kvmtool, stratovirt,
 alioth, vfkit.
- **firecracker-go-sdk**: maintained at low velocity, no tagged
 release since v1.0.0 (2022) but commits continue.
- **firectl**: stale. **Ignite** (weaveworks/ignite): archived Dec
 2023 - do not adopt.

## Unikernels on Firecracker

- **unikraft / kraftkit**: `fc` platform target supported.
- **Nanos/ops** (NanoVMs): boots on FC.
- **OSv**: supported guest.
- **urunc**: containerd runtime for unikernels across FC/QEMU/CH.

## Who runs Firecracker in production

- **AWS Lambda and Fargate** since 2018 (NSDI'20 paper).
- **Fly.io** Machines and Sprites sandboxes (Cloud Hypervisor for GPU
 machines).
- **Vercel Sandbox / Hive**: ~2.7M microVM deployments/day, public
 $1M escape bounty, FC + host-side network firewall on bare-metal
 EC2.
- **E2B**: managed FC sandboxes for AI agents, ~150 ms boot,
 pause/resume.
- **CodeSandbox SDK** (Together AI), **Deno Sandbox**, **Northflank**
 (FC/Kata/gVisor per workload), **Koyeb**, **actuated** (ephemeral
 GitHub Actions runners in FC microVMs).
- Not FC: Modal is gVisor, Cloudflare sandboxes are containers,
 Gitpod/Ona are containers via workspacekit.

## Decision guide

- Untrusted code, minimal surface, snapshots, scale: Firecracker.
- Need GPU, Windows, live migration, virtio-fs: Cloud Hypervisor.
- Lightweight macOS or GPU without passthrough: libkrun.
- Kubernetes pods needing VM isolation: Kata (prefer CH/QEMU
 hypervisor), or flintlock for microVM nodes.
- Fastest possible sandbox reuse: Firecracker diff snapshots +
 pooling, which is the E2B/Fly model.
