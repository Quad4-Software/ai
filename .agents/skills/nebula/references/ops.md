# Operations

## SSH debug console (sshd)

Built-in SSH server for inspecting and tweaking a running node. Public key
or SSH CA auth only, no passwords. Port 22 is refused.

```sh
ssh-keygen -t ed25519 -f ssh_host_ed25519_key -N "" < /dev/null
```

```yml
sshd:
  enabled: true
  listen: 127.0.0.1:2222     # or the host's nebula IP for overlay access
  host_key: /path/to/ssh_host_ed25519_key
  authorized_users:
    - user: steeeeve
      keys: ['ssh-ed25519 AAAA...']
```

Bind to `127.0.0.1` for local debugging or to the nebula IP for access over
the overlay, then allow the port in `firewall.inbound` as needed. Since
v1.11, profiling commands write only inside `sshd.sandbox_dir` (default
`$TMP/nebula-debug`), which is not created for you.

Commands via `help`:

- `list-hostmap [-by-index] [-json] [-pretty]`: known connected hosts and
  their underlay addresses.
- `list-pending-hostmap`: hosts mid-handshake.
- `list-lighthouse-addrmap`: lighthouse map entries.
- `print-tunnel <vpn ip>`: full JSON tunnel state, remoteAddrs, relays,
  cert details, message counter.
- `print-cert [vpn ip]`: local or peer certificate.
- `query-lighthouse <vpn ip>`: run twice, first call triggers a background
  query.
- `print-relays`: relay state in JSON.
- `create-tunnel`, `close-tunnel`, `change-remote`: force or fix tunnel
  state. change-remote is temporary, nebula reverts to the preferred remote.
- `log-level`, `log-format`: get or set at runtime.
- `reload`: same as SIGHUP.
- `start-cpu-profile`, `stop-cpu-profile`, `save-heap-profile`,
  `save-mutex-profile`, `mutex-profile-fraction`: profiling into
  sandbox_dir.
- `version`, `device-info`, `logout`.

## Logging

- `logging.level`: panic, fatal, error, warning, info, debug. Runtime
  adjustable via sshd `log-level`.
- `logging.format`: text or json.
- v1.11 moved to slog: levels are uppercase, trace prints as `DEBUG-4`,
  timestamps are always RFC3339Nano, `logging.timestamp_format` is ignored.
  Update any log parsers.
- Useful lines: `Handshake message received` carries certName, certVersion,
  fingerprint, from, vpnAddrs, issuer. `(relayed)` suffix on `from` means
  the tunnel is relayed.

## Stats

- `stats.type: prometheus` serves `listen` + `path` (usually /metrics).
  Graphite pushes to `stats.host` over tcp or udp with `prefix`.
- `stats.interval` is required, recommended 60s.
- `message_metrics: true` adds meta packet counters like
  `messages.tx.handshake`. `lighthouse_metrics: true` adds per-type
  lighthouse packet counters. `message.{tx,rx}.recv_error` is always
  emitted.

## Lighthouse DNS (experimental)

- `lighthouse.serve_dns: true` plus `lighthouse.dns.host`/`port` on a
  lighthouse answers A queries (cert name -> nebula IP) from any reachable
  client, and TXT queries (nebula IP -> cert details) over the overlay only.
- Needs a firewall rule for udp/53 or the chosen port.
- Bind `dns.host` to the nebula IP to restrict it to overlay hosts.
- Limitations: only answers for hosts that have handshaked since the
  lighthouse started, only the cert name resolves, no CNAMEs, no upstream
  forwarding, duplicate names give unstable answers, invalid RFC 1035
  hostnames return empty (punycode works). Since v1.11 the lighthouse
  records its own name too.
- Alternative: point public DNS A records at private nebula IPs.

## Running non-root (Linux)

- Only `CAP_NET_ADMIN` is needed for tun setup, addresses, MTU, routes.
- Add `CAP_NET_BIND_SERVICE` for listeners under 1024 (DNS on 53,
  `listen.port: 443`, prometheus, sshd).
- systemd: `User=nebula`, `Group=nebula`,
  `CapabilityBoundingSet=CAP_NET_ADMIN`, `AmbientCapabilities=CAP_NET_ADMIN`
  in [Service].
- Or setcap on the binary:
  `setcap cap_net_admin,cap_net_bind_service+ep /usr/local/bin/nebula`.
- macOS and Windows still need root/Administrator for tun. With
  `tun.disabled: true` (lighthouse or relay duty) no privilege is needed.

## Logs and service management

- systemd unit example: `examples/service_scripts/nebula.service`,
  Type=notify, `ExecReload=/bin/kill -HUP $MAINPID`, Before=sshd.service.
- SIGHUP reloads config and refreshes certs, firewall, allow lists without
  dropping tunnels.
- `nebula -test -config <path>` validates config and prints the result.
- `nebula -version`, `nebula -help`.
- Log locations: journalctl -u nebula (Linux systemd), Console.app or
  unified log (macOS), Event Viewer or service log (Windows), app log view
  (iOS, Android).

## Gotchas

- `query-lighthouse` returns empty on first call for an unconnected host,
  it triggers a background query. Run it again.
- `change-remote` is temporary, the remote list reverts to preferred
  ordering.
- Encrypted CA passphrases can come from an environment variable since
  v1.10.
