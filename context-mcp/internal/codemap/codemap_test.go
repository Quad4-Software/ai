// SPDX-License-Identifier: 0BSD
package codemap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(content), 0o644)
}

func TestOutline(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", "import os\n\ndef alpha(x):\n    return x\n\nclass Beta:\n    pass\n")
	syms, total, err := Outline(filepath.Join(dir, "a.py"))
	if err != nil || total != 8 || len(syms) != 2 {
		t.Fatalf("outline: %v total=%d err=%v", syms, total, err)
	}
	if syms[0].Name != "alpha" || syms[0].Line != 3 || syms[1].Kind != "class" {
		t.Fatalf("symbols: %+v", syms)
	}
}

func TestReadLines(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt", "one\ntwo\nthree\n")
	got, err := ReadLines(filepath.Join(dir, "a.txt"), 2, 2)
	if err != nil || !strings.Contains(got, "2: two") {
		t.Fatalf("read: %q %v", got, err)
	}
	if _, err := ReadLines(filepath.Join(dir, "a.txt"), 5, 2); err == nil {
		t.Fatal("bad range should error")
	}
}

func TestRepoMapRanksByRefs(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", "def shared():\n    pass\n")
	write(t, dir, "b.py", "def lone():\n    pass\n")
	write(t, dir, "c.py", "import a\nx = a.shared()\n")
	out, err := RepoMap(dir, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	// a.py has the referenced symbol so it should rank first
	if !strings.HasPrefix(strings.TrimSpace(out), "a.py") {
		t.Fatalf("map should rank a.py first:\n%s", out)
	}
}

func TestFindReferences(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", "def f():\n    g()\n")
	write(t, dir, "b.py", "def h():\n    g()\n")
	write(t, dir, "skip.txt", "g()\n")
	refs, err := FindReferences(dir, "", "g", 10)
	if err != nil || len(refs) != 2 {
		t.Fatalf("refs: %v %v", refs, err)
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", "x = 1\ny = danger_call(2)\nz = 3\n")
	out, err := Search(dir, "", "danger_call", 1, 10)
	if err != nil || !strings.Contains(out, "y = danger_call") || !strings.Contains(out, "a.py:2") {
		t.Fatalf("search: %s %v", out, err)
	}
}
