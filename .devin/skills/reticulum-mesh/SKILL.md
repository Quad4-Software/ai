---
name: reticulum-mesh
description: >
  Use when working with the Reticulum-related servers (rns-mcp, lxmf-mcp,
  lxmfy-mcp, meshchatx-mcp): Reticulum, LXMF, LXMFy, NomadNet, MeshChatX
  concepts, ~/.reticulum state, interface config, and diagnostics.
---

# Reticulum Mesh Concepts

## Core concepts

- **Reticulum (RNS)**: encrypted mesh networking stack. Local state lives in
  `~/.reticulum` (config file, storage dir, identities, known destinations).
- **LXMF**: lightweight messaging protocol over Reticulum; destinations and
  delivery stamps.
- **LXMFy**: Python bot framework on LXMF; `lxmfy-mcp` has searchable docs and
  bot scaffolding.
- **NomadNet**: LXMF-based mesh comms app; pages use Micron markup.
- **MeshChatX**: LXMF mesh chat client; `meshchatx-mcp` exposes its docs.

## Servers

- `rns-mcp` - Reticulum manual as searchable, section-aware tools; low memory.
- `lxmf-mcp` - reads `~/.reticulum` directly: sanitized config, storage
  inventory, identity names, decoded known destinations. Read-only.
- `lxmfy-mcp` - LXMFy docs, bot scaffolding, diagnostics, test guidance.
- `meshchatx-mcp` - MeshChatX documentation tools.

## RNS lifecycle and reload

- The rnsd daemon reads `~/.reticulum/config`. Changing interface or
  destination config requires an rnsd restart (or SIGHUP where supported).
- `lxmf-mcp` only reads state; it must never write to `~/.reticulum`,
  restart rnsd, or expose private keys. Identity files are listed by name
  only; config secrets are redacted.

## Interface config

`~/.reticulum/config` holds `[[interfaces]]` sections (AutoInterface, TCP,
UDP, I2P, RNode, serial, etc.) plus logging and storage settings. When
surfacing config, redact passphrases, keys, and any credential fields.

## Plugin / god-file pattern

The doc servers embed section-aware doc corpora in `internal/docs` (and
`internal/community` for rns-mcp). Keep each server's `main.go` thin: tool
registration plus handlers delegating to internal packages. Avoid growing a
single god-file; split handlers by domain.

## Diagnostics

- `rnstatus`, `rnpath`, `rncp`, `rnid` - standard Reticulum CLI utilities.
- Inspect `~/.reticulum/storage` and `~/.reticulum/config` for state.
- `lxmf-mcp` tools decode known destinations without needing python beyond
  the bundled decoder usage (python3 used solely to decode stored blobs).
