# Firewall

## Basics

- Per-host firewall on the overlay interface. Default: deny ALL inbound and
  outbound. Rules are allow-only, there are no explicit deny rules.
- A common baseline: allow all outbound, allow inbound ICMP for ping.
- Rules are evaluated per packet against the peer's certificate fields, so
  groups in certs are the main access-control mechanism.

## Rule fields

- `port`: `0` or `any` for any, a single port `80`, a range `200-901`, or
  `fragment` to match second and later fragments (no port available).
- `proto`: `any`, `tcp`, `udp`, `icmp`. Port is optional on icmp rules since
  v1.11.
- `ca_name`: issuing CA name. `ca_sha`: issuing CA shasum.
- `host`: `any` or a literal cert name.
- `group`: `any` or one group name.
- `groups`: list, ANDed together, the cert must contain every listed group.
- `cidr`: remote CIDR. `0.0.0.0/0` any IPv4, `::/0` any IPv6, `any` both.
- `local_cidr`: local side CIDR, used to scope rules to unsafe_routes
  destinations. Default covers only the cert's own overlay networks.

## Evaluation logic

Since v1.9.0 a rule matches when:

```
port AND proto AND (ca_sha OR ca_name) AND (host OR group OR groups OR cidr) AND local_cidr
```

Before v1.9.0 the last term was `(host OR group OR groups OR cidr OR
local_cidr)`, so a local_cidr match alone used to pass identity checks. On
v1.9+ write `local_cidr` rules with the intended host or group constraints
too.

## local_cidr and default_local_cidr_any

- `default_local_cidr_any: true` made rules without `local_cidr` apply to
  unsafe_routes destinations as well as the local host. Introduced in v1.9.0
  defaulted true, defaulted false in v1.10.0, now deprecated.
- With the v1.10 default (false), a rule without `local_cidr` applies only
  to the local host. To permit traffic toward unsafe_routes destinations,
  write rules with explicit `local_cidr` matching those routes.

## Actions and conntrack

- `firewall.outbound_action` / `inbound_action`: `drop` (silent, default) or
  `reject` (TCP gets RST, others get ICMP code 13 administratively
  prohibited since v1.11, code 3 before).
- These two settings were applied to the opposite direction until v1.11.0
  fixed the swap. If upgrading across v1.11 with either set, swap them.
- `conntrack`: `tcp_timeout` 12m, `udp_timeout` 3m, `default_timeout` 10m.
  ICMP is connection-tracked since v1.11.

## Examples

```yml
firewall:
  outbound:
    - port: any
      proto: any
      host: any
  inbound:
    - port: any
      proto: icmp
      host: any
    # SSH only from certs carrying BOTH laptop and ops groups
    - port: 22
      proto: tcp
      groups: [laptop, ops]
    # web traffic from one group
    - port: 443
      proto: tcp
      group: user-endpoint
    # lighthouse DNS for everyone
    - port: 53
      proto: udp
      group: any
    # allow reaching an unsafe_routes subnet through this host
    - port: any
      proto: any
      local_cidr: 192.168.86.0/24
      group: trusted
```

## Gotchas

- Since v1.10, a rule containing `any` that negates a more restrictive
  filter logs a warning.
- Rules are reloadable via SIGHUP.
- v1.11.1: IPv6 packets whose next header is an unparsed protocol (SCTP,
  GRE, IP-in-IP) are now classified as that protocol with no ports, closing
  a bypass where crafted payloads matched tcp/udp rules. If the overlay
  carries those protocols they now need `proto: any`.
