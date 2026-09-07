// SPDX-License-Identifier: 0BSD
package sysutil

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func mockRunner(t *testing.T, wantName string, wantArgs []string, out string) *Runner {
	r := New()
	r.execFn = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != wantName {
			t.Fatalf("want %s, got %s", wantName, name)
		}
		if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
			t.Fatalf("want args %v, got %v", wantArgs, args)
		}
		return []byte(out), nil
	}
	return r
}

const hash32 = "acacfce6c8e2b39e727973a98895314c"

func TestStatus(t *testing.T) {
	r := mockRunner(t, "rnstatus", []string{"-a"}, "interfaces up")
	got, err := r.Status(context.Background(), true)
	if err != nil || got != "interfaces up" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestPathTable(t *testing.T) {
	r := mockRunner(t, "rnpath", []string{"-t", "-m", "3"}, "table")
	if _, err := r.PathTable(context.Background(), "3"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PathTable(context.Background(), "3; rm -rf /"); err == nil {
		t.Fatal("invalid hops should fail")
	}
}

func TestPathLookupValidation(t *testing.T) {
	r := New()
	r.execFn = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("ok"), nil
	}
	if _, err := r.PathLookup(context.Background(), hash32); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "nothex", hash32 + ";id", strings.Repeat("a", 31), "../etc"} {
		if _, err := r.PathLookup(context.Background(), bad); err == nil {
			t.Fatalf("%q should fail validation", bad)
		}
	}
}

func TestDestinationHashValidation(t *testing.T) {
	r := New()
	r.execFn = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		return []byte("ok"), nil
	}
	if _, err := r.DestinationHash(context.Background(), hash32, "lxmf.delivery"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.DestinationHash(context.Background(), "bad", "lxmf.delivery"); err == nil {
		t.Fatal("bad identity should fail")
	}
	if _, err := r.DestinationHash(context.Background(), hash32, "x;rm -rf /"); err == nil {
		t.Fatal("bad aspects should fail")
	}
}

func TestProbe(t *testing.T) {
	r := mockRunner(t, "rnprobe", []string{"nomadnetwork.node", hash32}, "reply in 2.1s")
	got, err := r.Probe(context.Background(), "nomadnetwork.node", hash32)
	if err != nil || !strings.Contains(got, "reply") {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := r.Probe(context.Background(), "bad`name`", hash32); err == nil {
		t.Fatal("bad app name should fail")
	}
}

func TestHelp(t *testing.T) {
	r := mockRunner(t, "rngit", []string{"--help"}, "usage: rngit")
	got, err := r.Help(context.Background(), "rngit")
	if err != nil || !strings.Contains(got, "rngit") {
		t.Fatalf("got %q err %v", got, err)
	}
	r.execFn = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("should not run"), nil
	}
	for _, bad := range []string{"rm", "rngit;id", "", "RNSTATUS"} {
		if _, err := r.Help(context.Background(), bad); err == nil {
			t.Fatalf("%q should fail allowlist", bad)
		}
	}
}

func TestExecErrorAndEmpty(t *testing.T) {
	r := New()
	r.execFn = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("exit status 1")
	}
	if _, err := r.Status(context.Background(), false); err == nil {
		t.Fatal("empty output plus error should surface error")
	}
	r.execFn = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, nil
	}
	got, _ := r.Status(context.Background(), false)
	if got != "(no output)" {
		t.Fatalf("got %q", got)
	}
}
