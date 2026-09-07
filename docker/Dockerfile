# syntax=docker/dockerfile:1
# SPDX-License-Identifier: 0BSD

ARG VERSION="0.2.0"
ARG VCS_REF="unknown"
ARG CREATED="unknown"

FROM golang:1.27-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b AS builder

WORKDIR /build

RUN apt-get update \
 && apt-get install -y --no-install-recommends python3 \
 && rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH
ENV GOTOOLCHAIN=local \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    CGO_ENABLED=0 \
    GOFLAGS=-buildvcs=false

COPY . .

RUN make build

RUN for d in *-mcp; do \
      if [ -f "$d/$d" ]; then \
        mkdir -p "/app/$d" && \
        cp "$d/$d" "/app/$d/$d"; \
      fi \
    done

RUN python3 scripts/gen-configs.py \
      --manifest mcp-servers.json \
      --repo /app \
      --out /app/config

FROM gcr.io/distroless/static-debian13:nonroot@sha256:1c2c046bc09ed40fad370b599a0b1ae7987f55b01e247cf27a7c27cd97e5bbc7

WORKDIR /app

ARG VERSION
ARG VCS_REF
ARG CREATED

COPY --from=builder /app/ /app/

ENV GATEWAY_CONFIG=/app/config/quad4-mcp.json

USER 65532:65532

LABEL org.opencontainers.image.source="https://github.com/Quad4-Software/ai" \
      org.opencontainers.image.url="https://github.com/Quad4-Software/ai" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.title="Quad4 MCP Toolkit" \
      org.opencontainers.image.description="Multi-module stdio MCP server toolkit" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.licenses="0BSD" \
      org.opencontainers.image.vendor="Quad4-Software"

ENTRYPOINT ["/app/gateway-mcp/gateway-mcp"]
