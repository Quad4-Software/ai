---
name: reticulum-mesh
description: >
  Use when working with the Reticulum-related servers (rns-mcp, lxmf-mcp,
  lxmfy-mcp, meshchatx-mcp): Reticulum, LXMF, LXST, LXMFy, NomadNet, MeshChatX
  concepts, source mirrors, pip-rns, ~/.reticulum state, interface config,
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
  bot scaffolding.
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

## Servers

- `rns-mcp` serves the Reticulum manual as searchable, section-aware tools with
  low memory usage.
- `lxmf-mcp` reads `~/.reticulum` directly for sanitized config, storage
  inventory, identity names and decoded known destinations.
- `lxmf-mcp` is strictly read-only. It must never write to `~/.reticulum`,
  restart rnsd or expose private keys. Identity files are listed by name only
  and config secrets are redacted.
- `lxmfy-mcp` provides LXMFy docs, bot scaffolding, diagnostics and test
  guidance.
- `meshchatx-mcp` provides MeshChatX documentation tools for lookups and
  browsing.

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

## Plugin / god-file pattern

- The doc servers embed section-aware doc corpora in `internal/docs` and in
  `internal/community` for `rns-mcp`. Keep each server's `main.go` thin with
  tool registration plus handlers delegating to `internal` packages. Avoid
  growing a single god-file. Split handlers by domain.

## Conventions for Reticulum MCP tools

- Read-only by default. Tools never start, stop or reconfigure the daemon.
- Path-jail all file access to `~/.reticulum` (or a configured root), reject
  traversal.
- Redact secrets: passphrases, private keys, tokens, credentials and any field
  that could identify or compromise a mesh peer.
- Return destination hashes in canonical angle-bracket hex form, e.g.
  `<13425ec15b621c1d928589718000d814>`, and never display private key data.
- Cap and paginate large outputs; RNS networks may be low bandwidth and the
  consumers of these tools may be remote.
- Use `#nosec` sparingly and only with an inline justification comment.
