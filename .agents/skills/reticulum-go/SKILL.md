---
name: reticulum-go
description: >
  This skill exposes the Reticulum-Go docs through reticulum-go. Use
  when you need the Reticulum-Go package map, API, or Go implementation
  details.
compatibility: stdio-mcp
metadata:
  server: reticulum-go
---

## When to use this skill

- You need the Reticulum-Go docs or package map.
- You are working with the Go implementation of Reticulum.
- You want to compare the Go docs against the Python reference.

## How to use

1. Build reticulum-go: `cd mcp/reticulum-go && go test ./... && go build`.
2. Add the binary to your MCP client config as `reticulum-go`.
3. Call `list_topics` or `search_docs {query}` for docs.
4. Use `fetch_page {url}` for a specific page or `github_repo` for repository info.

## Examples

- "Search the Reticulum-Go docs for `packet` handling."
- "Show the package map topic."
- "List open issues on the Reticulum-Go GitHub repo."

# Reticulum-Go

reticulum-go exposes the Reticulum-Go docs at https://reticulum-go.quad4.io
as searchable, section-aware MCP tools. The server is Go stdlib only, low memory,
and fetches pages on demand over HTTPS.

## Tool reference

Tool details are in [references/tools.md](references/tools.md). This section is optional if reticulum-go is installed.

## Topic list

The docs include overview, getting started, examples, architecture, package map,
API reference, microvm, configuration, interfaces, transport, utilities,
packet debug, identity and destinations, links/channels/resources, cryptography,
embedding and WebAssembly, control API, librns, compatibility, security,
development and testing, and interop timeline.

## Scope

Use this skill for Go Reticulum implementation questions, package layout, CLI
utilities, configuration and interface setup, and comparisons with the Python
reference.
