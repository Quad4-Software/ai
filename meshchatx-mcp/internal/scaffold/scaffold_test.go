// SPDX-License-Identifier: 0BSD
package scaffold

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	files, err := Generate("svelte-feature", "peer-map")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 2 || !strings.Contains(files[0].Content, "registerFeature") {
		t.Fatalf("feature scaffold: %+v", files)
	}
	if !strings.Contains(files[1].Content, `lang="ts"`) {
		t.Fatal("svelte page should use lang=ts")
	}
	if _, err := Generate("plugin", "Bad Name"); err == nil {
		t.Fatal("bad name should fail")
	}
	if _, err := Generate("nonsense", "x"); err == nil {
		t.Fatal("bad kind should fail")
	}
}

func TestChecks(t *testing.T) {
	got, err := Checks("plugin")
	if err != nil || !strings.Contains(got, "pytest") {
		t.Fatalf("checks: %v %v", got, err)
	}
	if _, err := Checks("nope"); err == nil {
		t.Fatal("unknown surface should error")
	}
}
