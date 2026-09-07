# SPDX-License-Identifier: 0BSD
# Orchestrates every server dir. Per-server targets live in each Makefile.
SERVERS := $(wildcard *-mcp)

.PHONY: all build test vet fmt go-fix gosec clean install golangci release

all: fmt go-fix vet test build

go-fix:
	@for d in $(SERVERS); do \
		(cd $$d && go fix $$(go list ./... | grep -v /third_party/)) || exit 1; \
	done

golangci:
	@for d in $(SERVERS); do \
		(cd $$d && golangci-lint run ./...) || exit 1; \
	done

install: build
	@echo "=== One-entry MCP client config (paste into your client) ==="
	@python3 scripts/gen-configs.py --print

mcp-config:
	@python3 scripts/gen-configs.py --repo $(CURDIR) --out dist

server-json: build
	@python3 scripts/gen-configs.py --repo $(CURDIR) --out dist --server-json

inspector: build mcp-config
	@python3 scripts/mcp-inspector.py

build test vet fmt gosec clean:
	@for d in $(SERVERS); do \
		$(MAKE) -C $$d $@ || exit 1; \
	done

release:
	goreleaser release --clean
