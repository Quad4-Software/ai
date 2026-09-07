// SPDX-License-Identifier: 0BSD
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root = dir
	os.MkdirAll(filepath.Join(dir, ".agents/skills/demo"), 0o755)
	os.WriteFile(filepath.Join(dir, ".agents/skills/demo/SKILL.md"), []byte("# Demo\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644)
	return dir
}

func TestJailed(t *testing.T) {
	dir := setup(t)
	if p, err := jailed(".agents/skills/demo/SKILL.md"); err != nil || !strings.HasPrefix(p, dir) {
		t.Fatalf("jailed: %v %v", p, err)
	}
	for _, bad := range []string{"../x", "../../etc/passwd", "/etc/passwd", "a/../../b"} {
		if _, err := jailed(bad); err == nil {
			t.Fatalf("%q should be refused", bad)
		}
	}
	// symlink escape
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("x"), 0o644)
	link := filepath.Join(dir, "link")
	os.Symlink(outside, link)
	if _, err := jailed("link"); err == nil {
		t.Fatal("symlink escape should be refused")
	}
}

func TestGetSkillAndList(t *testing.T) {
	setup(t)
	got, err := readJailed(".agents/skills/demo/SKILL.md")
	if err != nil || got != "# Demo\n" {
		t.Fatalf("readJailed: %q %v", got, err)
	}
	ents, err := listDir(".agents/skills")
	if err != nil || len(ents) != 1 {
		t.Fatalf("listDir: %v %v", ents, err)
	}
}
