# Release history

Notable changes by version. Full changelog: CHANGELOG.md in
slackhq/nebula. Releases ship `nebula` and `nebula-cert` binaries for all
supported platforms, plus the `nebulaoss/nebula` distroless image.

## v1.11.x (2026)

- v1.11.0: logging switched from logrus to slog (breaking for parsers and
  embedders, levels print uppercase, timestamps always RFC3339Nano).
  inbound_action/outbound_action swap fixed. Windows WFP PERMIT filters and
  `private` network category by default. ICMP code 13 rejects. sshd
  profiling confined to `sshd.sandbox_dir`. `-` stdio for nebula-cert.
  config.yaml searched alongside config.yml. macOS rebind on network change
  (`listen.rebind_on_network_change`). Lighthouse records itself in its DNS.
  Many relay, DNS, lifecycle, and reload fixes. Built on Go 1.26.
- v1.11.1: IPv6 next-header classifier fix (firewall bypass). Message
  counter limit enforcement. ICMPv6 conntrack echo id fix.

## v1.10.x (2025-12 to 2026-02)

- v1.10.0: cert v2 ASN.1 format, IPv6 and multiple overlay addresses per
  host, `pki.initiating_version`. `listen.so_mark` on Linux. ECMP
  unsafe_routes. PKCS11 P256 keys. `default_local_cidr_any` defaults false
  and deprecated. yaml.v3. Encrypted CA passphrase via env var. Built on
  Go 1.25.
- v1.10.1: `listen.accept_recv_error` option.
- v1.10.2: panic fix for `use_system_route_table`.
- v1.10.3: P256 signature malleability blocklist bypass fix
  (GHSA-69x3-g4r3-p962).

## v1.9.x (2024-05 to 2025-10)

- v1.9.0: `local_range` removed, use `preferred_ranges`. Firewall
  local_cidr semantics changed, `default_local_cidr_any` added (true).
  Official Docker image `nebulaoss/nebula`. sshd trusted_cas and inline host
  keys. unsafe_routes reloadable. Go 1.22, needs Windows 10+.
- v1.9.5: v2 certs ignored gracefully (forward compat).
- v1.9.6: `tunnels.drop_inactive` + `inactivity_timeout`.
- v1.9.7: source IP spoofing fix (CVE-2025-62820).

## v1.8.x (2023-12 to 2024-01)

- v1.8.0: systemd notify support, Windows Registered IO (~50x throughput),
  NetBSD/OpenBSD support, `pki.disconnect_invalid` defaults true,
  `timers.requery_wait_duration`, hostmap memory refactor.
- v1.8.1: handshake deadlock fix, x/crypto update for CVE-2023-48795.

## v1.7.x (2023)

- v1.7.0: encrypted CA keys (AES-256-GCM + Argon2id, `-encrypt`,
  `-argon-*` flags). P256 curve and BoringCrypto support. Firewall
  `local_cidr` field. `unsafe_routes.install`. `tun.use_system_route_table`.
  `punchy.respond_delay`. `lighthouse.calculated_remotes`. Firewall reject
  action. DNS names in static_host_map auto-refresh. Rehandshake all
  tunnels on cert reload. Case-insensitive lighthouse DNS.
- v1.7.2: config reload freeze fix for static_host_map changes.

## v1.6.x (2022)

- v1.6.0: experimental relay support. `lighthouse.advertise_addrs` (manual
  ip:port reporting). `listen.send_recv_error`. punchy and lighthouse
  options hot reloadable. `routines` promoted from experimental. x509 config
  stanza removed.
- v1.6.1: reject underlay packets received from overlay IPs (unsafe routes
  confusion fix).

## v1.5.x (2021)

- v1.5.0: `pki.disconnect_invalid` added (default false). unsafe_routes
  `metric`. wintun driver on Windows replaces tap0901. `preferred_ranges`
  documented, `local_range` deprecated (it existed since v1.0.0).
  nebula-cert enforces IPv4. CGO_ENABLED=0 builds.

## v1.4.x (2021)

- v1.4.0: QR output in nebula-cert for mobile. `routines` experimental
  multi-queue. IPv6 underlay. ping works with `tun.disabled`. Big memory and
  CPU reductions.

## v1.1 to v1.3 (2020)

- v1.1.0: unsafe_routes added (Linux, macOS). `lighthouse.dns.host/port`.
  `-service` mode on macOS and Windows.
- v1.2.0: `remote_allow_list` and `local_allow_list`. unsafe_routes on
  Windows. Wireshark dissector. `punchy.punch`/`punchy.respond` naming
  settled. `punchy.delay`. `handshakes` options.
- v1.3.0: `stats.message_metrics`/`lighthouse_metrics`. `tun.disabled` for
  rootless lighthouses. `pki.blacklist` renamed to `pki.blocklist`.
  FreeBSD. Experimental library embedding.

## Build and packaging

- `make bin`, `bin-pkcs11`, `bin-boringcrypto`, `bin-fips140`, release
  targets per platform. Windows binaries signed since v1.11.
- Nightly builds: github.com/NebulaOSS/nebula-nightly, docker
  nebulaoss/nebula-nightly.
- v1.0.0 released 2019-11-19 alongside the Slack open sourcing.
