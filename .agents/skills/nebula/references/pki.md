# PKI and nebula-cert

## Trust model

- One or more CAs form the root of trust. `pki.ca` is a PEM bundle, so
  multiple concatenated CA certs are supported. This is how CA rotation and
  cert v1->v2 migration work without downtime.
- A host cert binds a name, overlay networks, groups, unsafe route subnets,
  and a validity window to a public key. Mutual auth validates the peer cert
  against the trusted CA set during the Noise handshake.
- `ca.key` is the most sensitive artifact. Keep it offline, never copy it to
  lighthouses or hosts. Encrypt it at creation with `-encrypt`.
- CA key encryption (v1.7+): AES-256-GCM with Argon2id KDF, defaults are the
  RFC 9106 FIRST RECOMMENDED parameters (1 iteration, 4 lanes, 2 GiB RAM).
  Tune with `-argon-iterations`, `-argon-parallelism`, `-argon-memory`.
  The passphrase can come from an environment variable (v1.10).

## nebula-cert subcommands

- `ca`: create a CA.
  - `-name` required. `-duration` (default 1 year). `-curve` `25519`
    (default) or `P256` (NIST). `-encrypt` to protect ca.key.
    `-version 2` for a v2 CA (default on v1.10+ builds).
  - Restrictions: `-ips` (v1) or `-networks` (v2) limit cert IPs, `-groups`
    limits signable groups, `-subnets` (v1) or `-unsafe-networks` (v2)
    limits routeable subnets. A restricted CA cannot sign certs outside its
    limits.
  - `-out-crt`, `-out-key`, `-out-qr` (QR for mobile import).
- `sign`: create a host cert and keypair.
  - `-name`, `-ip "192.168.100.5/24"` (v1) or
    `-networks "192.168.1.1/24,fdc8::1/64"` (v2, multiple allowed).
  - `-groups "a,b"`, `-subnets` (v1) / `-unsafe-networks` (v2) for
    unsafe_routes sources, `-duration` (default: CA expiry minus 1s).
  - `-in-pub host.pub` signs a public key so private keys never leave the
    device. `-ca-crt`, `-ca-key` point at CA material, `-version` picks the
    output cert version, `-out-crt`, `-out-key`, `-out-qr`.
  - P256 CA implies PKCS11 signing support via `-ca-key` pkcs11 URL.
- `keygen`: `-out-key`, `-out-pub`, `-curve`. Generates a keypair on the
  target device for the `-in-pub` flow.
- `print`: `-path`, `-json` (pipe to `jq .details`), `-out-qr`.
- `verify`: `-ca ca.crt -crt host.crt` checks signature and expiry.
- `-` reads stdin or writes stdout (v1.11).

## Certificate fields

v1 cert fields: Name, Ips (one CIDR), Subnets (unsafe route CIDRs), Groups,
NotBefore, NotAfter, IsCA, Issuer (CA fingerprint), PublicKey, Curve
(CURVE25519 or P256), Fingerprint (SHA-256), Signature.

v2 cert (v1.10+, ASN.1 format): Ips renamed to Networks (multiple IPv4 and
IPv6 CIDRs), Subnets renamed to UnsafeNetworks. `nebula-cert print` shows
`"version": 2`. v2 enables IPv6 overlay addressing.

## Workflows

### Sign without copying private keys

1. On the device: `nebula-cert keygen -out-key alice.key -out-pub alice.pub`
2. Move `alice.pub` to the CA host, sign:
   `nebula-cert sign -in-pub alice.pub -name Alice -ip "192.168.100.25/24" -groups "users,developers"`
3. Copy only `Alice.crt` back.

### Rotate a CA without downtime

1. `nebula-cert ca -name "Org #2"` matching the old CA's CIDR, group, and
   subnet restrictions (`nebula-cert print` the old CA to check).
2. Append the new CA PEM to `pki.ca` on every host, then reload
   (`kill -HUP <pid>`). Look for `Trusted CA certificates refreshed` in the
   log.
3. Re-sign each host cert with the same name, IPs, groups, subnets. Deploy
   and reload.
4. Remove the old CA from `pki.ca` everywhere and reload.
5. Enable `pki.disconnect_invalid` before rotating so stale tunnels close
   fast and problems surface early.

### Migrate to cert v2 / IPv6 overlay (v1.10+)

1. Upgrade ALL hosts to v1.10+ first. Older builds cannot validate v2 certs
   (v1.9.5+ at least ignores them gracefully).
2. Create a v2 CA: `nebula-cert ca -name "CA v2" -encrypt -version 2`.
3. Append it to `pki.ca` on all hosts and reload. Hosts can run a v1 and a
   v2 cert at once.
4. Re-sign hosts with `-networks`, optionally dual-stack. Set
   `pki.initiating_version: 2` once the mesh is all v1.10+.
5. Update every place a nebula IP appears for dual-stack: static_host_map,
   lighthouse.hosts, tun.unsafe_routes via, relay.relays, firewall cidr.
6. Remove the v1 CA and v1 certs when the network is fully migrated.
   Rollback: keep both CAs, restore v1 cert and key, set
   `initiating_version: 1`, restart.

## Gotchas

- Cert IP changes require a restart, not just a reload. All other cert
  fields can change on reload.
- CA defaults to 1 year. Host certs default to CA expiry minus 1 second.
  Alert on expiry months ahead.
- Duplicate cert names are legal. Only the IP must be unique. This makes
  lighthouse DNS answers for duplicate names unstable.
- v1 certs can technically carry multiple Ips, but nebula-cert never issues
  that. Multi-IP v1 certs were part of CVE-2025-62820 exposure.
