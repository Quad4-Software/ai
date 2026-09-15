# Security

## Trust model

- Mutual authentication on every tunnel. Both sides present a cert signed by
  a CA in the peer's `pki.ca` bundle during the Noise IXpsk0 handshake.
- A cert proves name, overlay networks, groups, unsafe route subnets, and
  validity. Peers cannot lie about these without breaking the signature.
- Handshake DH: Curve25519 default, NIST P-256 for compliance (requires P256
  CA and certs throughout, or PKCS11-held keys).
- Transport: AES-256-GCM or ChaCha20-Poly1305, set by `cipher`, must match
  network-wide. Message counters are bounded and force rehandshake before
  nonce reuse (v1.11.1 hardening).
- FIPS: build with `boringcrypto` (BoringSSL) or `fips140` tags
  (`make bin-fips140`, `release-fips140` targets). FIPS mode enforces P256
  and AES-GCM, rejects Curve25519.
- Source IP validation: inbound packet source must equal a cert Networks IP
  or fall inside an UnsafeNetworks CIDR, then firewall rules apply.

## Revocation and blocking

- `pki.blocklist`: SHA-256 fingerprints of certs to reject even if valid
  and unexpired. Use when a host key is compromised.
- Blocklists are NOT distributed by lighthouses. Push to every host with
  config management.
- `pki.disconnect_invalid: true` closes tunnels to peers whose certs become
  expired or untrusted. Enable before CA rotation.
- Real revocation window = cert lifetime. Short `-duration` on host certs
  shrinks exposure, at the cost of reissuing.

## Key hygiene

- `ca.key` signs the whole network. Keep offline, encrypt at creation
  (`nebula-cert ca -encrypt`), or keep it in an HSM via PKCS11 (v1.10,
  P256 only).
- Never copy ca.key to lighthouses, relays, or hosts.
- Prefer `nebula-cert keygen` + `sign -in-pub` so host private keys never
  leave their device.
- `sshd` debug console: bind to 127.0.0.1 unless overlay access is needed,
  use SSH CA auth (`trusted_cas`) rather than per-user keys at scale.

## Exposure knobs

- `listen.send_recv_error: never` or `private` stops nebula from answering
  unknown packets, hiding its presence. `accept_recv_error` (v1.10.1)
  controls acting on them.
- `lighthouse.dns.host`: bind the DNS responder to the nebula IP so it is
  unreachable off-overlay.
- Lighthouses and relays see metadata only: who talks to whom, and when.
  Payloads are end-to-end encrypted peer to peer.

## Bulletins and CVEs

- **CVE-2025-62820** (fixed v1.9.7, 2025-10): hosts whose certs had
  unsafe_routes (v1/v2) or multiple Networks entries (v2) could spoof source
  IPs within the cert's Ips subnet. No return traffic possible, so TCP
  spoofing was infeasible. Upgrade past v1.9.6.
- **GHSA-69x3-g4r3-p962** (fixed v1.10.3, 2026-02): P256 signature
  malleability allowed a blocklist bypass since one signature has two valid
  encodings. Both fingerprint forms are now checked and new P256 certs clamp
  to low-s.
- **v1.11.1** (2026-08): IPv6 classifier fix, crafted next-header payloads
  could steer the firewall into matching tcp/udp rules for other protocols.
  Message counter limits enforced to prevent nonce reuse.

## Security posture notes for tool builders

- Never return `ca.key`, host keys, or key passphrases in output. Cert
  fields, fingerprints, and issuer hashes are safe to display.
- `nebula-cert print` output contains no secrets. `pki` config paths do.
- The sshd debug console is an administrative surface. Treat access to it
  like root on the host's networking.
