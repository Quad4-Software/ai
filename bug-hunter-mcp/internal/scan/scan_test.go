// SPDX-License-Identifier: 0BSD
package scan

import (
	"context"
	"os"
	"os/exec"
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

func TestTOCTOU(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", `import os
if os.path.exists(p):
    x = open(p).read()
`)
	write(t, dir, "b.py", `import os
x = open(p).read()
`)
	pairs, err := TOCTOU(dir, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 1 || pairs[0].CheckLine != 2 || pairs[0].UseLine != 3 {
		t.Fatalf("pairs: %+v", pairs)
	}
}

func TestSoftFuzz(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "tests/test_a.py", `def test_x():
    try:
        thing()
    except Exception:
        pass
`)
	write(t, dir, "tests/test_b.py", `def test_y():
    assert thing() == 1
`)
	out, err := SoftFuzzScan(dir, "tests", 50)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, f := range out {
		files = append(files, f.File)
	}
	joined := strings.Join(files, ",")
	if !strings.Contains(joined, "test_a.py") || strings.Contains(joined, "test_b.py") {
		t.Fatalf("soft fuzz findings: %v", out)
	}
}

func TestTOCTOUConfidence(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", `import os
if os.path.exists(p):
    x = open(p).read()
if os.path.exists(a):
    y = open(main_a).read()
`)
	pairs, err := TOCTOU(dir, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 2 {
		t.Fatalf("pairs: %+v", pairs)
	}
	if pairs[0].Confidence != "high" {
		t.Fatalf("same-ident pair should be high: %+v", pairs[0])
	}
	if pairs[1].Confidence != "medium" {
		t.Fatalf("main_a is a different file than a; should be medium: %+v", pairs[1])
	}
}

func TestHotspotsSkipsDataFiles(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("init")
	write(t, dir, "data.json", "{}\n")
	write(t, dir, "code.py", "x=1\n")
	run("add", ".")
	run("commit", "-m", "init")
	hs, err := Hotspots(context.Background(), dir, "2020-01-01", 10)
	if err != nil || len(hs) != 1 || hs[0].Path != "code.py" {
		t.Fatalf("hotspots should skip .json: %+v err=%v", hs, err)
	}
}

func TestSurface(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "srv.py", `import subprocess
subprocess.call(x)
`)
	sinks, err := Surface(dir, "", 50)
	if err != nil || len(sinks) == 0 || sinks[0].Kind != "exec" {
		t.Fatalf("sinks: %v err %v", sinks, err)
	}
}

func TestHotspots(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("init")
	write(t, dir, "hot.py", "x=1\n")
	write(t, dir, "cold.py", "y=1\n")
	run("add", ".")
	run("commit", "-m", "init")
	write(t, dir, "hot.py", "x=2\n")
	run("commit", "-am", "change hot")
	hs, err := Hotspots(context.Background(), dir, "2020-01-01", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) == 0 || hs[0].Path != "hot.py" || hs[0].Commits != 2 {
		t.Fatalf("hotspots: %+v", hs)
	}
}
