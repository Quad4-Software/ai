// SPDX-License-Identifier: 0BSD
package refactor

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

func TestSymbols(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", "def alpha():\n    pass\n\nclass Beta:\n    pass\n")
	syms, n, err := Symbols(filepath.Join(dir, "a.py"))
	if err != nil || len(syms) != 2 || syms[0].Name != "alpha" || syms[1].Kind != "class" {
		t.Fatalf("symbols: %+v n=%d err=%v", syms, n, err)
	}
	if syms[0].End != 3 {
		t.Fatalf("alpha should end at line 3, got %d", syms[0].End)
	}
}

func TestGodFiles(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := range 30 {
		b.WriteString("def f" + strings.Repeat("x", 1) + string(rune('a'+i%26)) + string(rune('a'+i/26)) + "():\n")
		for range 20 {
			b.WriteString("    pass\n")
		}
	}
	write(t, dir, "big.py", b.String())
	write(t, dir, "small.py", "def f():\n    pass\n")
	got, err := GodFiles(dir, "", 10)
	if err != nil || len(got) != 1 || got[0].Path != "big.py" {
		t.Fatalf("god files: %+v err=%v", got, err)
	}
}

func TestSplitPlanGroupsByPrefix(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "m.py", "def load_a():\n    pass\ndef load_b():\n    pass\ndef save_a():\n    pass\ndef misc():\n    pass\n")
	groups, syms, err := SplitPlan(dir, "m.py")
	if err != nil || len(syms) != 4 {
		t.Fatalf("split: %v %+v", err, syms)
	}
	var load, save int
	for _, g := range groups {
		if g.Target == "internal/load" {
			load = len(g.Symbols)
		}
		if g.Target == "internal/save" {
			save = len(g.Symbols)
		}
	}
	if load != 2 || save != 1 {
		t.Fatalf("groups: %+v", groups)
	}
}

func TestSurfaceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "w.py", `registry.register("chat.send", h)
this.$t("settings.title")
emit("closed")
`)
	snap, err := SurfaceSnapshot(dir, "w.py")
	if err != nil || len(snap) == 0 {
		t.Fatalf("snapshot: %v %v", snap, err)
	}
	// simulate a split that lost the WS registration
	write(t, dir, "w.py", `this.$t("settings.title")
emit("closed")
`)
	missing, added, err := SurfaceDiff(dir, "w.py", snap)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range missing {
		if strings.Contains(m, "chat.send") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing should contain chat.send: %v", missing)
	}
	if len(added) != 0 {
		t.Fatalf("unexpected additions: %v", added)
	}
}
