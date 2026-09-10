---
name: lxmf
description: >
  This skill describes LXMF messaging and the `~/.reticulum` state exposed
  by lxmf. Use when you need message format details, local Reticulum
  storage, or destination lookups.
compatibility: stdio-mcp
metadata:
  server: lxmf
---

## When to use this skill

- You need to understand LXMF message structure, delivery methods, or stamps.
- You want to inspect `~/.reticulum` storage, identities, or destinations.
- You are building a tool that reads local Reticulum state.

## How to use

1. Build lxmf: `cd mcp/lxmf && go test ./... && go build`.
2. Add the binary to your MCP client config as `lxmf`.
3. Call `store_config`, `store_inventory`, `store_identities`, or `store_destinations` to read state.
4. Combine with the `reticulum` skill for mesh concepts.

## Examples

- "Show sanitized `~/.reticulum/config` without keys."
- "List known destinations and their first/last seen times."
- "Explain `PROPAGATED` delivery and propagation nodes."

# LXMF Concepts

## Core concepts

- LXMF is a simple, flexible messaging format and delivery protocol built on Reticulum.
- It provides zero-conf message routing, end-to-end encryption, forward secrecy and delivery confirmations.
- It is efficient enough for LoRa, packet radio and other low-bandwidth links.
- An LXMF message has 111 bytes of overhead: 16 bytes destination hash, 16 bytes source hash, 64 bytes Ed25519 signature, then a msgpacked payload.
- The payload is a msgpacked list: [Timestamp, Content, Title, Fields].
- Content, Title and Fields are optional but must be present in the structure.
- The message-id is a SHA-256 hash of Destination + Source + Payload. It is never included in the message because it can be inferred.
- A transient-id is used when a Propagation Node stores an encrypted message for an offline user.

## Message structure

- Destination (16 bytes) - Reticulum destination hash of the recipient.
- Source (16 bytes) - Reticulum destination hash of the sender.
- Ed25519 Signature (64 bytes) - signs Destination + Source + Payload + message-id.
- Payload (msgpack):
  - Timestamp (float, seconds since UNIX epoch).
  - Content (optional bytes or string body).
  - Title (optional bytes or string subject).
  - Fields (optional dictionary of arbitrary structure).
- The signature gives integrity and non-repudiation. Source and destination are authenticated by Reticulum.

## Message signing

- Every LXMF message is signed by the source identity using Ed25519.
- The signature covers the destination, source, payload and inferred message-id.
- A receiver can validate the signature without trusting the transport, because it only needs the source public key.
- If the source is unknown, the message is marked as unverified with reason `SOURCE_UNKNOWN` or `SIGNATURE_INVALID`.
- Do not strip or forge signatures. Never expose private keys to validate or sign on a user's behalf.

## Delivery methods

- `OPPORTUNISTIC` - the message is sent as a single Reticulum packet and opportunistically routed. Good for small, one-shot messages when a link is not yet established.
- `DIRECT` - the message is delivered over a Reticulum Link, providing reliability, forward secrecy and delivery confirmation. This is the default for most messages.
- `PROPAGATED` - the message is handed to a Propagation Node for store-and-forward delivery. Used when the destination is offline or for bulletin/discussion boards.
- `PAPER` - the encrypted message is encoded as a QR code or as a text `lxm://...` URI. Allows completely analog, physical or human-carried transport.

## Stamps and tickets

- A `stamp` is a proof-of-work token attached to an LXMF message.
- `LXStamper` generates it by creating a workblock from the message-id through repeated HKDF rounds, then searching for a random 32-byte stamp that, when hashed with the workblock, produces a value below a target threshold.
- `stamp_cost` controls the difficulty. A higher cost requires more CPU work.
- Stamps limit abuse on public-facing or resource-constrained nodes, such as propagation nodes or paper-message gateways.
- Stamp generation is CPU-bound; it uses multiprocessing on supported platforms.
- A `ticket` is a stamp-like token that can be included with a message to pre-pay or defer stamp validation. Default ticket expiry is 21 days with a 5-day grace period. Tickets automatically renew when less than 14 days remain.
- Set `include_ticket=True` on a message to attach a ticket.

## Propagation nodes

- Propagation Nodes store and forward messages to offline or unreachable endpoints.
- They peer with each other and synchronise messages, forming a distributed encrypted message store.
- Messages can be sent `PROPAGATED`, handed to a node and later retrieved by the destination from any reachable node.
- Propagation Nodes are also the basis for distributed bulletin boards, news groups and discussion systems.
- Run `lxmd -p` to start a propagation node.
- Propagation nodes can require stamps on inbound messages to rate-limit or charge for storage and forwarding.

## Paper messages and sneakernet

- Encrypted LXMF messages can be encoded as QR codes or as `lxm://<payload>` URIs.
- This allows transport by any physical, analog or human-carried medium: print the QR, write the URI, carry it on USB or any other removable media.
- This pattern is often called sneakernet: messages are moved by people on foot, by printed QR, by USB or any other physical carrier.
- To read a paper message, the recipient needs the destination identity (or source identity, depending on encoding) imported into an LXMF client such as Sideband.
- Paper messages bypass radio and network links entirely. This is useful for extreme off-grid, censorship-resistant or long-delay scenarios.
- `PAPER_MDU` is sized for QR-code storage; the `lxm://` URI is a single string that contains the full encrypted message.

