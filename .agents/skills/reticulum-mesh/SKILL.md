---
name: reticulum-mesh
description: >
  Use when working with the Reticulum-related servers (rns-mcp, lxmf-mcp,
  lxmfy-mcp, meshchatx-mcp): Reticulum, LXMF, LXMFy, NomadNet, MeshChatX
  concepts, ~/.reticulum state, interface config, and diagnostics.
---

# Reticulum Mesh Concepts

## Core concepts

- **Reticulum (RNS)** is the encrypted mesh networking stack.
- Local state lives in ~/.reticulum.
- That directory holds the config file, storage directory, identities, and
  known destinations.
- **LXMF** is a lightweight messaging protocol over Reticulum that routes
  messages by destinations and delivery stamps.
- **LXMFy** is a Python bot framework on LXMF.
- lxmfy-mcp has searchable docs and bot scaffolding.
- **NomadNet** is an LXMF-based mesh comms app whose pages use Micron markup.
- **MeshChatX** is an LXMF mesh chat client, and meshchatx-mcp exposes its
  docs.

## Servers

- rns-mcp serves the Reticulum manual as searchable, section-aware tools
  with low memory usage.
- lxmf-mcp reads ~/.reticulum directly for sanitized config, storage
  inventory, identity names, and decoded known destinations.
- This server is strictly read-only.
- lxmfy-mcp provides LXMFy docs, bot scaffolding, diagnostics, and test
  guidance.
- meshchatx-mcp provides MeshChatX documentation tools for lookups and
  browsing.

## RNS lifecycle and reload

- The rnsd daemon reads interface and destination config from
  ~/.reticulum/config, and changing those values requires either a full daemon
  restart or a SIGHUP reload on running platforms that support it.
- Never restart rnsd from these read-only tools.
- lxmf-mcp only reads state.
- It must never write to ~/.reticulum, restart rnsd, or expose private keys.
- Identity files are listed by name only, and config secrets are redacted.

## Interface config

~/.reticulum/config holds [[interfaces]] sections for AutoInterface, TCP,
UDP, I2P, RNode, serial, and more, plus logging and storage settings. When
surfacing config, redact passphrases, keys, and any credential fields.

## Plugin / god-file pattern

The doc servers embed section-aware doc corpora in internal/docs and in
internal/community for rns-mcp. Keep each server's main.go thin with tool
registration plus handlers delegating to internal packages. Avoid growing a
single god-file. Split handlers by domain.

## Diagnostics

- rnstatus, rnpath, rncp, and rnid are the standard Reticulum CLI utilities.
- Inspect ~/.reticulum/storage and ~/.reticulum/config to understand state.
- lxmf-mcp tools decode known destinations without needing python beyond the
  bundled decoder usage, because python3 is used solely to decode stored
  blobs.
