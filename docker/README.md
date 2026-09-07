# Docker deployment notes

## Image

`ghcr.io/Quad4-Software/ai/mcp-toolkit:latest`

Multi-arch, rootless, distroless image that contains the gateway and all
individual MCP servers. The gateway is the only entry point and starts child
servers lazily from `/app/config/quad4-mcp.json`.

## Local build and run

```
docker compose -f docker/docker-compose.yml up --build
```

## Public read-only deployment

For an internet-facing instance use the hardened `docker-compose.public.yml`:

```
docker compose -f docker/docker-compose.public.yml up -d
```

This sets `MCP_READ_ONLY=1`, runs as UID 65532, drops all capabilities, mounts
no writable filesystem, and uses `read_only: true` plus `no-new-privileges`.

## Coolify

Use `docker-compose.public.yml` for Coolify. The gateway starts a read-only HTTP
status/health endpoint on port 8080 when `HTTP_PORT` is set. In Coolify assign
a domain to the `mcp` service and use `http(s)://example.com:8080`. The
`expose` list tells the proxy where to route traffic.

## What read-only mode blocks

- `tools/list` does not advertise tools marked `Write: true`.
- `tools/call` for a hidden write tool returns a JSON-RPC error.
- MeshChatX issue create/update/comment/close, memory remember/update/forget,
  workspace task and test execution, and `rns_rngit` mutations are disabled.

## What read-only mode does not block on its own

- Network side effects such as `rns_probe` or `rns_path_lookup` still transmit.
- Tool handlers may still consume CPU, memory, and log output.
- Subprocess-based tools may read more of the container than intended.

For a public instance, keep the container `read_only`, `cap_drop: [ALL]`, and
non-root. Do not mount host paths, `~/.reticulum`, `.env` files, or GitHub
personal access tokens. Tokens are never needed for the read-only tool surface.

## Threat model

Treat the MCP client as untrusted. Read-only mode is an application-level gate,
not a sandbox. The container hardening is the second layer. Combine both.

## Pinning the image

For production, pin to a specific digest instead of `latest`. Find the digest in
the `docker` workflow output or by running:

```
docker manifest inspect ghcr.io/Quad4-Software/ai/mcp-toolkit:latest
```
