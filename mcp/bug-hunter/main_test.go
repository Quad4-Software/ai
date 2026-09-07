// SPDX-License-Identifier: 0BSD
package main

import (
	"strings"
	"testing"
)

func TestMethodSection(t *testing.T) {
	got, err := methodSection("toctou")
	if err != nil || !strings.Contains(got, "check-then-use") {
		t.Fatalf("toctou section: %v %q", err, got[:80])
	}
	if _, err := methodSection("nonsense"); err == nil {
		t.Fatal("unknown method should error")
	}
	got, _ = methodSection("churn")
	if strings.Contains(got, "## Error-path") {
		t.Fatal("section bleed into next section")
	}
}

func TestNewMethodSections(t *testing.T) {
	for _, name := range []string{"supply-chain", "secrets", "injection", "crypto",
		"concurrency", "ci-pipeline", "mcp-security", "authz-matrix", "ai-code"} {
		got, err := methodSection(name)
		if err != nil || len(got) < 100 {
			t.Fatalf("method %q: err=%v len=%d", name, err, len(got))
		}
		if strings.Count(got, "\n## ") != 0 {
			t.Fatalf("method %q section bleeds: %q", name, got[:200])
		}
	}
}
