# Config reference

One YAML file per host. Many options are reloadable via SIGHUP without
tearing down tunnels. `nebula -test -config <file>` validates a config.
Annotated example: `examples/config.yml` in slackhq/nebula.

## pki

```yml
pki:
  ca: /etc/nebula/ca.crt      # required, reloadable, PEM bundle of trusted CAs
  cert: /etc/nebula/host.crt  # required, reloadable (IP must not change on reload)
  key: /etc/nebula/host.key   # required, reloadable, PKCS11 URL allowed (v1.10)
  blocklist:                  # reloadable, cert fingerprints to reject
    - <sha256 fingerprint>
  disconnect_invalid: false   # reloadable, drop tunnels to expired/invalid certs
  initiating_version: 0       # v1.10, pick cert version for new handshakes
```

- ca, cert, and key may be file paths or inline PEM via YAML `|` blocks.
- PKCS11 keys (v1.10): `pkcs11:id=...;object=...?module-path=...` with P256
  only, requires build with cgo and pkcs11 tags (`make bin-pkcs11`).
- blocklist is per-host, NOT distributed by lighthouses. Push it with config
  management.

## static_host_map

```yml
static_host_map:
  '192.168.100.1': ['100.64.22.11:4242', 'lighthouse.example.com:4242', '[2001:db8::1]:4242']
```

- Maps a peer Nebula IP to routable underlay addresses. Dialed directly with
  no lighthouse involved. Required for every lighthouse in every host config.
- Accepts literal IPs, DNS names (re-resolved periodically), bracketed IPv6.

## static_map

```yml
static_map:
  network: ip4          # ip4 (default), ip6, ip
  cadence: 30s          # DNS re-query interval for static_host_map names
  lookup_timeout: 250ms # DNS lookup timeout
```

Keep `ip4` unless the host is IPv6-only. Lighthouses learn a node's public
IPv4 from the source address of its report packets.

## lighthouse

```yml
lighthouse:
  am_lighthouse: false      # true ONLY on lighthouse nodes
  serve_dns: false          # lighthouse-only DNS responder, experimental
  dns:
    host: 0.0.0.0           # bind addr, use nebula IP to limit to overlay
    port: 53
  interval: 10              # seconds between host reports to lighthouses
  hosts:                    # EMPTY on lighthouse nodes, nebula IPs not real IPs
    - '192.168.100.1'
  remote_allow_list:        # underlay CIDRs allowed for handshakes
    '0.0.0.0/0': true
    '10.0.0.0/8': false
  local_allow_list:         # filter local addrs reported to lighthouses
    interfaces:
      'docker.*': false
    '10.0.0.0/8': true
  advertise_addrs:          # extra routable addrs to report, port 0 = real port
    - '1.1.1.1:4242'
  calculated_remotes:       # EXPERIMENTAL, guess remotes while querying
    10.0.10.0/24:
      - mask: 192.168.1.0/24
        port: 4242
```

- interval, hosts, allow lists, advertise_addrs, calculated_remotes are
  reloadable.
- serve_dns answers A records (hostname -> nebula IP) and TXT records (cert
  details, overlay only). Needs a firewall rule for udp/53.
- Mixed allow/deny rules require an explicit `0.0.0.0/0` (and `::/0` for v6)
  default entry. Most specific CIDR wins.

## listen

```yml
listen:
  host: 0.0.0.0        # '[::]' for IPv6
  port: 4242           # 0 = random, recommended for roaming nodes
  batch: 64            # packets per recvmmsg syscall
  read_buffer: 10485760
  write_buffer: 10485760
  send_recv_error: always    # always|never|private, v1.6
  accept_recv_error: always  # always|never|private, v1.10.1
  so_mark: 4242              # SO_MARK on Linux, v1.10, for full-tunnel routing
  rebind_on_network_change: true  # macOS, v1.11
```

- Lighthouses and relays need a fixed port. Port 0 breaks port forwards.
- recv_error packets speed reconnection but reveal Nebula presence.
  `private` limits them to private network remotes.

## punchy

```yml
punchy:
  punch: true         # keepalive empty packets hold NAT mappings open
  delay: 1s           # wait before punching after lighthouse notify
  respond: true       # reverse-handshake on peer attempt notification
  respond_delay: 5s
```

See references/discovery.md for when to enable each.

