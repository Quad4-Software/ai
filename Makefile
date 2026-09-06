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

build test vet fmt gosec clean install:
	@for d in $(SERVERS); do \
		$(MAKE) -C $$d $@ || exit 1; \
	done

release:
	goreleaser release --clean
