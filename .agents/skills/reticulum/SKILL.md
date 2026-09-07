---
name: reticulum
description: >
  Use when working with the Reticulum-related servers (rns-mcp, lxmf-mcp,
  lxmfy-mcp, meshchatx-mcp): Reticulum protocol and API, cryptography, links,
  channels, requests, responses, transports, LXMF, LXST, LXMFy, NomadNet,
  MeshChatX, source mirrors, pip-rns, ~/.reticulum state, interface config,
  diagnostics and tool-building conventions.
---

# Reticulum Mesh Concepts

## Core concepts

- **Reticulum (RNS)** is the encrypted, medium-agnostic mesh networking stack.
- A destination is a 16-byte hash (128 bits) derived from a SHA-256 hash of the
  destination's public key and identifying characteristics. Addresses are
  portable identities, not locations.
- Local state lives in `~/.reticulum`. That directory holds the config file,
  storage directory, identities and known destinations.
- **LXMF** is a lightweight messaging format and delivery protocol over
  Reticulum. It uses propagation nodes for offline / store-and-forward delivery.
- **LXMFy** is a Python bot framework on LXMF. lxmfy-mcp has searchable docs and
  bot scaffolding. See the lxmfy skill for framework details.
- **LXST** is a real-time streaming (voice / signal) protocol over Reticulum.
- **NomadNet** is an LXMF-based mesh comms app whose pages use Micron markup.
- **MeshChatX** is an LXMF mesh chat client, and meshchatx-mcp exposes its docs.

## Zen of Reticulum for tool builders

