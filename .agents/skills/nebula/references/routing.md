# Routing and the tun device

## tun options

- `tun.disabled: true` runs nebula with no tun device. Used for
  lighthouse-only or relay-only nodes, and lets them run without root.
  Such a node cannot exchange overlay IP traffic.
- `tun.dev`: interface name. macOS must be `utun[0-9]+`, FreeBSD must be
  `tun[0-9]+`, Linux defaults if unset.
- `tun.mtu` default 1300, safe for internet paths. Reloadable on Linux.
- `tun.routes`: per-destination MTU overrides for paths that support larger
  frames.
- `tun.tx_queue` default 500. Raise if transmit drops appear on the tun.
- `tun.drop_local_broadcast`, `tun.drop_multicast`: drop broadcast or
  multicast packets instead of forwarding them.
- `routines` (top level, Linux only): thread pairs on tun and UDP queues.
  Above 1 sets IFF_MULTI_QUEUE and SO_REUSEPORT. Keep below half the cores.

## unsafe_routes

Route a real subnet through a nebula host for devices that cannot run
nebula, such as printers or embedded gear.

```yml
tun:
  unsafe_routes:
    - route: 172.16.1.0/24     # real subnet reachable via the gateway host
      via: 192.168.100.99      # nebula IP of the gateway host
      mtu: 1300                # defaults to tun.mtu
      metric: 100
      install: true            # install into the system route table
```

- The `via` host's certificate MUST list the route in its `subnets` (v1) or
  `unsafeNetworks` (v2) field or it silently refuses the traffic. Sign with
  `nebula-cert sign ... -subnets '172.16.1.0/24'`.
- The CA itself must permit the subnets: a CA created with `-subnets`
  restricts what it can sign. Check with `nebula-cert print -path ca.crt`,
  an empty Subnets set means unrestricted.
- Inbound firewall rules on the gateway host need `local_cidr` covering the
  routed subnet since v1.9/1.10 (see references/firewall.md).
- The gateway host still needs OS-level forwarding: `net.ipv4.ip_forward=1`,
  return routing or SNAT on the LAN side.
- Reloadable via SIGHUP since v1.9.

## ECMP for unsafe_routes (v1.10)

Multiple gateways for one route with optional weights:

```yml
unsafe_routes:
  - route: 192.168.100.0/24
    via:
      - gateway: 10.0.0.1
        weight: 10             # default weight 1
      - gateway: 10.0.0.2
        weight: 5
```

- Hash-threshold mapping per RFC 2992 distributes flows by packet hash.
- Unreachable gateways fail over automatically, balance is uneven until
  recovery.

## System route table (Linux, v1.7+)

- `tun.use_system_route_table: true` lets nebula read unsafe routes from the
  system route table (gateway routes) instead of config. Built for dynamic
  routing like BGP into the overlay.
- `tun.use_system_route_table_buffer_size` (v1.10) sizes the netlink read
  buffer for massive route updates. 0 = system default.
- v1.10 fixed high CPU with thousands of routes and a panic on missing
  destinations.

## Full-tunnel 0.0.0.0/0 via so_mark (Linux, v1.10)

`listen.so_mark` marks nebula's underlay packets so policy routing can keep
them out of the overlay default route:

```yml
listen:
  so_mark: 4242
tun:
  unsafe_routes:
    - route: 0.0.0.0/0
      via: 192.168.100.99
```

```sh
ip rule add not from all fwmark 4242 lookup 4242
ip rule add from all lookup main suppress_prefixlength 0
ip route add default dev nebula1 via <gateway nebula IP> table 4242
```

## Platform notes

- Windows: v1.11 installs WFP PERMIT filters for the adapter and listener by
  default, sitting below Windows Defender Firewall. `tun.windows_bypass_wdf`
  and `listen.windows_bypass_wdf` set to false restore WDF control. The
  adapter gets the `private` network category by default,
  `tun.network_category: unset` keeps the old behavior. Unsafe routes
  install as link routes since v1.11.
- macOS: `listen.rebind_on_network_change` (default true, v1.11) rebinds and
  re-queries lighthouses when the underlay changes.
