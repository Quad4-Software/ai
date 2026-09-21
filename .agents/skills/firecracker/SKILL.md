---
name: firecracker
description: >
  This skill covers Firecracker, the Rust microVM monitor (KVM-only)
  that powers AWS Lambda and Fargate, plus the microVM ecosystem
  (flintlock, libkrun, microvm.nix, Cloud Hypervisor, Kata, unikernels).
  Use it for microVM isolation vs containers, the jailer, snapshotting,
  the REST API boot sequence, vsock guest agents, MMDS metadata,
  tap-device networking, and where GPU or Windows guests force a
  different VMM.
---

## When to use this skill

- You are sandboxing untrusted or multi-tenant code beyond container
  isolation (CI runners, AI agent execution, user code).
- You are driving the Firecracker API, jailer, or snapshot flow.
- You are picking between Firecracker, Cloud Hypervisor, libkrun, Kata,
  QEMU microvm, or a unikernel.
- You are debugging tap networking, vsock, or snapshot restore.

## How to use

1. Read this file for the architecture and current version facts.
2. Load [references/api.md](references/api.md) for the boot sequence,
   jailer invocation, networking, vsock, MMDS, and snapshot calls.
3. Load [references/ecosystem.md](references/ecosystem.md) for the
   tool comparison and who-runs-what.
4. Upstream docs: https://github.com/firecracker-microvm/firecracker
   (docs/ dir is authoritative. swagger/firecracker.yaml is the API).

## Examples

- "Boot a microVM with a tap interface and a vsock agent."
- "Why won't firecracker start on this Azure VM?" (no nested KVM)
- "We need GPU in a sandbox - Firecracker or Cloud Hypervisor?"

# Firecracker

## Core facts

- Rust VMM on top of KVM only. One process per microVM: API thread +
  VMM thread + one thread per vCPU running KVM_RUN.
- Current **v1.17.0** (Sept 10, 2026). ~3-month release cadence, 6
  months support per minor. v1.16.x also supported.
- Minimal device model: virtio-net, virtio-block, virtio-vsock,
  virtio-rng, virtio-balloon, virtio-pmem, serial console. No USB, no
  display, no BIOS (direct kernel boot, Linux boot protocol + PVH).
  virtio-mmio default. Virtio-pci opt-in since v1.13
  (`--enable-pci`).
- SLAs: VMM boot ~8 CPU-ms, <=125 ms to guest init, <=5 MiB overhead,
  ~5 microVMs/sec/core. Max 32 vCPUs. Raw disk images only, no qcow2,
  no virtio-fs, no DHCP/DNS inside.
- Guests: Linux 5.10/6.1/6.18 validated, OSv unikernel, FreeBSD 14+
  amd64 via PVH. No Windows. Guest arch must equal host arch.
- Hosts: Linux with /dev/kvm, kernels 5.10/6.1/6.18. x86_64
  (Haswell+/Zen+) and aarch64 (Graviton2-5, Ampere).

## v1.13-v1.17 highlights

- v1.13: PCI/virtio-pci opt-in transport (first CVE in that path:
  CVE-2026-5747, OOB write - keep it off unless needed).
- v1.16: device hotplug developer preview (block/pmem/net, PCI only,
  guest must rescan), serial rate limiting, MTU advertisement.
- v1.17: virtio device reset, bzImage kernels on x86_64, transparent
  huge pages, `sync_snapshot_files` on snapshot create.

## Security model

Three containment layers: the KVM boundary, per-thread seccomp-BPF
filters (most-restrictive default. Do not pass `--no-seccomp` in
prod), and the **jailer** companion binary (chroot via pivot_root,
netns/pidns, cgroup v1/v2, uid/gid drop, rlimits, then execs the VMM).
Assume the guest is hostile. The API socket and host resources are
trusted. Production hardening checklist lives in docs/prod-host-setup.md.

This is why AI-agent sandboxes standardized on it: a container escape
compromises a shared kernel, a microVM escape has to cross KVM plus a
~50k-LOC memory-safe VMM. Vercel's Hive runs ~2.7M FC microVM
deployments/day behind a public $1M escape bounty. E2B, Fly.io, and
AWS Lambda/Fargate run it at larger scale still.

## Snapshotting

Pause -> `PUT /snapshot/create` (Full or Diff) -> `PUT
/snapshot/load` on a fresh process. Full snapshots resume directly,
diffs must merge into a base except diffs of booted VMs. Restore is
CPU-model sensitive (use cpu templates). VMGenID protects against
entropy reuse across restores. No live migration.

## Gotchas

- Needs /dev/kvm. Nested virt: GCP yes on select families, AWS on
  recent Intel families since ~Feb 2026 (else .metal), Azure basically
  no. Bare metal is the reliable path.
- No VFIO/GPU passthrough shipped. For GPU microVMs use Cloud
  Hypervisor (VFIO) or libkrun (virtio-gpu venus).
- Networking is DIY: tap devices, host NAT, no built-in DHCP. Push
  guest config via MMDS or a vsock agent.
- Balloon is pre-boot only. Memory overcommit is soft.
- Orchestration (images, taps, CNI, snapshot merge, pooling) is on
  you. Most consumers go through Kata, flintlock, or a hosted
  platform rather than raw API.

## Sources

- Repo and docs: https://github.com/firecracker-microvm/firecracker
- NSDI'20 paper (Lambda architecture): usenix.org NSDI 2020
- Releases: github.com/firecracker-microvm/firecracker/releases
