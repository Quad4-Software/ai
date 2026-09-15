---
name: nebula
description: >
  This skill covers Nebula, the open source mutually authenticated
  peer-to-peer overlay network originally built at Slack and maintained by
  Defined Networking (github.com/slackhq/nebula). Use when you need Nebula
  config reference details, nebula-cert PKI operations, firewall rules,
  lighthouse and relay setup, NAT traversal behavior, unsafe_routes, or
  release history.
---

## When to use this skill

- You are configuring or debugging a Nebula overlay network.
- You need details on lighthouses, relays, hole punching, or NAT behavior.
- You are working with nebula-cert, CA rotation, or certificate versions.
- You are building a tool that reads or manages Nebula state.

## How to use

1. Start from this file for the mental model and the map of reference docs.
2. Load a topic file from references/ when you need field-level detail.
3. Fall back to the official docs at https://nebula.defined.net/docs/ for
   anything not covered here. Docs source: github.com/DefinedNet/nebula-docs.

## Examples

- "Explain the difference between static_host_map and lighthouse discovery."
- "Write a firewall rule allowing SSH only from hosts in the ops group."
- "Walk me through rotating a Nebula CA without downtime."
- "Why do two hosts behind CGNAT need a relay?"

# Nebula Overlay Networking

## Core concepts

- **Nebula** is a self-hosted, mutually authenticated, peer-to-peer layer 3
  overlay network written in Go. Slack open-sourced it in 2019 after years of
  internal use. It still runs Slack's production overlay of 50k+ hosts. The
  creators founded Defined Networking in 2020 to maintain it. Latest stable
  is v1.11.x as of September 2026.
- **Crypto.** Handshakes use the Noise Protocol Framework (Noise IXpsk0
  pattern). Key exchange is Curve25519 ECDH by default, NIST P-256 supported
  for compliance. Transport cipher is AES-256-GCM by default, or
  ChaCha20-Poly1305 via the `cipher` option. The cipher must match on every
  host and lighthouse. FIPS 140 builds exist via the boringcrypto and
  fips140 build tags.
- **Identity is a certificate.** Every host has a Nebula cert signed by a CA
  created with `nebula-cert ca`. The cert pins the host name, overlay
  networks (IPs), group membership, unsafe route subnets, and validity
  window. Hosts cannot modify their own cert without invalidating it.
- **Lighthouses** are hosts on stable public addresses that record where
  every host was last reachable and assist UDP hole punching. They relay no
  overlay traffic. Every host must list every lighthouse in
  `static_host_map`, because the discovery service itself cannot be
  discovered.
- **Relays** (v1.6+) forward encrypted traffic between peers that cannot
  punch a direct tunnel, for example symmetric NAT to symmetric NAT. Relays
  see routing metadata only, never plaintext.
- **Transport.** All Nebula traffic rides a single UDP socket, default port
  4242 on lighthouses, port 0 (random) recommended for roaming hosts. There
  is no TCP fallback. Overlay traffic flows through a TUN device with default
  MTU 1300.
- **Firewall.** A per-host groups-based firewall filters overlay traffic.
  Default posture is deny all inbound and outbound until rules allow it.
- **Config.** One YAML file per host. Many options are reloadable with
  SIGHUP without tearing down tunnels. `nebula -test -config config.yml`
  validates a config. Both `config.yml` and `config.yaml` are searched.
- **Platforms.** Linux, macOS, Windows, FreeBSD, OpenBSD, iOS, Android.
  Distroless Docker image `nebulaoss/nebula`. Distro packages exist for
  Debian, Fedora, Arch, Alpine, and Homebrew. Managed Nebula from Defined
  Networking automates PKI, lighthouses, and SSO.

## Reference files

- [references/config.md](references/config.md): every config section with
  key options, defaults, and version markers.
- [references/pki.md](references/pki.md): nebula-cert commands, cert fields,
  CA ops, key encryption, PKCS11, cert v2 and IPv6 overlay migration.
- [references/discovery.md](references/discovery.md): static_host_map,
  lighthouse reports, hole punching, NAT types, punchy, relays,
  preferred_ranges.
- [references/firewall.md](references/firewall.md): rule fields, evaluation
  logic, local_cidr, conntrack, drop vs reject.
- [references/routing.md](references/routing.md): tun options,
  unsafe_routes, ECMP, system route table, so_mark.
- [references/ops.md](references/ops.md): sshd debug console, logging,
  stats, lighthouse DNS, non-root operation, signals.
- [references/security.md](references/security.md): trust model, security
  bulletins and CVEs, hardening notes.
- [references/versions.md](references/versions.md): notable changes by
  release.

## Sources

- Docs: https://nebula.defined.net/docs/
- Code, releases, changelog: https://github.com/slackhq/nebula
- Slack engineering announcement (2019):
  https://slack.engineering/introducing-nebula-the-open-source-global-overlay-network-from-slack/
- Defined Networking blog: https://www.defined.net/blog
- Community: NebulaOSS Slack workspace, GitHub Discussions
