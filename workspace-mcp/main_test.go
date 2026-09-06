// SPDX-License-Identifier: 0BSD
package main

import (
	"testing"
)

func TestTestCandidates(t *testing.T) {
	got := testCandidates("meshchatx/src/backend/foo.py")
	if len(got) == 0 || got[0] != "tests/backend/test_foo.py" {
		t.Fatalf("py mapping: %v", got)
	}
	got = testCandidates("src/x/bar.ts")
	if got[0] != "tests/frontend/bar.test.js" {
		t.Fatalf("ts mapping: %v", got)
	}
	got = testCandidates("internal/x/y.go")
	if got[0] != "internal/x/y_test.go" {
		t.Fatalf("go mapping: %v", got)
	}
}

func TestValidation(t *testing.T) {
	for _, ok := range []string{"test", "test:backend", "lint-all", "fmt_check"} {
		if !taskRe.MatchString(ok) {
			t.Fatalf("valid task %q rejected", ok)
		}
	}
	for _, bad := range []string{"test;id", "test $(rm)", "-x", "test x", ""} {
		if taskRe.MatchString(bad) {
			t.Fatalf("invalid task %q accepted", bad)
		}
	}
	for _, bad := range []string{"HEAD;rm", "a b", "--all", "x$y"} {
		if refRe.MatchString(bad) {
			t.Fatalf("invalid ref %q accepted", bad)
		}
	}
}
