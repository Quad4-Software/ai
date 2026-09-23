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

Use `docker-compose.public.yml` for Coolify. The gateway runs with `-http`,
which serves the HTTP surface on `HTTP_PORT` and blocks on SIGTERM instead of
stdio, so the container stays up without an attached stdin. The compose file
has a Docker `healthcheck` that runs the same binary with `-health-check`,
which is required because the distroless image has no shell or curl.

In Coolify assign a domain to the `mcp` service, for example
`https://mcp.example.com` (no port in the URL). The `SERVICE_FQDN_MCP_8080`
variable in the environment block is what Coolify fills in and what routes the
domain to container port 8080.

## Public HTTP/SSE API

The gateway also speaks the official MCP over SSE transport on `HTTP_PORT`.
This lets any HTTP-capable client (including AI agents) connect without needing
stdio access.

Endpoints:

- `GET /`      JSON status: `name`, `version`, `read_only`, `servers`.
- `GET /healthz`  plain `ok` for load balancers.
- `GET /sse`   Server-Sent Events stream. The first event (`event: endpoint`)
  contains the POST URL for this session. Later `event: message` events carry
  JSON-RPC responses.
- `POST /messages?session=<id>`  send one JSON-RPC request body. Returns
  `202 Accepted`; the result comes back on the matching `/sse` stream.

Example with two terminals:

```
# Terminal 1: open the SSE stream and read the session URL
$ curl -N -H 'Accept: text/event-stream' http://localhost:8080/sse

event: endpoint
data: /messages?session=abc123

event: message
data: {"jsonrpc":"2.0","id":1,"result":{...}}
```

```
# Terminal 2: send requests to the endpoint returned above
$ curl -X POST -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"agent","version":"0.1.0"}}}' \
  'http://localhost:8080/messages?session=abc123'

$ curl -X POST -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  'http://localhost:8080/messages?session=abc123'
```

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
