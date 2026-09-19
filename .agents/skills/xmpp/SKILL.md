---
name: xmpp
description: >
  This skill covers the XMPP protocol and its ecosystem. Use it for
  JID/stanza/stream mechanics, DNS SRV records, the XEP landscape (MUC,
  MAM, OMEMO, carbons, HTTP upload), server selection and deployment
  (Prosody, ejabberd, MongooseIM, Openfire, Snikket), client selection,
  TLS and federation, and anti-spam.
---

## When to use this skill

- You are deploying an XMPP server or choosing between Prosody,
  ejabberd, MongooseIM, Openfire, or Snikket.
- You need JID, stanza, stream-negotiation, or SRV-record mechanics.
- You are picking which XEPs matter (MUC, MAM, OMEMO, carbons, upload,
  push) and what version of them is current.
- You are debugging federation, TLS, MUC, or message-sync issues.
- You are picking clients for desktop/mobile or setting up push.

## How to use

1. Read this file for the protocol model, the current server/client
   landscape, and the deployment checklist.
2. Load [references/protocol.md](references/protocol.md) for JID
   anatomy, stanza types, stream negotiation, SRV records, and the XEP
   map.
3. Load [references/deployment.md](references/deployment.md) for
   server-specific install notes, ports, TLS, anti-spam, and backup.
4. Fall back to https://xmpp.org/extensions/ for XEP text and
   https://prosody.im/doc, https://docs.ejabberd.im for server docs.

## Examples

- "Which XEPs do I need for reliable multi-device chat?"
- "Set up Prosody with MUC, MAM, HTTP upload, and invite-only registration."
- "Why does s2s to my domain fail even though the cert is valid?"
- "Explain the three incompatible OMEMO versions."
- "Pick an iOS client that does OMEMO and push."

# XMPP

XMPP is a federated messaging protocol defined by three RFCs and extended
by XEPs (XMPP Extension Protocols). It is decentralized like email: every
domain runs its own server, users address each other as
`local@domain/resource`, and servers federate over server-to-server
links.

## The protocol in one paragraph

A client opens a TCP stream to its server (usually 5222), negotiates TLS
via STARTTLS, authenticates over SASL (SCRAM-SHA-256 modern), binds a
resource, and exchanges XML stanzas. Three stanza types carry everything:
`message` (chat, groupchat, headline, normal, error), `presence`
(subscription and availability), and `iq` (get/set/result/error
request-response pairs). Federation works the same way between servers on
port 5269. Modern additions layer on top: XEP-0198 Stream Management
resumes a broken connection, XEP-0280 Message Carbons copies a message to
every online device, and XEP-0313 MAM keeps a server-side archive so
offline devices catch up.

Full mechanics: [references/protocol.md](references/protocol.md).

## Servers (all actively maintained, Sept 2026)

| Server | Version | Language | Character |
|---|---|---|---|
| Prosody | 13.0.6 | Lua | Minimal core, module-driven. Community modules at modules.prosody.im add MAM/upload/push. Security releases are frequent. |
| ejabberd | 26.07 | Erlang | Batteries-included: MUC, MAM, push, STUN/TURN integration, admin API. OTP 27 soft minimum. |
| MongooseIM | 6.8.1 | Erlang | Enterprise/scale sibling of ejabberd. GraphQL admin API. OTP 27+. |
| Openfire | 5.1.2 | Java | Easiest admin GUI, plugin ecosystem. Apache-2.0. |
| Snikket | stable.20260611 | Prosody-based | Opinionated Docker distro. Invite-only by design. Ships with matching clients. |

Pick **Prosody** for a small-to-mid deployment where you want to choose
each module. Pick **ejabberd** or **MongooseIM** when you want everything
in the box and plan to scale. Pick **Openfire** when the admin GUI
matters more than the protocol edge. Pick **Snikket** when you want a
batteries-included, invite-only distro that already made the module
choices for you.

## Clients (Sept 2026)

| Client | Platform | Latest | Notes |
|---|---|---|---|
| Conversations | Android | 2.20.2 | Reference Android client. OMEMO (legacy axolotl ns), DTLS-SRTP calls, MAM, UnifiedPush. Paid on Play, free on F-Droid. Maintained by Daniel Gultsch on Codeberg. |
| Quicksy | Android/iOS | - | Conversations flavor with phone-number discovery. iOS variant built on Monal since 2024. |
| Cheogram | Android | - | JMP.chat fork, actively developed. |
| Dino | Linux/desktop | 0.5.1 | GTK/Vala. Jingle calls, OMEMO, XEP-0447 file transfer. DinoX fork adds features. |
| Gajim | Desktop | 2.6.0 | Python/GTK. Built-in OMEMO via omemo-dr, OpenPGP XEP-0374. |
| Kaidan | Desktop/mobile | 0.16.0 | Qt/KDE, cross-platform. Experimental Jingle calls since 0.15, OMEMO:2 via QXmpp. |
| Converse.js | Web | 14.0.0 | Embeddable web client. OMEMO:2 via libomemo.js 2.0. |
| Monal | iOS/macOS | 6.4.21 | Most complete iOS client: OMEMO, calls, MAM, APNS push with notification-filtering entitlement. |
| Siskin IM | iOS | 7.4.1 | Tigase. OMEMO, calls, push. |
| Movim | Web | 0.33 | PubSub-as-social-network. First XEP-0503 Spaces implementation. |

