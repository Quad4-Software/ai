# SPDX-License-Identifier: 0BSD
BIN := micron
PKGS := $(shell go list ./... | grep -v /third_party/)

.PHONY: all build test vet fmt lint gosec clean install

all: fmt vet test build

build:
	go build -ldflags="-s -w" -o $(BIN) .

test:
	go test -count=1 $(PKGS)

vet:
	go vet $(PKGS)

fmt:
	find . -name '*.go' -not -path './third_party/*' -exec gofmt -w {} +

lint: fmt vet

gosec:
	gosec ./...

clean:
	rm -f $(BIN)

install: build
	@echo 'mcpServers entry:'
	@echo '  {"command": "$(CURDIR)/$(BIN)"}'