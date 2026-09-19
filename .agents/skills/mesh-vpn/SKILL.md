---
name: mesh-vpn
description: >
  This skill covers Tailscale and NetBird, the two WireGuard-based mesh
  overlay networks. Use it for architecture (control vs data plane), NAT
  traversal, subnet routing, exit nodes, ACLs/grants vs group policies,
  MagicDNS vs nameserver groups, self-hosting (headscale vs full
  self-hosted NetBird), and picking between them.
---

## When to use this skill

- You are choosing between Tailscale and NetBird, or deploying either.
- You need NAT traversal, DERP/relay, or peer-relay behavior.
- You are writing a tailnet policy file (grants/ACLs, tagOwners,
  autoApprovers) or NetBird access policies.
- You are setting up subnet routers, exit nodes, Funnel/Serve, or
  NetBird Networks.
- You are self-hosting the control plane (headscale or NetBird's stack).

## How to use

1. Read this file for the shared model and the decision table.
2. Load [references/tailscale.md](references/tailscale.md) for the
   tailnet policy file, Serve/Funnel, auth keys, and Kubernetes details.
3. Load [references/netbird.md](references/netbird.md) for the
   management/signal/relay architecture, Networks, setup keys, and
   self-hosting.
4. Fall back to https://tailscale.com/docs and https://docs.netbird.io
   for anything not covered here.

## Examples

- "Write a grants rule letting the ops group SSH into tagged servers."
- "Why do two nodes behind symmetric NAT relay instead of going direct?"
- "Set up a NetBird exit node with Auto Apply for the mobile group."
- "Compare Tailscale's control plane to self-hosted NetBird."
- "Configure a subnet router so the tailnet can reach a VPC CIDR."

# Tailscale and NetBird

Both are WireGuard meshes: every node gets an overlay IP, peer-to-peer
encrypted tunnels are preferred, and a relay is the fallback when direct
UDP fails. They differ in who runs the control plane and how policy is
expressed.

| | Tailscale | NetBird |
|---|---|---|
| Latest (Sept 2026) | v1.102.3 (Aug 2026) | v0.77.1 (Aug 2026) |
| Control plane | Hosted SaaS, closed source | Open source, self-hostable. Hosted cloud also available |
| Self-hosted control | headscale (community, single tailnet, hobbyist scope) | Full stack: management + signal + relay + dashboard, all BSD-3 |
| Relay | DERP (HTTPS) + peer relays (v1.86+) | Own relay (QUIC + WebSocket), replaced coturn since v0.29 |
| Policy | Central HuJSON policy file: grants, acls, tagOwners, autoApprovers, ssh, tests | Dashboard/API group-based policies: directional, ports, posture checks |
| DNS | MagicDNS @100.100.100.100, `*.ts.net`, search domains, split DNS, DoH | Peer DNS domain + nameserver groups with match domains, custom zones |
| Public ingress | Funnel (443/8443/10000, TLS-only) + Serve (tailnet-only) | None built in |
| SSH | Tailscale SSH rules in policy file | Built-in SSH on port 22022, browser SSH/RDP |
| K8s | First-class operator (API proxy, L3/L7 ingress, egress) | Client runs in-cluster, no operator |
| Free tier | 6 users, unlimited user devices, 50 tagged resources | 5 users, 100 machines |

## Shared architecture

- **Data plane:** WireGuard P2P. Direct UDP preferred. Relay fallback
  when NAT blocks direct paths. Relayed traffic stays end-to-end
  encrypted. Relays cannot decrypt.
- **Control plane:** distributes WireGuard public keys, coordinates NAT
  traversal, and pushes policy. Tailscale's is hosted and closed.
  NetBird's is four open-source components (management, signal, relay,
  dashboard) deployable as one `netbird-server` container.
- **NAT traversal order:** direct UDP -> peer relay (if configured) ->
  relay fallback. Tailscale listens on UDP 41641 by default and tries
  UPnP/NAT-PMP/PCP port mapping. NetBird races QUIC and WebSocket
  transports to its relay.

## Policy models

- **Tailscale** keeps one HuJSON file per tailnet. `grants` is the
  recommended syntax (implied `accept`, separates `dst` from `ip`, adds
  `app` capabilities and `via` route filters). `acls` coexists and still
  works. `tagOwners` removes user identity from tagged devices and
  disables their key expiry by default. `autoApprovers` lets routes and
  exit nodes bypass admin approval. `tests`/`sshTests` validate before
  the file saves.
- **NetBird** is deny-by-default and expresses policy as
  source-group -> destination-group rules with protocol/ports,
  bidirectional or unidirectional flags, and optional posture checks
  (client version, OS, geo, CIDR, running processes). There is no
  central file. Everything lives in the management DB and is edited
  through the
  dashboard or API.

## Routing beyond the mesh

- **Tailscale subnet routers** advertise CIDRs (`--advertise-routes`),
  need admin or `autoApprovers` approval, and clients accept them
  (`--accept-routes` on Linux. Default on desktop/mobile).
- **NetBird Networks** (v0.35.0+) map an environment to typed resources
  (single IP, CIDR, domain, wildcard domain), resource groups, routing
  peers, and inline access policies. The older network-routes feature is
  deprecated. Everything except exit nodes moved to Networks. A route
  without ACL groups bypasses access control entirely, so legacy routes
  need review.
- **Exit nodes** work the same way on both: `0.0.0.0/0` through a
  routing peer with masquerade. Tailscale adds Mullvad exit nodes and
  app connectors (domain-based routing to SaaS apps). NetBird adds an
  Auto Apply flag (v0.55.0+) that enables the exit node for whole
  distribution groups without client selection.

## DNS

- **Tailscale MagicDNS** answers at 100.100.100.100 (and
  `fd7a:115c:a1e0::53`) for `machine.tailnet.ts.net`. It adds a tailnet
  search domain, supports custom upstreams and split DNS per domain,
  and can upgrade to DoH for supported resolvers.
- **NetBird** gives each peer a name under `netbird.cloud` (or
  `netbird.selfhosted`), supports nameserver groups scoped to match
  domains or as primaries, and Custom Zones for private records that
  override nameservers. Match domains auto-cover subdomains. The `*.x`
  wildcard syntax is not supported.

## Self-hosting

- **Tailscale:** the coordination server is closed. `headscale` is the
  community reimplementation (v0.29.3, July 2026), deliberately scoped to
  a single tailnet for hobbyists and small orgs. Minimum supported
  client v1.80.0. The client and `derper` are open. You can also run a
  custom DERP relay (region IDs 900-999 reserved) with caveats: no node
  sharing across tailnets, no geo-steering, no Mullvad exits.
- **NetBird:** the whole control plane self-hosts. Since v0.62.0 the
  quickstart uses an embedded Dex-based IdP (local users + external
  OIDC). Zitadel is no longer required. The `netbird-server` container
  bundles management, signal, relay, and embedded STUN behind a reverse
  proxy, needing TCP 80/443 + UDP 3478. Unified `config.yaml` replaced
  the old `management.json`/`setup.env`/`relay.env` split.

## Choosing

- Pick **Tailscale** when a managed control plane is acceptable and you
  want the most polished client surface, Funnel for public ingress, the
  k8s operator, Tailscale SSH, or Mullvad exits. GitOps-friendly via
  the policy file API.
- Pick **NetBird** when the control plane must be self-hosted without
  giving up features, when posture checks or group-based policies fit
  the security model better than a flat file, or when the Networks
  resource model maps cleanly to existing VPC/LAN layouts.
- Pick **headscale** only for single-tailnet hobbyist scale where you
  accept the community-maintenance risk and the narrower feature set.