## LXM Router and API

- `LXMF.LXMRouter()` is the main API entry point.
- It packs and signs messages, encrypts them, handles delivery method selection, retries, receipts and failure notifications.
- Basic send:
  ```python
  import LXMF
  lxm_router = LXMF.LXMRouter()
  message = LXMF.LXMessage(destination, source, "This is a short, simple message.")
  lxm_router.handle_outbound(message)
  ```
- `LXMessage` states: `GENERATING`, `OUTBOUND`, `SENDING`, `SENT`, `DELIVERED`, `REJECTED`, `CANCELLED`, `FAILED`.
- `LXMessage` representations: `UNKNOWN`, `PACKET`, `RESOURCE`.
- `LXMessage` transport encryption descriptions: `AES-128`, `Curve25519`, `Unencrypted`.
- Set callbacks for delivery: `register_delivery_callback`, `register_failed_callback`.
- Set title, content and fields with `set_title_from_string`, `set_content_from_string` and `set_fields`.

## LXMF fields

This page is under construction and most fields are experimental. All fields are packed by LXMF before sending. Internally LXMF uses MessagePack.

- `FIELD_EMBEDDED_LXMS` = `0x01` - not yet fully implemented.
- `FIELD_TELEMETRY` = `0x02` - node telemetry, all enabled sensors. See Sideband source for format.
- `FIELD_TELEMETRY_STREAM` = `0x03` - aggregated downstream telemetry for bulk transfer.
- `FIELD_ICON_APPEARANCE` = `0x04` - icon for the situation map. Format: `[string ICON, byte[3] FG_COLOR, byte[3] BG_COLOR]`. Icon is a Material symbol name. Colors are RGB.
- `FIELD_FILE_ATTACHMENTS` = `0x05` - list of `[file_name, file_bytes]`.
- `FIELD_IMAGE` = `0x06` - image container. Format: `["webp", image_bytes]`.
- `FIELD_AUDIO` = `0x07` - audio container for non-realtime purposes. Format: `[audio_mode, audio_data]`. See `LXMF.py` for audio modes such as `AM_CODEC2_2400`.
- `FIELD_THREAD` = `0x08` - not yet fully implemented.
- `FIELD_COMMANDS` = `0x09` - direct commands. Built-ins include Ping, Echo and Signal. See Sideband source for format.
- `FIELD_RESULTS` = `0x0A` - command results.
- `FIELD_GROUP` = `0x0B` - not yet fully implemented.

Example appearance field:
```python
lxm_fields = { LXMF.FIELD_ICON_APPEARANCE: ["hiking", b"\xff\xff\x00", b"\x00\x00\xff"] }
lxm = LXMF.LXMessage(dest, source, message_content, desired_method=LXMF.LXMessage.DIRECT, fields=lxm_fields)
```

Example file attachment:
```python
with open("some_file.pdf", "rb") as att_file:
    file_attachment = ["some_file.pdf", att_file.read()]
    lxm_fields = { LXMF.FIELD_FILE_ATTACHMENTS: [file_attachment] }
    lxm = LXMF.LXMessage(dest, source, message_content, desired_method=LXMF.LXMessage.DIRECT, fields=lxm_fields)
```

Example image:
```python
image = ["webp", image_data]
lxm_fields = { LXMF.FIELD_IMAGE: image }
lxm = LXMF.LXMessage(dest, source, message_content, desired_method=LXMF.LXMessage.DIRECT, fields=lxm_fields)
```

Example audio:
```python
audio = [LXMF.AM_CODEC2_2400, audio_data]
lxm_fields = { LXMF.FIELD_AUDIO: audio }
lxm = LXMF.LXMessage(dest, source, message_content, desired_method=LXMF.LXMessage.DIRECT, fields=lxm_fields)
```

## Clients and ecosystem

- User-facing clients: Sideband, MeshChat (original by Liam Cottle), MeshChatX, Nomad Network.
- Community tools: LXMFy, LXMF-Bot, LXMF Messageboard, LXMEvent, RangeMap, LXMF Tools.
- `lxmfy` in this repo provides searchable LXMFy docs and bot scaffolding.

## Installation and daemon

- Install with `pip install lxmf` or `pipx install lxmf`.
- `lxmd` is the included daemon:
  - `-p, --propagation-node` run as a propagation node.
  - `-i, --on-inbound PATH` run an executable on each received message.
  - `--status` show node status.
  - `--peers` show peered nodes.
  - `--sync SYNC` request a sync with a peer.
  - `-b, --break UNPEER` break peering.
  - `-r, --remote REMOTE` remote propagation node hash.
  - `--identity IDENTITY` identity for remote requests.
- `lxmd` can be configured in `~/.lxmd/` and uses the Reticulum instance from `~/.reticulum/`.

## Security and abuse considerations

- Keep private identity files private. Never return key material.
- Stamps raise the cost of flooding but are not perfect spam prevention.
- Propagation nodes should redact or omit source/destination contents when listing metadata.
- Read-only tools should not start or reconfigure `lxmd` or other daemons.
