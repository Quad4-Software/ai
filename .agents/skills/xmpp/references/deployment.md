# XMPP deployment reference

## Server picks

- **Prosody 13.0.6** (May 2026). Minimal Lua core. You enable each
  module. Community modules at modules.prosody.im cover MAM, upload,
  push, anti-spam. Old stable 0.12.6 still gets security fixes. Pick it
  for control and a small footprint.
- **ejabberd 26.07** (July 2026). Erlang, batteries-included: MUC, MAM,
  push, STUN/TURN, admin API. OTP 27 soft minimum. `mod_invites` added
  invite pages. ConverseJS 14 supported. Pick it for features out of
  the box and horizontal scale.
- **MongooseIM 6.8.1** (Aug 2026). Erlang Solutions' enterprise sibling.
  GraphQL admin API, OTP 27+. Pick it when scale and a formal admin API
  matter.
- **Openfire 5.1.2** (Aug 2026). Java, Apache-2.0, easiest admin GUI,
  plugin ecosystem. Pick it when the GUI matters more than the protocol
  edge.
- **Snikket stable.20260611** (June 2026). Opinionated Prosody-based
  Docker distro, invite-only by design, ships matching clients and a
  "Borogove" SDK/web app in development. Pick it when you want the
  module choices already made.

## Ports

| Port | Use |
|---|---|
| 5222 | c2s STARTTLS |
| 5223 | c2s direct TLS (XEP-0368) |
| 5269 | s2s STARTTLS |
| 5270 | s2s direct TLS (XEP-0368) |
| 5280/5281 | HTTP alt: BOSH/WebSocket/upload/admin |
| 3478 | STUN/TURN |
| 5349 | TURN over TLS |
| 49152-65535 | coturn relay range (typical) |

## DNS

- `_xmpp-client._tcp` -> 5222, `_xmpp-server._tcp` -> 5269 (RFC 6120).
- `_xmpps-client._tcp` -> 5223, `_xmpps-server._tcp` -> 5270
  (XEP-0368). Both families share priority/weight. `xmpps-` requires
  direct TLS, `xmpp-` forbids it.
- Missing SRV records make s2s fall back to the A record on 5269, which
  fails if the XMPP host is not the A record.
- MUCs on `conference.domain` are reached via the parent domain's s2s,
  but the subdomain still needs DNS and cert coverage if it serves HTTP
  or is independently reachable.
- STUN/TURN discovery via `_stun.*`/`_turn.*` SRV or XEP-0215 extdisco.

## TLS

- Effectively mandatory for federation. Let's Encrypt is fine.
- Cover the domain plus every component subdomain you expose:
  `conference.`, `upload.`, `proxy.`, `pubsub.`. A wildcard cert is
  simplest.
- RFC 7590 permits accepting dialback-verified s2s when cert validation
  fails. XEP-0220 is the weak DNS-based fallback, not a substitute for
  a real cert.

## Registration and anti-spam

- Open XEP-0077 in-band registration is a spam magnet. Prefer
  invite-only (Snikket default, Prosody `mod_invites`, ejabberd
  `mod_invites` new in 26.07), CAPTCHA, and rate limits.
- **xmppbl.org** publishes pubsub-based blocklists
  (`muc_bans_sha256`, `spam_source_domains`). Prosody `mod_anti_spam`
  consumes them. Other servers not yet supported.
- Prosody `mod_firewall` covers custom rules. The JabberSPAM resources
  document DNSBL-gated registrations.
- XEP-0377 lets users report spam to their server.

## Monitoring and testing

- **xmpp.net IM Observatory is gone** (shut down Oct 2022). Successors:
  **inspect.xmpp.net** (testxmpp, early preview) and
  **connect.xmpp.net** (quick cert/connectivity check).
- **compliance.conversations.im** is alive and runs XEP compliance
  checks (ejabberd-based).

## Backup

- **Prosody:** internal storage is flat files under `/var/lib/prosody`
  (SQL optional via `mod_storage_sql`). Back up the directory plus
  config and certs.
- **ejabberd:** default Mnesia files under `/var/lib/ejabberd`. SQL
  backends supported. Dump the DB plus config and certs.
- **Snikket:** a single volume holds everything. The distro documents
  `docker run --rm` backup/restore flows.

## Common deployment pitfalls

- A valid cert on the wrong name still fails s2s. The cert must cover
  the XMPP domain, not just the host's.
- Upload/MUC on subdomains need their own DNS and cert coverage if
  clients reach them directly.
- XEP-0160 offline store is small. MAM is the real archive. Carbons only
  sync online resources. SM resume covers short reconnects. Clients
  rely on MAM catch-up for real gaps.
- OMEMO messages to a new device are unreadable until the device
  publishes its bundle over PEP and gets trusted.
