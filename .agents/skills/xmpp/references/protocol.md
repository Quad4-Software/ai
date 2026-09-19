# XMPP protocol reference

## RFCs

- **RFC 6120**: XMPP Core: XML streams, TLS, SASL, resource binding.
- **RFC 6121**: IM & Presence: roster, subscriptions, presence stanzas.
- **RFC 7622**: XMPP Address Format (JIDs).
- **RFC 7590**: Use of TLS in XMPP (TLS recommendations for c2s/s2s).
  Server Dialback is **XEP-0220**, a weak DNS-based identity check still
  used as fallback when cert validation fails. RFC 7590 explicitly
  permits accepting dialback-verified s2s.

## JIDs

`localpart@domainpart/resourcepart`

- **Bare JID** (`local@domain`) addresses the account. Messages to it
  fan out to all online resources.
- **Full JID** (`local@domain/resource`) addresses one device or
  connection.
- **Domainpart-only** JIDs address servers and components:
  `conference.example.org` for MUC, `upload.example.org` for HTTP
  upload, `proxy.example.org` for SOCKS5.

## Stanza types

- **`message`**: `chat` (1:1), `groupchat` (MUC), `headline`
  (transient, no offline store), `normal` (standalone), `error`.
- **`presence`**: `subscribe`/`subscribed`/`unsubscribe`/`unsubscribed`
  (roster negotiation), `probe`, `unavailable`, `error`. Bare presence
  broadcasts availability and capabilities (XEP-0115).
- **`iq`**: `get`/`set` (request) and `result`/`error` (response).
  Request-response pairs keyed by `id`.

## Stream negotiation

```
TCP connect -> <stream:stream> -> <stream:features>
  -> STARTTLS upgrade -> new stream
  -> SASL (SCRAM-SHA-256 modern, PLAIN legacy, channel binding optional)
  -> stream restart -> resource binding
  -> XEP-0198 <enable/> for resumption
```

XEP-0388 SASL2 + XEP-0386 Bind2 collapse the post-auth round trips into
one. XEP-0198 Stream Management adds `<enable/>` for resumable sessions
and `<a>`/`<r>` ack counting.

## DNS SRV records

- `_xmpp-client._tcp` -> 5222 (STARTTLS c2s), `_xmpp-server._tcp` ->
  5269 (s2s). RFC 6120.
- `_xmpps-client._tcp` -> 5223, `_xmpps-server._tcp` -> 5270 (direct
  TLS). XEP-0368. Both record families share priority/weight, `xmpps-`
  requires direct TLS, `xmpp-` forbids it.
- STUN/TURN discovery: `_stun._udp`/`_stun._tcp`, `_stuns._tcp`,
  `_turn._udp`/`_turn._tcp`, `_turns._tcp` (RFC 5389/8489/5766
  conventions). The XMPP-side fallback is XEP-0215 extdisco: a disco
  `urn:xmpp:extdisco:2` query returns STUN/TURN hosts plus temporary
  credentials (Prosody `mod_external_services`/`mod_turn_external`,
  MongooseIM `mod_extdisco`).

## XEP map

| Need | XEP | Notes |
|---|---|---|
| Group chat | XEP-0045 MUC | Presence-bound. ejabberd MUC/Sub and Tigase MEMO are proprietary, not XEPs. "New MUC" is only a ProtoXEP. |
| File upload | XEP-0363 | HTTP File Upload. |
| History | XEP-0313 MAM | Server-side archive, plus MUC-MAM variant. |
| Multi-device | XEP-0280 Carbons, XEP-0198 SM, XEP-0352 CSI | Carbons copies to online resources, SM resumes, CSI throttles when backgrounded. |
| E2EE | XEP-0384 OMEMO | Three incompatible wire versions (see below). |
| PubSub | XEP-0060 + XEP-0163 PEP | Underpins avatars, OMEMO device lists, bookmarks. |
| Avatars | XEP-0084 (PEP) vs XEP-0153 (vCard) | XEP-0398 covers conversion. |
| Bookmarks | XEP-0048 legacy vs XEP-0402 PEP-native | XEP-0411 covers conversion. |
| Calls | XEP-0166 Jingle, 0167 RTP, 0176 ICE-UDP, 0320 DTLS-SRTP, 0353 Jingle Message Initiation, 0234 Jingle FT | The Kaidan/Conversations call stack. |
| Compliance | XEP-0479 | "Compliance Suites 2023" (v0.1.0, May 2023) is the latest published. No CS2024/2025 exists. |
| Ops | XEP-0157 contact addresses, 0077 IBR, 0050 ad-hoc, 0357 push, 0215 extdisco, 0030 disco, 0115 caps, 0199 ping, 0156 alt-connection | |
| UX | XEP-0308 correction, 0424 retraction, 0425 moderation, 0444 reactions, 0461 replies, 0359 stanza-ids, 0421 occupant-id, 0333 markers, 0490 MDS, 0447 stateless file sharing | |

## OMEMO wire versions

XEP-0384 current spec is v0.9.1 (April 2026). Three incompatible
namespaces:

- `eu.siacs.conversations.axolotl`: spec 0.2/0.3, "oldmemo". Widest
  deployment. Conversations and its forks.
- `urn:xmpp:omemo:1`: spec 0.4-0.7.0. Sparse adoption.
- `urn:xmpp:omemo:2`: spec 0.8+, "Twomemo". Current target. Kaidan,
  Converse.js 14 (libomemo.js 2.0).

Devices publish their bundle and device list over PEP (XEP-0163). Trust
is per-device and typically TOFU or fingerprint/QR verified. A message
is encrypted per-recipient-device, so an untrusted device gets nothing.

## Push

- **Android:** XEP-0357 + an FCM app server (Play builds) or a
  persistent connection (F-Droid builds). UnifiedPush is the
  distributor-neutral path Conversations supports.
- **iOS:** APNS only, ~30-second wake. Requires server-side XEP-0357
  plus a vendor push component. Monal uses Apple's
  notification-filtering entitlement and its own app server. Under
  OMEMO the push carries no plaintext. The client wakes, fetches over
  MAM, then notifies.