## cipher

`cipher: aes` (AES-256-GCM, recommended, uses AES-NI) or `chachapoly`
(ChaCha20-Poly1305). Must be identical on ALL nodes and lighthouses.

## preferred_ranges

```yml
preferred_ranges: ['172.16.0.0/24']
```

Priority order for underlay addresses, high to low: preferred_ranges IPv6,
preferred_ranges public IPv4, preferred_ranges private IPv4, IPv6, public
IPv4, private IPv4, relay. Replaces deprecated `local_range` (removed in
v1.9.0).

## relay

```yml
relay:
  relays:            # nebula IPs of hosts allowed to relay TO this host
    - '192.168.100.1'
  am_relay: false    # this host forwards for others
  use_relays: true   # allow this host to use relays
```

- Added v1.6. A relay host only forwards for hosts that name it in `relays`.
- Relays need stable public addresses like lighthouses.
- No relay-to-relay: `am_relay: true` hosts cannot list their own `relays`.

## tun

```yml
tun:
  disabled: false    # run without tun (lighthouse-only nodes, no root needed)
  dev: nebula1       # utun[0-9]+ on macOS, tun[0-9]+ on FreeBSD
  drop_local_broadcast: false
  drop_multicast: false
  tx_queue: 500
  mtu: 1300          # reloadable on Linux
  routes:            # per-route MTU overrides, reloadable
    - mtu: 8800
      route: 10.0.0.0/16
  unsafe_routes:     # reloadable, see references/routing.md
    - route: 172.16.1.0/24
      via: 192.168.100.99
  use_system_route_table: false  # Linux, v1.7
  use_system_route_table_buffer_size: 0  # Linux, v1.10
  windows_bypass_wdf: true       # Windows WFP PERMIT filters, v1.11
  network_category: private      # Windows adapter category, v1.11
```

## tunnels (v1.9.6)

```yml
tunnels:
  drop_inactive: false    # drop tunnels idle past inactivity_timeout
  inactivity_timeout: 10m
```

## sshd

```yml
sshd:
  enabled: true
  listen: 127.0.0.1:2222   # port 22 refused
  host_key: /path/to/ssh_host_ed25519_key   # inline PEM allowed
  authorized_users:
    - user: steeeeve
      keys: ['ssh public key string']
  trusted_cas:             # v1.9, SSH cert auth
    - 'ssh ca public key string'
  sandbox_dir: /tmp/nebula-debug  # v1.11, confines profiling output paths
```

No password auth. Public keys or SSH CA certs only.

## logging

```yml
logging:
  level: info       # panic|fatal|error|warning|info|debug
  format: text      # text|json
  disable_timestamp: false
  timestamp_format: '2006-01-02T15:04:05.000Z07:00'  # ignored since v1.11
```

v1.11 switched to slog: levels print upper case, trace is DEBUG-4,
timestamps are always RFC3339Nano.

## firewall

Default deny all both directions. See references/firewall.md.

```yml
firewall:
  outbound_action: drop   # drop|reject (reject sends RST or ICMP admin-prohibited)
  inbound_action: drop    # note: these were applied to the wrong direction before v1.11
  conntrack:
    tcp_timeout: 12m
    udp_timeout: 3m
    default_timeout: 10m
  outbound:
    - port: any
      proto: any
      host: any
  inbound:
    - port: any
      proto: icmp
      host: any
```

## routines

`routines: 1` (Linux only). Thread pairs for tun and UDP queues. Above 1
enables IFF_MULTI_QUEUE and SO_REUSEPORT. Keep below half the CPU cores.

## stats

```yml
stats:
  type: prometheus    # prometheus|graphite, unset = disabled
  listen: 127.0.0.1:8080
  path: /metrics
  namespace: nebula
  subsystem: nebula
  interval: 10s
  message_metrics: false     # per-message counters
  lighthouse_metrics: false  # lighthouse packet counters
```

Graphite options: `prefix` (default nebula), `protocol` (tcp|udp), `host`.

## handshakes

```yml
handshakes:
  try_interval: 100ms  # linear backoff per attempt
  retries: 10          # ~5.5s to resolve at defaults
  trigger_buffer: 64   # buffer for post-lighthouse-query handshakes
```

## local_range

Deprecated, removed in v1.9.0. Use `preferred_ranges`.
