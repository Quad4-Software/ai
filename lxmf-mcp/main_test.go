// SPDX-License-Identifier: 0BSD
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseINIRedaction(t *testing.T) {
	dir := t.TempDir()
	cfg := `[reticulum]
  enable_transport = false
  rpc_key = deadbeef1234

[interfaces]
  [[TCP]]
    type = TCPInterface
    passphrase = secret1
    listen_port = 4242
`
	p := filepath.Join(dir, "config")
	os.WriteFile(p, []byte(cfg), 0o644)
	got, err := parseINI(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "deadbeef1234") || strings.Contains(got, "secret1") {
		t.Fatalf("secrets leaked:\n%s", got)
	}
	if !strings.Contains(got, "enable_transport = false") || !strings.Contains(got, "listen_port = 4242") {
		t.Fatalf("non-secret keys missing:\n%s", got)
	}
	if strings.Contains(got, "#") {
		t.Fatalf("comments should be stripped:\n%s", got)
	}
}
