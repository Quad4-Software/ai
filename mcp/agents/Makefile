# SPDX-License-Identifier: 0BSD
BIN := agents

.PHONY: all build test vet fmt lint gosec clean install

all: fmt vet test build

build:
	go build -ldflags="-s -w" -o $(BIN) .

test:
	go test -count=1 ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: fmt vet

gosec:
	gosec ./...

clean:
	rm -f $(BIN)

install: build
	@echo 'mcpServers entry:'
	@echo '  {"command": "$(CURDIR)/$(BIN)"}'
