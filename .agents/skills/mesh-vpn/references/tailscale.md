# Tailscale reference

- **Latest:** v1.102.3 (Aug 2026). Highlights: TS-2026-011 security fix
  (host-scoped IPv4 on 4via6), large-tailnet memory reductions,
  `TS_BOOT_TIMEOUT` for containers, k8s operator peer-relay fixes.
- Client and `cmd/derper` are BSD-3 open source. The coordination server
  is hosted and closed. `headscale` (v0.29.3, July 2026) is the
  community single-tailnet reimplementation.

## Tailnet policy file

HuJSON (comments and trailing commas allowed), edited in the admin
console, through the API, or via GitOps.

Sections: `grants`, `acls`, `ssh`, `autoApprovers` (`routes`,
`exitNode`, app connectors), `nodeAttrs`, `postures`, `tagOwners`,
`groups`, `hosts`, `ipsets`, `tests`, `sshTests`, `derpMap`, DNS config.

### grants vs acls

`grants` is the recommended syntax. It implies `accept`, separates `dst`
(the target) from `ip` (ports/protocols), and adds:

- `app`: application-layer capabilities (e.g.
  `tailscale.com/cap/tailsql`).
- `via`: route filtering through specific routers.
- `srcPosture`: posture requirements on the source.

`acls` coexists in the same file and continues to work. New policy
should use grants.

```json
{
  "grants": [
    {
      "src": ["group:ops"],
      "dst": ["tag:server"],
      "ip": ["tcp:22", "icmp"]
    }
  ],
  "ssh": [
    {
      "action": "accept",
      "src": ["group:ops"],
      "dst": ["tag:server"],
      "users": ["autogroup:nonroot", "root"]
    }
  ],
  "tagOwners": {
    "tag:server": ["group:ops"]
  },
  "autoApprovers": {
    "routes": {
      "10.0.0.0/8": ["tag:router"]
    },
    "exitNode": ["tag:router"]
  },
  "nodeAttrs": [
    {"target": ["tag:funnel-host"], "attr": ["funnel"]}
  ]
}
```

- `tagOwners` defines `tag:*` names and who may assign them. Tagging a
  device removes its user identity. Tagged devices get key-expiry
  disabled by default.
- `autoApprovers.routes`/`exitNode` lets listed users/groups/tags
  advertise routes or exit nodes without manual admin approval.
- `ssh` rules control Tailscale SSH: `accept` (direct) vs `check`
  (re-verification within `checkPeriod`).
- `tests`/`sshTests` are assertions the policy engine validates before
  saving.

## Connectivity

- Default listen port UDP 41641. Order: direct UDP -> peer relay ->
  DERP. `tailscale netcheck` and `tailscale status` show the active
  path.
- **DERP** relays are Tailscale-run HTTPS relays across ~30 regions
  (Sydney, Sao Paulo, Toronto, Helsinki, Paris, Frankfurt, Nuremberg,
  Hong Kong, Bengaluru, Tokyo, Nairobi, Amsterdam, Warsaw, Singapore,
  Johannesburg, Madrid, Dubai, London, plus 10 US cities). Traffic stays
  E2E-encrypted. The relay cannot decrypt. Full map at
  `https://controlplane.tailscale.com/derpmap/default`. `derpMap` in the
  policy file disables regions or adds custom ones.
- **Peer relays** (GA since Oct 2025, client v1.86+) let any tailnet
  node relay via `--relay-server-port`. Preferred over DERP when
  available. Controlled through grants.
- **Custom DERP** is possible with `cmd/derper` (`go install
  tailscale.com/cmd/derper@latest`). Caveats: no node sharing or
  cross-tailnet, no control-plane geo-steering, Mullvad exits do not
  work, and region IDs 900-999 are reserved for user DERP.
- UPnP, NAT-PMP, and PCP port mapping are all attempted
  (`net/portmapper`). "Easy NAT" (full-cone, port mapping, hairpin,
  consistent mapping) yields direct paths. Symmetric/hard NAT falls to
  relay.

## Serve and Funnel

- `tailscale serve` reverse-proxies a local service to the tailnet with
  an auto-provisioned Let's Encrypt cert for `node.tailnet.ts.net` via
  DNS-01 (the `*.ts.net` TXT record stays under Tailscale's control. The
  private key stays local). Supports HTTP, TCP, file, and text backends.
- `tailscale funnel` exposes the service to the public internet through
  Tailscale's Funnel ingress nodes. Restrictions: ports 443, 8443,
  10000 only. TLS only. Requires MagicDNS + HTTPS certs enabled + the
  `funnel` node attribute in `nodeAttrs`. Bandwidth-limited. A port
  cannot be serve-private and funnel-public at once (last command wins).
  **Funnel bypasses tailnet auth on the listener**, so anything exposed
  needs its own application auth.
- CLI redesigned in v1.52: `serve status`, `serve --bg`, `funnel`
  subcommands replace the old flag style.

## Keys and identity

- **Auth keys** (pre-auth keys): one-off or reusable, 1-90 day expiry,
  can be tagged, ephemeral (auto-delete when offline), and
  pre-authorized.
- **OAuth clients** (trust credentials): client ID + secret issue
  scoped API tokens (`devices:core`, `auth_keys`, `dns:read`, `all`,
  `all:read`). The `tags` parameter restricts which device tags the
  token can assign. Preferred over long-lived API tokens. Also supports
  federated OIDC workload identities.
- **Node key expiry:** default 180 days per tailnet (configurable
  1-180). Disable per-device on the Machines page. Tagged devices
  default to no expiry.

## Other features

- **Taildrop:** `tailscale file cp`/`get` P2P file transfer between your
  own devices. Alpha. Opt-in per tailnet via the "Send Files" toggle.
- **tsnet:** Go library embedding a Tailscale node in-process (own
  tailnet IP, cert management). Import `tailscale.com/tsnet`.
- **Tailscale SSH:** built-in SSH server. Rules in the `ssh` policy
  section. `check` mode re-verifies within `checkPeriod`.
- **Tailnet Lock:** disables the coordination server's ability to add
  nodes without a signature from a trusted node.
- **Mullvad exit nodes:** paid add-on using Mullvad VPN servers. Enabled
  in the admin console or via `nodeAttrs` `"mullvad"` (mutually
  exclusive paths).
- **App connectors:** route tailnet traffic to SaaS/cloud apps by
  domain. A connector device learns routes via DNS. Preset apps are
  auto-approved, custom routes need `autoApprovers`.
- **Kubernetes operator:** API-server proxy (Tailscale-identity
  impersonation into k8s RBAC, or noauth. HA `ProxyGroup`), L3 ingress
  (Service annotation/`loadBalancerClass: tailscale`), L7 ingress
  (`Ingress` + `ingressClassName: tailscale` via serve), L3 egress
  (`ExternalName` + `tailscale.com/tailnet-fqdn` or
  `tailnet-target-ip`).

## MagicDNS

- Stub resolver at 100.100.100.100 (Quad100) and `fd7a:115c:a1e0::53`
  answers authoritatively for `machine.tailnet.ts.net` and adds the
  tailnet search domain.
- Custom nameservers and split DNS per domain. DoH upgrade for
  supported upstreams. The old `beta.tailscale.net` suffix retired Sept
  2024.
