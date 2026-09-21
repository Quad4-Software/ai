# Firecracker API quick reference

All calls go over a unix socket:
`firecracker --api-sock /tmp/fc.sock` then
`curl --unix-socket /tmp/fc.sock -X PUT -H 'Content-Type:
application/json' -d '<body>' http://localhost/<path>`.

## Boot sequence

```
PUT /boot-source           {kernel_image_path, boot_args, initrd_path?}
PUT /drives/<id>           {drive_id, path_on_host, is_root_device,
                            is_read_only, rate_limiter?, io_engine:
                            Sync|IoUring}
PUT /network-interfaces/<id> {iface_id, guest_mac, host_dev_name,
                            rx_rate_limiter?, tx_rate_limiter?, mtu?}
PUT /vsock                 {guest_cid: 3, uds_path}
PUT /machine-config        {vcpu_count: 1..32, mem_size_mib, smt,
                            cpu_template, track_dirty_pages,
                            huge_pages: None|2M|Transparent}
PUT /entropy               {rate_limiter?}     # virtio-rng
PUT /balloon               {amount_mib, deflate_on_oom, ...}  # pre-boot only
PUT /mmds / PUT /mmds/config  metadata + V1/V2 mode
PUT /logger / PUT /metrics / PUT /serial
PUT /actions               {action_type: "InstanceStart"}
PATCH /vm                  {state: Paused|Resumed}
```

Alternative: `firecracker --config-file vm.json` boots the whole
config with no API control after.

boot_args typically: `console=ttyS0 reboot=k panic=1 pci=off` plus
static guest IP via `ip=172.16.0.2::172.16.0.1:255.255.255.0::eth0:off`
when needed.

Kernel: uncompressed `vmlinux` ELF (x86 also takes `bzImage` since
v1.17. Aarch64 `Image`). Rootfs: raw ext4 image, commonly built from
an Alpine minirootfs or a Docker image export.

## Jailer

```bash
jailer --id <uuid> --exec-file /usr/bin/firecracker \
  --uid 30000 --gid 30000 --netns /var/run/netns/ns1 \
  --cgroup-version 2 --cgroup cpu.max="50000 100000" \
  --daemonize -- --api-sock /run/api.socket
```

Pair it with the statically linked (musl) firecracker of the same
version. It chroots into `<chroot-base>/firecracker/<id>/root`,
applies cgroups/rlimits, drops uid/gid, then execs. CVE-2026-1386
(jailer symlink-follow host file overwrite) hit <=1.13.1/1.14.0 -
keep it current.

## Networking

Firecracker provides no DHCP or NAT. Host side:

```bash
ip tuntap add dev tap0 mode tap
ip addr add 172.16.0.1/24 dev tap0
ip link set tap0 up
sysctl -w net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

Guest gets a static IP via kernel cmdline or config delivered over
MMDS/vsock. firecracker-containerd wires CNI via tc-redirect-tap if
you want CNI semantics.

## vsock guest-agent pattern

Guest listens on AF_VSOCK. Host connects to the configured UDS path,
writes `CONNECT <port>\n`, reads `OK <port>\n`, then it is a
transparent byte stream (typically HTTP to an in-guest agent).
Guest-to-host connections map to `uds_path_<port>` listeners. CID 2
is the host. Since v1.16 the vsock UDS path can be overridden on
snapshot restore, avoiding CID/path collisions when restoring one
snapshot into many sandboxes.

## MMDS

In-guest metadata via link-local 169.254.169.254 (embedded HTTP
stack, "Dumbo"). `PUT /mmds/config` `{network_interfaces, version:
V1|V2, ipv4_address}`. V2 mirrors IMDSv2 session tokens. Content via
PUT/PATCH/GET `/mmds`. Default cap 51200 bytes.

## Snapshots

```
PATCH /vm                {state: "Paused"}
PUT /snapshot/create     {snapshot_type: Full|Diff, snapshot_path,
                          mem_file_path, sync_snapshot_files: true,
                          version}
PUT /snapshot/load       {snapshot_path, mem_file_path,
                          enable_diff_snapshots, resume_vm,
                          huge_pages}
```

`track_dirty_pages` must be set at boot (or `enable_diff_snapshots`
on load) for Diff snapshots. Restores are CPU-model sensitive. Use
cpu templates for cross-host restore.

## Rate limiters

Token buckets (ops/sec + bandwidth) per virtio-net rx/tx and per
virtio-block drive, plus optional limiters on serial output,
virtio-rng, and virtio-pmem flush (v1.16). Set them per untrusted
tenant to keep one VM from starving the host.
