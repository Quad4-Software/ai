# Discovery, NAT traversal, and relays

## Two discovery paths

1. **Static path**: `static_host_map` maps a peer nebula IP to routable
   `ip:port` or `name:port` entries. Dialed directly, no lighthouse needed,
   works during lighthouse outages. Cost: the entry must exist in every
   dialing peer's config and goes stale if the address changes.
2. **Dynamic path**: hosts report to lighthouses every `lighthouse.interval`
   seconds (default 10). A report carries self-observed interface addresses
   (filtered by `lighthouse.local_allow_list`) plus `advertise_addrs`. The
   lighthouse also records the source address it sees on report packets,
   which is the NAT'd public address peers can actually dial.

- Every host must list every lighthouse in `static_host_map`. The discovery
  service cannot discover itself.
- Lighthouses have empty `lighthouse.hosts` and usually empty
  `static_host_map`. Multiple lighthouses do not need to know each other.
- `lighthouse.advertise_addrs` on a host publishes addresses nebula cannot
  observe: forwarded ports, extra internet paths. Port `0` substitutes the
  real listen port. Propagates to peers automatically, unlike
  static_host_map.
- `static_map` controls DNS re-resolution of static entries: `network`
  (ip4 default, ip6, ip), `cadence` 30s, `lookup_timeout` 250ms.
- `calculated_remotes` (experimental) guesses a peer's underlay address by
  masking its nebula IP into a known underlay CIDR while the lighthouse
  query is in flight.

## How hole punching works

1. Host A queries lighthouses for B's addresses and starts sending handshake
   packets, which opens a mapping in A's own NAT.
2. The lighthouse notifies B that A is handshaking. B sends an empty packet
   to A's addresses, punching B's NAT so A's packets count as return
   traffic.
3. First handshake through wins. Both NATs hold live mappings and traffic
   flows directly.

Connections always start inside-out, so hosts behind ordinary NATs need no
inbound rules. Nebula has no TCP fallback.

## NAT types and outcomes

- **Public address**: fixed known address or a port forward plus
  `advertise_addrs` plus a pinned `listen.port`. No punching needed.
- **Easy NAT** (endpoint-independent mapping, RFC 4787): one external
  mapping reused for all destinations. Hole punching works. Typical home
  and small office routers.
- **Hard NAT** (symmetric NAT, most CGNAT, many enterprise firewalls): new
  external port per destination, so the address the lighthouse saw is dead
  for other peers.

| Side A vs side B | Public | Easy NAT | Hard NAT |
| --- | --- | --- | --- |
| Public | Direct | Direct | Direct |
| Easy NAT | Direct | Direct via punching | Usually direct, direction-sensitive |
| Hard NAT | Direct | Usually direct, direction-sensitive | Relayed only |

- Mixed easy+hard: the hard side can dial out to the easy side's valid
  observed address, so the handshake works only if it starts from the hard
  side and the easy side's inbound filtering accepts it. `punchy.respond`
  retries the handshake in the reverse direction to rescue this pairing.
- Hard+hard: impossible to punch, use relays.

## Look-alike problems

- Short UDP timeouts kill mappings between packets: enable `punchy.punch`
  for keepalives, and consider lowering `lighthouse.interval` below the
  path's UDP timeout so reported addresses stay fresh.
- Different VLANs or egress paths may have different NAT policy. A tunnel
  working on wifi but not wired points at network policy.
- Enterprise firewalls often have an endpoint-independent or persistent NAT
  mode added for STUN. Enabling it fixes hole punching the same way.

## Firewall requirements on the underlay

- Outbound UDP with return traffic to lighthouse, relay, and peer ports.
  Nebula looks like unknown UDP to deep inspection, like QUIC.
- Inbound only on lighthouses and relays, at a stable public UDP port.
- Consistent NAT mappings (endpoint-independent) for punching to succeed.

## punchy settings

- `punchy.punch` (default false): periodic empty packets to known remotes to
  hold NAT mappings open.
- `punchy.delay` (default 1s): wait after lighthouse notification before
  punching, helps NAT race conditions.
- `punchy.respond` (default false): initiate a reverse handshake when the
  lighthouse reports a peer is trying to reach this host.
- `punchy.respond_delay` (default 5s): wait before the reverse handshake.

## Relays (v1.6+)

- Fallback when no direct path exists. The relay forwards encrypted packets
  between two peers. It sees routing metadata, not contents. Extra latency
  and relay-bandwidth limits apply.
- `relay.am_relay: true` on the relay host, `relay.relays: [<nebula IPs>]`
  on hosts that allow relaying to them, `relay.use_relays: true` (default)
  to let a host use relays at all.
- Relays need public addresses like lighthouses. Any host can be one, it
  does not need lighthouse duty.
- No relay chaining: `am_relay: true` hosts cannot specify their own relays.
- Detection: handshake log lines show `from="ip:port (relayed)"` and
  `Send handshake via relay` names the relay's nebula IP.

## preferred_ranges ordering

High to low: preferred_ranges IPv6, preferred_ranges public IPv4,
preferred_ranges private IPv4, then plain IPv6, public IPv4, private IPv4,
then relay paths. Use it to prefer LAN paths when peers share a network.