The [Zen of Reticulum](https://reticulum.network/manual/zen.html) is not only a
user philosophy, it is a set of engineering constraints. Reticulum-related MCP
tools should be designed in its spirit:

- **No center.** Tools must not act as privileged servers that users "connect
  to". They are peers that read and report; they do not restart rnsd or modify
  host state.
- **Assume a hostile network.** Every tool that surfaces local state must treat
  paths, config and storage as potentially sensitive. Redact keys, passphrases
  and credentials. Never return private key material.
- **Encryption is not a feature.** It is the foundation. Do not implement
  plaintext workarounds or "insecure" convenience modes.
- **Scarcity matters.** Keep MCP responses small and bounded. Cap lists,
  truncate logs, and report when output is capped. Respect low-bandwidth
  contexts where these tools may be consumed.
- **Store and forward.** Where data cannot be fetched immediately (for example a
  remote mesh peer), design for asynchronous, deferred, or retryable discovery
  rather than synchronous polling.
- **Identity is not a location.** Refer to RNS destinations / identity hashes,
  not to IP, DNS or hostnames, when discussing mesh resources.
- **Tool ethics.** Do not build Reticulum tools that expose others' private
  state, enable surveillance, or strip redaction from config. The reference
  implementation's license reflects this intent.

## Installing and running RNS

- The Python reference implementation is installed with `pip install rns` or a
  downloaded wheel.
- Main daemon is `rnsd`, which reads `~/.reticulum/config`.
- Generate a commented example config with `rnsd --exampleconfig`.
- Common utilities: `rnsd`, `rnstatus`, `rnpath`, `rnprobe`, `rncp`, `rnid`.

## Destinations and addressing

- Destinations are 16-byte hex hashes, e.g. `<13425ec15b621c1d928589718000d814>`.
- Any node can generate as many destinations as it needs without coordination.
- Uniqueness comes from the public key in the hash. Two nodes can both use the
  name `messenger.user.inbox` and still get unique destination hashes.
- Single and packet communication uses X25519 / Ed25519. Links add forward
  secrecy via ECDH on Curve25519.

## Interfaces

- `~/.reticulum/config` holds `[[interfaces]]` sections. Common types:
  `AutoInterface`, `BackboneInterface`, `TCPServerInterface`,
  `TCPClientInterface`, `UDPInterface`, `RNode`, `Serial`, `I2P`, `KISS` packet
  radio TNCs and custom Python modules.
- `AutoInterface` uses UDP ports `29716` and `42671` for local discovery. If
  discovery scope is configured, defaults include discovery port `48555` and
  data port `49555`.
- The `BackboneInterface` is Linux/Android only and uses a kernel-event I/O
  backend. It is intercompatible with `TCPServerInterface` and
  `TCPClientInterface`.
- Custom interfaces can be loaded from user or community supplied Python
  modules.
- When surfacing config, redact passphrases, keys, credentials and any token
  fields.

## Building networks

- Reticulum is a toolkit for creating networks, not a single network you join.
- Add interfaces; routing and convergence are automatic.
- A **transport node** or **transport instance** forwards traffic blindly for
  others. It only knows its immediate neighbours. Any node can become a
  transport node, but too many degrade performance.
- Good transport candidates are stationary, well-connected and long-running.
- Use `bootstrap_only` interfaces with discovery to grow organically instead of
  hard-coding central entrypoints. After enough peers are discovered, the
  bootstrap interface can be disconnected.
- Directory / bootstrap references include `directory.rns.recipes` and
  `rmap.world`.

## LXMF

- **Lightweight Extensible Message Format** over Reticulum.
- A message has: 16-byte Destination hash, 16-byte Source hash, 64-byte Ed25519
  signature, and a msgpacked Payload.
- Payload contains: Timestamp, Content, Title and Fields (all may be empty).
- The `message-id` is the SHA-256 hash of Destination + Source + Payload. A
  `transient-id` is used when a propagation node stores an encrypted message for
  an offline user.
- Delivery methods: over a Reticulum Link (default, forward secret), as an
  opportunistic single packet, or to a GROUP destination (symmetric key).
- **Propagation Nodes** store and forward messages for offline endpoints and can
  power distributed bulletin / discussion boards. They peer and synchronise
  automatically.
- **LXM Router** is the Python API entry point: `LXMF.LXMRouter()`. It handles
  packing, encryption, delivery receipts, path lookup, routing, retries and
  failure notifications.
- The `lxmd` daemon (ships with `lxmf`) runs an LXMF router and optional
  propagation node.
- Overhead is 111 bytes for the non-payload portion.
- Install with `pip install lxmf` or `pipx install lxmf`.
- User-facing clients: Sideband, MeshChat, Nomad Network.

## LXST

- **Lightweight Extensible Signal Transport** over Reticulum.
- Real-time streaming for telephony, voice, two-way radio, media streaming,
  broadcast and public-address.
- Supports OPUS, raw / lossless streams, and Codec2 for ultra low-bandwidth
  voice (700 bps to 3200 bps). Can switch codecs mid-stream.
- End-to-end encryption, integrity, authenticity and forward secrecy via
  Reticulum.
- Ships with the `rnphone` telephony example.
- Install with `pip install lxst`.
- The repository is a public mirror; development happens elsewhere.
- It is early alpha: APIs are unstable, documentation is sparse.

## NomadNet

- NomadNet 1.4.0 is available from https://pypi.org/project/nomadnet/.
- A prototype for inline image rendering was discussed in a rns.recipes forum thread. The screenshot is live in nomadnet, not a mockup.
- It uses the Kitty Terminal Graphics Protocol with zero new dependencies and pure Python.
- Supports image buffer re-use, sizing in cols/rows or percentages, and left/right/center alignment.
- On unsupported terminals it falls back to an alt-text placeholder.
- For low-bandwidth links the serving side needs scaling, compression, selective retrieval and a user-selectable text-only mode.
- Terminals tested: Kitty, Konsole and Wezterm. Ghostty appears supported. Windows Terminal support is being explored.
- Screenshot: ![nnimgs.png](https://rns.recipes/storage/forum/XClxgw3Xynkh1DZYiNnKMj5pwp90moVGIiIVxg3T.png)
- Source thread: https://rns.recipes/forum/general/so-this-is-possible-but

## pip-rns

- `pip-rns` installs Python packages directly from Reticulum `rngit` remotes.
- Also works with pip, pipx, uv and poetry backends, supports version pinning,
  signed releases, a trust store, offline `.opip` bundles, and aliases.
- Install the tool itself with `pip install pip-rns` or `pipx install pip-rns`.
- Commands include:
  - `pip-rns install <identity/group/repo> [--from-release|--from-source|-s]`
  - `pip-rns trust` to remember publisher identities
  - `pip-rns export` to mirror release artifacts for USB / sneakernet sharing
  - `pipx-rns` and `opip` for pipx / offline bundle workflows
- `rns://id/group/repo` is the remote path format.
- `--verify IDENTITY` pins a publisher. `--require-release` / `--insecure` are
  fail-open / fail-closed flags.

## Latest source and mirrors

- The official public source mirror for Reticulum and related projects is on
  GitHub under `markqvist/`. Some repositories (LXST) note that active
  development happens elsewhere.
- The user has indicated the latest source lives on `rngit` over Reticulum at:
  - `rns://7649a50d84610232d1416b41d2896aff/reticulum/reticulum`
  - `rns://7649a50d84610232d1416b41d2896aff/reticulum/nomadnet`
  - `rns://7649a50d84610232d1416b41d2896aff/reticulum/website`
  - `rns://7649a50d84610232d1416b41d2896aff/reticulum/lxmf`
- These `rns://` paths are only resolvable inside a running Reticulum network.
  Tools should not attempt HTTP access to them and should not fabricate content.

## Reticulum MCP tools

Server and tool conventions are in [references/tools.md](references/tools.md). This section is optional if the relevant MCP server is installed.

## RNS lifecycle and reload

- The rnsd daemon reads interface and destination config from
  `~/.reticulum/config`. Changing those values requires either a full daemon
  restart or a SIGHUP reload on running platforms that support it.
- Never restart rnsd from these read-only tools.
- `lxmf-mcp` only reads state.

## Diagnostics

- `rnstatus` shows local Reticulum status and interface health.
- `rnpath` queries and prints paths to destinations.
- `rnprobe` tests reachability and latency to a destination.
- `rncp` copies files over Reticulum.
- `rnid` displays or generates identity information.
- `lxmd` runs an LXMF router / propagation node.
- Inspect `~/.reticulum/storage` and `~/.reticulum/config` to understand state.
- `lxmf-mcp` tools decode known destinations without needing python beyond the
  bundled decoder usage, because python3 is used solely to decode stored blobs.

## Security, Zen and anti-footguns

### Reticulum Zen for any application

The Zen of Reticulum is not only for the core stack. Apply it to every Reticulum application you build or operate:

- It should be useful as a tool, with no strings attached.
- It should be understandable and not require trust in opaque infrastructure.
- It should work reliably under harsh conditions and low bandwidth.
- It should keep the user in full control of their own keys, paths and data.
- It should default to safe behaviour and fail closed, never to plaintext or guessable defaults.

### Security-relevant fixes from upstream history

Recent RNS releases (especially 1.4.x through 1.5.2) include fixes worth knowing about:

- **Resource decompression bomb** - fixed a `bz2` decompression bomb vulnerability in Resource transfer assembly and Buffer `StreamDataMessage` unpacking. Do not accept arbitrary resources from untrusted sources on unpatched versions.
- **rnsh security** - fixed a critical security issue in `rnsh`. Keep `rnsh` updated and never run `rnsh -n` (no auth) on untrusted or public networks.
- **Interface Access Codes** - added early protocol violation checks for invalid frames, fixed incorrect default IFAC sizes for discovered interfaces and optimized IFAC validation. Always enable IFAC on any interface that can be reached by untrusted peers.
- **Null identity blocking** - added the ability to block unidentified or null-identity peers. Use the `null_ident` options in config when you want to reject anonymous stations.
- **Blackhole / distributed blackhole** - blackhole handling lets you drop announces and traffic from identified spammers; distributed blackhole lists let communities share abuse signals. Use `rnpath -B` and `rnstatus -b` to manage and inspect.
- **Discovery hardening** - invalid discovery stamps are now cached, corrupted interface discovery files are handled, and discovery sanitizer None-checks were added. Treat unknown discovery info as untrusted.
- **Link and request race fixes** - fixed race conditions in link watchdog, request timeout handling, and `BackboneInterface` fast-flap detection. These avoid stalls and potential resource leaks.
- **Ratchet and identity handling** - fixed ratchet cleaning, retained preservation and various identity handling bugs. Keep Reticulum updated so keys and ratchets are handled correctly.
- **Release verification** - `rngit release` and `rnid -V` let you verify signed release manifests and artifacts against Mark Qvist's release identity `<bc7291552be7a58f361522990465165c>`.

### Anti-footguns

- **Private keys are everything** - `rnid -P` prints private keys. Never log, share, or commit output that contains private keys. Back up identity files offline and encrypted.
- **Do not restart or reconfigure `rnsd` from a read-only tool** - the first `RNS.Reticulum` instance on a system owns the hardware interfaces. MCP tools should never `SIGKILL` it or rewrite `~/.reticulum/config`.
- **Transport node placement** - do not enable `enable_transport = Yes` on every node. It degrades the network on mobile, battery or low-bandwidth nodes. Use stationary, well-connected nodes as transport nodes.
- **Plain and group destinations** - `PLAIN` is local-only and unencrypted; `GROUP` currently requires establishing through a `SINGLE` destination. Do not rely on group or plain for multi-hop private traffic.
- **No auth on remote utilities** - `rnsh -n` and `rnx -n` accept commands from any identity. Only use them on fully trusted, closed links, never on public interfaces.
- **IFAC on public carriers** - any interface over the Internet, public WiFi, or shared radio should use IFAC with a strong passphrase or authentication. Without it, anyone can inject packets.
- **Monitor and blackhole** - use `rnstatus -b` to watch blocked IPs and `rnpath -B` to blackhole abusive identities. Combine with `null_ident` blocking for unknown peers.
- **Keep software updated** - 1.5.2 fixed a resource transfer regression and an I2P keepalive bug; earlier releases fixed `rnsh`, decompression bombs, and discovery issues. Use `rngit` or `pip` to stay current.
- **Redact and bound output** - tools should sanitize local state, never return key material, and keep responses short for low-bandwidth links.
- **Do not fake source addresses** - Reticulum uses cryptographic addresses. Do not invent destination hashes, fabricate announces, or replay signed messages.
- **Test on real links** - behaviour on fast TCP differs from LoRa. Test latency, packet loss, and retransmission before assuming a design works.

## Packet reference

Reticulum packets have a fixed 2-byte header, optional IFAC, one or two 16-byte addresses, a 1-byte context field and up to 465 bytes of data.

### Wire layout

```
[HEADER 2 bytes] [IFAC optional] [ADDRESSES 16/32 bytes] [CONTEXT 1 byte] [DATA 0-465 bytes]
```

- Header byte 1 bit layout: `[IFAC Flag][Header Type][Context Flag][Propagation Type][Destination Type][Packet Type]`.
- Header byte 2: hop count.

### Header flags and types

| Field | Values | Meaning |
|---|---|---|
| IFAC | `0` open, `1` authenticated | whether an Interface Access Code is appended |
| Header Type | `0` type 1, `1` type 2 | one or two 16-byte address fields |
| Context Flag | `0` unset, `1` set | used for signalling depending on packet context |
| Propagation | `0` broadcast, `1` transport | local broadcast or transport-routed |
| Destination | `00` single, `01` group, `10` plain, `11` link | destination type |
| Packet | `00` data, `01` announce, `10` link request, `11` proof | packet type |

### Typical packet sizes

- Path request: 51 bytes
- Announce: 167 bytes
- Link request: 83 bytes
- Link proof: 115 bytes
- Link RTT: 99 bytes
- Link keepalive: 20 bytes

### Examples

Two address, transport, single, data, 4 hops, no IFAC:
```
01010000 00000100 [HASH1 16 bytes] [HASH2 16 bytes] [CONTEXT 1 byte] [DATA]
```

One address, broadcast, single, data, 7 hops, no IFAC:
```
00000000 00000111 [HASH1 16 bytes] [CONTEXT 1 byte] [DATA]
```

One address, broadcast, single, data, 7 hops, with IFAC:
```
10000000 00000111 [IFAC N bytes] [HASH1 16 bytes] [CONTEXT 1 byte] [DATA]
```

## Weave

Weave is a switching fabric for Reticulum under development by Mark Qvist. It acts as an abstraction layer over network topology and lets remote nodes anywhere on the fabric be accessed as local interfaces, regardless of physical distance or number of hops.

- Primary use case is efficient, fast-converging backhauling for slower mediums such as LoRa.
- It can run over 802.11, IP or any other supported carrier. A Weave fabric can be built from cheap ESP32s switching over 2.4 GHz 802.11.
- An RNode can connect to the fabric using its 802.11 radio and expose a Weave interface endpoint. `rnsd` can then connect to that RNode remotely and use it as an interface, even if it is multiple hops away.
- Early tests achieved about 150 kbps raw throughput over 6.5 km using ESP32 2.4 GHz radios.
- Weave is more general than "RNode over IP". It is a universal switching fabric that can transport Reticulum traffic and backhaul many RNodes to a remote `rnsd` instance.

## Quick reference

- State and config live in `~/.reticulum`: `config`, `storage` and `identities`.
- Install: `pip install rns`, `pip install lxmf`, `pip install nomadnet`, `pip install lxst`.
- Daemons: `rnsd` (Reticulum core), `lxmd` (LXMF router / propagation node), `nomadnet` (console client).
- Generate an example config with `rnsd --exampleconfig`.
- Identities and hashes: `rnid` to create/view, `rnid -i <name> -H <aspects>` for the destination hash.
- Paths and reachability: `rnpath <hash>`, `rnpath -t` for the path table, `rnprobe <app_name> <hash>` for a probe.
- Files: `rncp <file> <hash>` copies a file over Reticulum.
- Status: `rnstatus` shows local instance and interface health.
- Bootstrap and directory references: `directory.rns.recipes` and `rmap.world`.
- Public source mirrors: `github.com/markqvist/Reticulum` and `rns://7649a50d84610232d1416b41d2896aff/reticulum/reticulum`.
- A destination hash is a 16-byte hex value in angle brackets, e.g. `<13425ec15b621c1d928589718000d814>`.

## Plugin / god-file pattern

- The doc servers embed section-aware doc corpora in `internal/docs` and in
  `internal/community` for `rns-mcp`. Keep each server's `main.go` thin with
  tool registration plus handlers delegating to `internal` packages. Avoid
  growing a single god-file. Split handlers by domain.

## Protocol and programming reference

This section is a condensed reference derived from the Reticulum manual. It is intended to let an LLM understand the protocol, cryptography, packet/link/resource model and the main API classes.

### Design goals

- Reticulum is a message-oriented networking stack for high-latency, low-bandwidth links.
- It can run over any half-duplex medium with more than 5 bits per second and a physical-layer MTU of 500 bytes.
- It does not need IP; it can be tunnelled over TCP or UDP, but it is a complete networking stack by itself.
- Coordination-less addressing, initiator anonymity, encryption by default and permissionless operation are core design goals.
- Reticulum is not one network. It is a toolkit for building thousands of autonomous, interoperable networks.

### Destinations and destination types

- A destination is a 16-byte (128-bit) hash, truncated from a SHA-256 hash of the app name, aspects and public key.
- Displayed as 16 hex bytes in angle brackets, e.g. `<13425ec15b621c1d928589718000d814>`.
- Any node can create as many destinations as it needs without coordination, because uniqueness comes from the public key in the hash.
- Three normal destination types:
  - `SINGLE` - the default. Uses the destination's public key for ECDH-encrypted unicast. Supports multi-hop transport. Use this for almost all private communication.
  - `PLAIN` - unencrypted broadcast. Only works on the local link and is not transported over multiple hops. Use for local discovery or public broadcasts.
  - `GROUP` - symmetric-key encrypted destination. Currently requires establishing through a `SINGLE` destination, then supports private one-to-many traffic.
- One special type:
  - `LINK` - a bidirectional, ephemeral, authenticated channel to a `SINGLE` destination. Offers forward secrecy, receipt proofs and a richer API.
- Destination names are dotted "aspects", e.g. `app.aspect.subaspect`. Names are hashed, so long names do not affect on-wire size. For `SINGLE` destinations, Reticulum automatically appends the public key before hashing.
- The `app_name` should identify the application. Subsequent `aspects` describe the endpoint.

### Identities

- An `RNS.Identity` is a 512-bit elliptic-curve keyset: a 256-bit X25519 encryption key and a 256-bit Ed25519 signing key.
- It can represent a user, a device, a service or a whole network.
- A `SINGLE` destination is always tied to an identity. `PLAIN` and `GROUP` destinations are not.
- Private keys must be kept secret. Identity files can be saved or loaded with `to_file`, `from_file` and `pub_to_file`.
- `RNS.Identity.recall(destination_hash)` returns the identity associated with a known destination, or `None`.

### Announces and public key distribution

- A destination can call `announce()`. The announce packet contains the destination hash, public key, an Ed25519 signature, an application data blob and a random nonce.
- Announces propagate through transport nodes. Each transport records the next hop and retransmits with these rules:
  - Ignore duplicates.
  - Stop after m+1 retransmissions (m defaults to 128).
  - Respect a per-interface bandwidth cap for announces (default 2%).
  - Prioritise closer (fewer hop) announces on slow links.
  - Retry r times if no newer retransmit is heard (r defaults to 1).
- Announces make a destination reachable network-wide and can also carry small `app_data` (e.g. nickname, status).
- Destinations are portable: they can move between interfaces, networks or mediums by sending a new announce.
- A path request can find a destination that has not been announced locally.

### Transport and routing

- Every Reticulum program creates one `RNS.Reticulum` instance. The first instance on a system becomes the master and owns the hardware interfaces.
- Transport nodes forward packets, respond to path requests and pass announces. Instances do not. Enable with `enable_transport = Yes` in config.
- No node knows the full path. Each transport node only knows the best next hop.
- Path requests are flooded until a node with a known path to the destination answers, and the answer propagates back.
- Transport nodes act as a distributed cryptographic keystore: they remember public keys from announces and can satisfy key recalls.
- Transport can be disabled on mobile or battery-powered nodes; they will still reach the network through nearby transport nodes.
- Good transport nodes are stationary, well-connected and always-on.

### Cryptographic primitives

- Ed25519 for signatures.
- X25519 for ECDH key exchange.
- HKDF for key derivation.
- SHA-256 for hashing and addresses; SHA-512 is also used internally.
- Encrypted tokens follow the Fernet spec, but without version and timestamp metadata:
  - Ephemeral key from X25519 ECDH on Curve25519.
  - AES-256 in CBC mode with PKCS7 padding.
  - HMAC-SHA256 for authentication.
  - IVs generated with `os.urandom()` or a stronger CSPRNG.
- By default X25519, Ed25519 and AES-256 are provided by OpenSSL via PyCA/cryptography; SHA by hashlib. HKDF, HMAC, Token and PKCS7 padding are provided by the internal `RNS/Cryptography/` modules.
- A complete pure-Python fallback implementation is included if the accelerated backend is not available.

### Packets

- A packet is the smallest unit of communication. It is created for a destination and carries a payload.
- For `SINGLE` destinations, each packet uses a fresh ephemeral key and ECDH with the destination's public key or ratchet key. This gives per-packet derived encryption and initiator anonymity.
- The destination can return a proof: a SHA-256 hash of the packet, signed with its Ed25519 key. Transport nodes route the proof back.
- `PLAIN` packets are not encrypted. `GROUP` packets use a pre-shared symmetric key.
- Typical overhead: path request 51 bytes, announce 167 bytes, link request 83 bytes, link proof 115 bytes, link RTT 99 bytes, keepalive 20 bytes.
- Wire format: `[HEADER 2 bytes] [optional IFAC] [ADDRESSES 16/32 bytes] [CONTEXT 1 byte] [DATA 0-465 bytes]`.
- Header byte 1: IFAC flag, header type, context flag, propagation type, destination type, packet type. Byte 2 is the hop count.
- Destination type values: `single=00`, `group=01`, `plain=10`, `link=11`. Packet type values: `data=00`, `announce=01`, `link request=10`, `proof=11`.

### Links

- A `RNS.Link` is a verified, encrypted channel to a `SINGLE` destination. Setup takes 3 packets, 297 bytes. Keepalive is about 0.45 bits per second.
- The link initiator generates an X25519 key pair and sends a link request containing `LKi`.
- Forwarding transport nodes store the `link_id` (hash of the link request) and hop count in a link table.
- The destination accepts or rejects, generates its own X25519 pair, derives a symmetric key and sends a `link proof` containing `LKr` and a signature of the `link_id` + `LKr` with its original Ed25519 signing key.
- Transport nodes verify the proof and mark the link active. Packets can then be addressed to the `link_id` rather than the destination hash.
- Links offer forward secrecy, initiator anonymity, reliable message receipt and retransmission (with `Resource`).
- The initiator can authenticate with an `RNS.Identity` inside the encrypted link or stay anonymous.

### Resources, channels and buffers

- `RNS.Resource` transfers arbitrary amounts of data over a `Link`.
- It auto-compresses, splits into packets, sequences, verifies integrity and reassembles.
- It can stream from memory or from files. Good for file transfer or large payloads.
- `RNS.Channel` provides reliable, sequential, bidirectional byte streams over a `Link`.
- `RNS.Buffer` provides a simpler buffer abstraction for sequential data.

### Requests and responses

- A destination can register a request handler with `register_request_handler(path, response_generator)`.
- A peer sends a request with `request(path, data, response_callback)` on a `Link`.
- Requests are received, validated and a response is returned by the `response_generator`.
- Requests can be automatically compressed and have a configurable `max_request_size`.
- This is a lightweight RPC-like mechanism for building applications.

### Interface modes and propagation

- Interfaces can be in `Full`, `Gateway`, `Roaming`, `Bound` or `Listen` mode, controlling how they forward announces and traffic.
- Full mode forwards all relevant traffic. Gateway mode connects two areas. Roaming is for mobile clients. Bound and Listen are special cases.
- `AutoInterface` uses UDP discovery on ports 29716 and 42671 (or 48555 and 49555 with a discovery scope).
- `BackboneInterface` is Linux/Android only, uses a kernel-event I/O backend and is compatible with TCP server/client.
- TCP, UDP, I2P, RNode, Serial, KISS, AX.25, Pipe and custom Python module interfaces are supported.
- Interface Access Codes (IFAC) can authenticate or isolate virtual networks by requiring an Ed25519 signature on every packet.
- Network Identities can sign discovery announces for federated or whitelisted infrastructure.

### Programming model

- Import `RNS`. Create one `RNS.Reticulum(configdir, loglevel...)` instance.
- Create or load an `RNS.Identity`.
- Create an `RNS.Destination(identity, direction, type, app_name, *aspects)`.
- Set callbacks: `set_packet_callback`, `set_link_established_callback`, `set_proof_requested_callback`.
- To send data to a known destination, create an outgoing destination, then create a `RNS.Packet` and call `.send()`.
- For reliable streams or large data, establish a `RNS.Link` and use `RNS.Resource` or `RNS.Channel`.
- For request/response, use `destination.register_request_handler` and `link.request`.
- Paths are established automatically by announces and path requests. The `RNS.Transport` layer handles convergence.

## CLI utilities

The `rns` package installs the following command-line utilities. The `rns_util_help` tool in `rns-mcp` returns the full `--help` for each.

### rnsd
Start the Reticulum daemon.
- `rnsd [--config CONFIG] [-v] [-q] [-s] [-i] [--exampleconfig] [--version]`
- `-s, --service` service mode, log to file.
- `-i, --interactive` drop into an interactive shell after init.
- `--exampleconfig` print a verbose example config to stdout and exit.

### rnstatus
Show status and interface health.
- `rnstatus [filter] [-a] [-A] [-P] [-l] [-B] [-b] [-t] [-p] [-q] [-z] [-s SORT] [-r] [-j] [-R hash] [-i path] [-w seconds] [-d] [-D] [-m] [-I seconds] [-v]`
- `-a, --all` show all interfaces.
- `-A, --announce-stats` show announce stats.
- `-P, --pr-stats` show path request stats.
- `-l, --link-stats` show link stats.
- `-t, --totals` display traffic totals.
- `-p, --pps` packets per second in totals.
- `-q, --queues` queue stats.
- `-j, --json` output in JSON.
- `-d, --discovered` list discovered interfaces.
- `-D` show details for discovered interfaces.
- `-m, --monitor` continuous monitor.
- `-R hash` query a remote transport instance.
- `-i path` identity for remote management.

### rnpath
Manage and query paths.
- `rnpath [destination] [list_filter] [-t] [-m hops] [-r] [-d] [-x] [-w seconds] [-R hash] [-i path] [-W seconds] [-b] [-B] [-U] [--duration HOURS] [--reason REASON] [-p] [-j] [-v]`
- `-t, --table` show all known paths.
- `-m, --max hops` filter by max hops.
- `-d, --drop` remove path to a destination.
- `-x, --drop-via` drop all paths via a transport instance.
- `-b, --blackholed` list blackholed identities.
- `-B, --blackhole` blackhole an identity.
- `-U, --unblackhole` unblackhole.
- `-p, --blackholed-list` view published blackhole list for a remote transport.
- `-j, --json` output JSON.

### rnprobe
Probe reachability and latency.
- `rnprobe [full_name] [destination_hash] [-s SIZE] [-n PROBES] [-t seconds] [-w seconds] [-v]`
- `-s SIZE` probe payload size.
- `-n PROBES` number of probes.
- `-t seconds` timeout.
- `-w seconds` wait between probes.

### rncp
File transfer.
- `rncp [file] [destination] [--config path] [-v] [-q] [-S] [-l] [-C] [-F] [-f] [-j path] [-s path] [-O] [-b seconds] [-a allowed_hash] [-n] [-p] [-i identity] [-w seconds] [-P]`
- `-l, --listen` listen for incoming transfer requests.
- `-f, --fetch` fetch a file from a remote listener instead of sending.
- `-F, --allow-fetch` allow authenticated clients to fetch files.
- `-j path` restrict fetch requests to a path.
- `-s path` save received files to a path.
- `-a allowed_hash` allow this identity.
- `-n, --no-auth` accept from anyone.
- `-P, --phy-rates` display physical layer transfer rates.

### rnid
Identity and encryption utility.
- `rnid [-i rid] [-g path] [-m rid] [-M rid] [-x] [-X] [-a [aspects]] [-H aspects] [-d [file ...]] [-e [file ...]] [-V [path ...]] [-s [path ...]] [-S text] [-E [path]] [--raw] [-w path] [-r path] [-f] [-R] [-N] [-t seconds] [-p] [-P] [-B] [-b] [-U] [-F] [--meta]`
- `-g path` generate a new identity and save to path.
- `-i rid` specify an identity, destination hash or identity file.
- `-H aspects` show destination hashes for aspects.
- `-d [file ...]` decrypt files.
- `-e [file ...]` encrypt files.
- `-s [path ...]` sign files.
- `-V [path ...]` validate signatures.
- `-S text` create an embedded signed message.
- `-x` export public identity.
- `-X` export private identity.
- `-R` request unknown identities from the network.
- `-p` print identity info.
- `-P` print private keys (careful).
- `-B, -b, -U, -F` choose base32, base64, base256 or hex encoding.

### rnx
Remote execution.
- `rnx [destination] [command] [--config path] [-v] [-q] [-p] [-l] [-i identity] [-x] [-b] [-a allowed_hash] [-n] [-N] [-d] [-m] [-w seconds] [-W seconds]`
- `-l, --listen` listen for commands.
- `-a allowed_hash` accept from this identity.
- `-n, --noauth` accept from anyone.
- `-x, --interactive` enter interactive mode.
- `-m` mirror remote exit code.

### rnsh
Remote shell.
- `rnsh [destination] [--config CONFIG] [--rnsconfig RNSCONFIG] [--identity IDENTITY] [-l] [-s SERVICE] [-b PERIOD] [-a HASH] [-n] [-A] [-C] [-N] [-m] [-w SECONDS]`
- `-l, --listen` server mode.
- `-s SERVICE` service name for identity.
- `-a HASH` allow this identity (repeatable).
- `-n, --no-auth` allow any identity.
- `-A` concatenate remote command to default program args.
- `-C, --no-remote-command` disable executing command lines from remote initiator.
- `-m, --mirror` return with remote process exit code.
- Use `--` to separate `rnsh` options from the command, e.g. `rnsh <destination> -- ls -la /tmp`.

### rngit
Git repository node over Reticulum.
- `rngit [--config CONFIG] [--rnsconfig RNSCONFIG] [-p] [-s] [-i] [-v] [-q] [--version]`
- `-p, --print-identity` print identity and destination info.
- `-s, --service` service mode, log to file.
- `-i, --interactive` drop into an interactive shell.

### rnodeconf
RNode configuration and firmware utility.
- `rnodeconf [port] [-i] [-a] [-u] [-U] [--fw-version version] [--fw-url url] [--nocheck] [-e] [-E] [-C] [-N] [-T] [-b] [-B] [-p] [-w mode] [--channel channel] [--ssid ssid] [--psk psk] [--ip ip] [--nm nm] [-D i] [-t s] [-R rotation] [--freq Hz] [--bw Hz] [--txp dBm] [--sf factor] [--cr rate] [-x] [-X] [-c] [--eeprom-backup] [--eeprom-dump] [--eeprom-wipe] [-P] [--trust-key hexbytes] [-f] [-r] [-k] [-S] [-H FIRMWARE_HASH]`
- `-i, --info` show device info.
- `-a, --autoinstall` automatic install on supported devices.
- `-u, --update` update firmware.
- `-U, --force-update` force update.
- `-N, --normal` switch to normal mode.
- `-T, --tnc` switch to TNC mode.
- `-b, --bluetooth-on` / `-B, --bluetooth-off` / `-p, --bluetooth-pair` Bluetooth controls.
- `-w mode` set WiFi mode (OFF, AP or STATION).
- `--freq Hz`, `--bw Hz`, `--txp dBm`, `--sf factor`, `--cr rate` TNC radio settings.
- `-x, --ia-enable` / `-X, --ia-disable` interference avoidance.
- `-f, --flash` flash firmware and bootstrap EEPROM.
- `-r, --rom` bootstrap EEPROM without flashing.
- `-k, --key` generate a new signing key.

### nomadnet
Nomad Network client.
- `nomadnet [--config CONFIG] [--rnsconfig RNSCONFIG] [-t] [-d] [-c] [--version]`
- `-t, --textui` run in text-UI mode.
- `-d, --daemon` run in daemon mode.
- `-c, --console` in daemon mode, log to console instead of file.

### lxmd
LXMF daemon and propagation node.
- `lxmd [--config CONFIG] [--rnsconfig RNSCONFIG] [-p] [-i PATH] [-v] [-q] [-s] [--status] [--peers] [--sync SYNC] [-b UNPEER] [--timeout TIMEOUT] [-r REMOTE] [--identity IDENTITY] [--exampleconfig] [--version]`
- `-p, --propagation-node` run as an LXMF propagation node.
- `-i, --on-inbound PATH` executable to run on each received message.
- `--status` show node status.
- `--peers` show peered nodes.
- `--sync SYNC` request a sync with a peer.
- `-b, --break UNPEER` break peering with a node.
- `-r, --remote REMOTE` remote propagation node destination hash.
- `--identity IDENTITY` identity for remote requests.


## IFAC, discovery and blackhole

- [IFAC](references/ifac.md) - Interface Access Codes
- [Interface discovery](references/discovery.md) - AutoInterface discovery and `discovery_lxmf_address`
- [Blackhole](references/blackhole.md) - Blackhole and distributed blackhole lists