## The XEPs that matter

- **Reliability/multi-device:** XEP-0198 Stream Management (acks +
  resume), XEP-0280 Message Carbons, XEP-0313 MAM, XEP-0352 Client State
  Indication, XEP-0359 stanza IDs.
- **Group chat:** XEP-0045 MUC (presence-bound). Server-proprietary
  extensions (ejabberd MUC/Sub, Tigase MEMO-style push) are not XEPs and
  do not federate.
- **Encryption:** XEP-0384 OMEMO. Three incompatible wire versions exist
  (see below). XEP-0374 OpenPGP and XEP-0364 OTR are legacy alternatives.
- **Files:** XEP-0363 HTTP File Upload, XEP-0447 stateless file sharing.
- **Calls:** XEP-0166 Jingle + 0167 RTP + 0176 ICE-UDP + 0320 DTLS-SRTP
  + 0353 Jingle Message Initiation.
- **UX:** XEP-0308 message correction, 0424 retraction, 0425 moderation,
  0444 reactions, 0461 replies, 0333 chat markers, 0490 displayed
  synchronization.
- **Ops:** XEP-0157 contact addresses, 0077 in-band registration, 0050
  ad-hoc commands, 0357 push notifications, 0215 external services
  discovery (STUN/TURN), 0030 service discovery, 0199 ping.
- **Compliance:** XEP-0479 "Compliance Suites 2023" is the latest
  published suite. No CS2024/2025 XEP exists.

## OMEMO versions

Three incompatible wire namespaces coexist:

- `eu.siacs.conversations.axolotl` ("oldmemo", spec 0.2/0.3): widest
  deployment. Conversations and its forks use it.
- `urn:xmpp:omemo:1` (spec 0.4-0.7.0): sparse adoption.
- `urn:xmpp:omemo:2` ("Twomemo", spec 0.8+, current spec v0.9.1 April
  2026): where the spec lives now. Kaidan and Converse.js 14 use it.

OMEMO-1 and OMEMO-2 clients cannot read each other's messages. In
practice, the Conversations-family namespace is what most mobile users
have, and 2 is what new implementations target. Device trust is
per-device: new devices must publish their bundle over PEP and be
trusted by the user.

## Deployment checklist

- **DNS:** `_xmpp-client._tcp` -> 5222, `_xmpp-server._tcp` -> 5269, and
  the `_xmpps-*` direct-TLS variants (XEP-0368) -> 5223/5270. Missing
  SRV records make s2s fall back to the A record on 5269, which fails if
  the XMPP host is not the A record.
- **TLS:** effectively mandatory for federation. A wildcard cert is
  simplest because MUC (`conference.`), upload (`upload.`), and pubsub
  components need names covered.
- **Ports:** 5222 c2s STARTTLS, 5223 c2s direct TLS, 5269 s2s, 5270 s2s
  direct TLS, 5280/5281 HTTP (BOSH/WebSocket/upload/admin), 3478/5349
  STUN/TURN, and a relay range for coturn (typically 49152-65535).
- **Registration:** open XEP-0077 IBR is a spam magnet. Prefer
  invite-only (Snikket default, Prosody `mod_invites`, ejabberd
  `mod_invites` new in 26.07), CAPTCHA, and rate limits.
- **Anti-spam:** xmppbl.org publishes pubsub blocklists
  (`muc_bans_sha256`, `spam_source_domains`). Prosody `mod_anti_spam`
  consumes them. `mod_firewall` covers custom rules. XEP-0377 handles
  user spam reports.
- **Monitoring:** the xmpp.net IM Observatory shut down in Oct 2022.
  Successors: inspect.xmpp.net (testxmpp, early) and connect.xmpp.net
  (cert/connectivity check). compliance.conversations.im still runs
  XEP compliance checks.
- **Backup:** Prosody internal storage is flat files under
  `/var/lib/prosody` (SQL optional via `mod_storage_sql`). Ejabberd
  defaults to Mnesia under `/var/lib/ejabberd` with SQL backends
  supported. Dump the DB, config, and certs.

Full deployment detail: [references/deployment.md](references/deployment.md).

## Common pitfalls

- **OMEMO device trust:** a new device cannot read old messages and will
  not be trusted until the user verifies it. Publishing the device list
  and bundle over PEP is required. A client that skips PEP looks broken
  to contacts.
- **Offline delivery:** XEP-0160 offline store is small. MAM is the
  real archive. Carbons only sync online resources, and SM resume only
  covers short reconnect windows, so clients rely on MAM catch-up after
  longer gaps.
- **MUC semantics:** a MUC room is presence-bound (going offline leaves
  the room). Affiliations (owner/admin/member/outcast) persist. Roles
  (moderator/participant/visitor) do not. Occupant identity in
  semi-anonymous rooms rides on XEP-0421 occupant-id.
- **Feature support is per-client-and-server:** editing (0308),
  retraction (0424), moderation (0425), and reactions (0444) need both
  ends to implement them. Under OMEMO they only work with full-stanza
  SCE (XEP-0420).
