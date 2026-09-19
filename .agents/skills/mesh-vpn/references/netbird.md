# NetBird reference

- **Latest:** v0.77.1 (Aug 2026). All components BSD-3 open source.
- Four logical components: **Management** (control plane, gRPC + HTTP
  API, SQLite default / PostgreSQL / MySQL), **Signal** (WebRTC-style
  ICE-candidate exchange. Encrypted, stores nothing), **Relay** (own
  relay since v0.29, QUIC primary + WebSocket/TCP fallback raced in
  parallel. Coturn is legacy), **Dashboard** (web UI). The client agent
  is WireGuard.

## Deployment

- **Quickstart:** `export NETBIRD_DOMAIN=...` then run
  `getting-started.sh` from the repo's `infrastructure_files/`. Needs a
  Linux VM (1 CPU/2 GB), Docker + compose v2, jq, curl, a public domain,
  and open **TCP 80, 443 + UDP 3478**. Generates `docker-compose.yml`,
  `config.yaml`, and `dashboard.env`.
- **`netbird-server` container** (v0.65+) bundles management + signal +
  relay + embedded STUN. The old five-container split (dashboard,
  management, signal, relay, coturn) is deprecated. Traefik ships as
  the built-in proxy. Nginx, Caddy, and Nginx Proxy Manager work as
  external frontends.
- **Ports since v0.29:** TCP 80+443 (dashboard, management API/gRPC,
  signal gRPC, relay WebSocket via HTTP/2 negotiation) + UDP 3478
  (STUN). The legacy ports (33073, 10000, 33080 TCP. 3478 + 49152-65535
  UDP) only matter for pre-v0.29 clients.
- **IdP:** since v0.62.0 an embedded Dex-based IdP ships in management.
  Local users plus external OIDC providers: Google, Microsoft/Entra,
  Okta, Zitadel, Keycloak, Authentik, PocketID, generic OIDC. Zitadel
  remains supported standalone. `netbird-idp-migrate` migrates an
  external IdP to the embedded one (requires >=v0.67.2 target).
- **Config:** unified `config.yaml` replaced the old `management.json` +
  `setup.env` + `relay.env` split.

## Concepts

- **Peers and groups:** everything is a peer. Groups are the policy and
  DNS unit. `All` is the built-in peer group. Network resources need
  explicit policies and cannot use `All`.
- **Setup keys** register machines non-interactively: one-off or
  reusable (usage limit), optional expiry (default prefilled 7 days,
  empty = never), **auto-assign groups**, **ephemeral peers**
  (auto-removed after ~10 min offline), `allow_extra_dns_labels`. Pass
  via `--setup-key` or `NB_SETUP_KEY`.
- **Access policies** are deny-by-default: source group -> destination
  group, protocol `ALL`/`ICMP`/`TCP`/`UDP`, ports and ranges (ranges
  since v0.48), bidirectional or unidirectional, optional posture
  checks.
- **Posture checks:** client version, OS type/version, geolocation
  allow/block, source CIDR, running processes.

## Networks (v0.35.0+)

Networks model an environment (LAN, VPC) as typed resources:

- **Resources:** single IP (/32), CIDR, domain, wildcard domain
  (`*.x.y` matches subdomains only).
- **Resource groups** bundle resources for policy.
- **Routing peers** are the exit points into the environment. Domain
  resources resolve through them.
- **Access policies** live inline on the network.

The legacy network-routes feature is **deprecated**. Every use case
except exit nodes moved to Networks. A route without ACL groups bypasses
access control entirely, so existing routes need review after migrating.

## Exit nodes

`0.0.0.0/0` through a routing peer with masquerade. IPv6 is blocked and
a `::/0` route is added when the exit node is enabled. Minimum policy:
source -> routing peer over ICMP. **Auto Apply** (v0.55.0+) enables the
exit node for whole distribution groups without user selection. With it
off, users pick the exit node in the client.

## DNS

- Peer DNS domain defaults to `netbird.cloud` (hosted) or
  `netbird.selfhosted` (self-hosted).
- **Nameserver groups:** primary (empty match domains) or match-domain
  scoped. A match domain auto-covers all subdomains. The `*.x` wildcard
  syntax is not supported. Per-distribution-group assignment, and a
  search-domain option on match domains since v0.24.
- **Custom Zones:** NetBird-hosted private records that take precedence
  over nameservers.
- The local resolver on each peer routes queries, so match-domain
  traffic stays inside the mesh.

## IdP sync and users

- **IdP group/user sync:** Entra ID (Graph API or SCIM), Okta (OIDC +
  SCIM), Google Workspace (Admin SDK), generic SCIM. Synced groups are
  usable in policies and DNS. One sync integration at a time.
- **User invites:** direct (email invite via dashboard/API, ~3-day
  expiry, auto-groups), indirect (verified-domain auto-join), or IdP
  sync. Self-hosted embedded IdP supports secure invite links and local
  users.

## NetBird SSH

Built-in SSH server in the client on **port 22022** (off by default).
Enable per-peer. A policy allowing TCP 22 to the peer auto-permits
22022 on management (v0.60+). Access via `netbird ssh` or native OpenSSH
(interception writes `ssh_config.d` host blocks). Browser SSH and RDP
are also exposed per the dashboard.

## Client

- Platforms: Linux, macOS, Windows, Android, Android TV, iOS, Apple TV,
  FreeBSD, plus Docker/rootless netstack images.
- CLI: `netbird up|down|status|login|ssh|expose|networks|profile|debug|
  service|version`.
- Kernel WireGuard on Linux, wireguard-go userspace elsewhere.
  `NB_WG_KERNEL_DISABLED` forces userspace on Linux, and
  `NB_USE_NETSTACK_MODE` uses gVisor netstack for TUN-less containers.
- `netbird networks` selects routes/networks client-side and resolves
  overlapping routes. Exit-node selection lives in the client menu when
  Auto Apply is off.
