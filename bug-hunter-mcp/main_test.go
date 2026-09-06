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
