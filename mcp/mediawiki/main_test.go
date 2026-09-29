// SPDX-License-Identifier: 0BSD
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// resetClient undoes the lazy client init between tests.
func resetClient() {
	clientOnce = sync.Once{}
	client, clientErr = nil, nil
}

func callTool(t *testing.T, name string, args string) (string, error) {
	t.Helper()
	for _, tool := range tools() {
		if tool.Name == name {
			return tool.Handle(context.Background(), json.RawMessage(args))
		}
	}
	t.Fatalf("tool %q not registered", name)
	return "", nil
}

func TestMissingEnv(t *testing.T) {
	resetClient()
	t.Cleanup(resetClient)
	_, err := callTool(t, "search", `{"query":"x"}`)
	if err == nil || !strings.Contains(err.Error(), "MEDIAWIKI_BASE_URL") {
		t.Fatalf("expected env error, got %v", err)
	}
}

func TestEndToEndAgainstFixture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Query().Get("meta") == "siteinfo":
			fmt.Fprint(w, `{"query":{"general":{"sitename":"E2E Wiki"}}}`)
		case r.URL.Query().Get("list") == "search":
			fmt.Fprint(w, `{"query":{"searchinfo":{"totalhits":1},"search":[{"title":"Hit","snippet":"s","size":1}]}}`)
		default:
			fmt.Fprint(w, `{"error":{"code":"x","info":"y"}}`)
		}
	}))
	defer srv.Close()
	resetClient()
	t.Cleanup(resetClient)
	t.Setenv("MEDIAWIKI_BASE_URL", srv.URL)
	t.Setenv("MEDIAWIKI_API_URL", "")
	t.Setenv("MEDIAWIKI_FLARESOLVERR_URL", "")
	t.Setenv("FLARESOLVERR_URL", "")
	t.Setenv("MEDIAWIKI_COOKIES", "")
	t.Setenv("MEDIAWIKI_USER_AGENT", "")
	t.Setenv("MEDIAWIKI_TIMEOUT", "")

	out, err := callTool(t, "search", `{"query":"hit"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"Hit"`) {
		t.Fatalf("search out: %s", out)
	}
	out, err = callTool(t, "status", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"sitename": "E2E Wiki"`) || !strings.Contains(out, `"reachable": true`) {
		t.Fatalf("status out: %s", out)
	}
}

func TestStatusReportsProtection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(403)
		fmt.Fprint(w, `<title>Just a moment...</title>`)
	}))
	defer srv.Close()
	resetClient()
	t.Cleanup(resetClient)
	t.Setenv("MEDIAWIKI_BASE_URL", srv.URL)
	t.Setenv("MEDIAWIKI_API_URL", "")
	t.Setenv("MEDIAWIKI_FLARESOLVERR_URL", "")
	t.Setenv("FLARESOLVERR_URL", "")
	t.Setenv("MEDIAWIKI_COOKIES", "")

	out, err := callTool(t, "status", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"reachable": false`, `"vendor": "cloudflare"`, "managed-challenge"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status %s missing %q", out, want)
		}
	}
}

func TestArgValidation(t *testing.T) {
	for name, args := range map[string]string{
		"search":    `{}`,
		"page":      `{"title":""}`,
		"info":      `{}`,
		"backlinks": `{}`,
		"category":  `{"name":""}`,
	} {
		resetClient()
		if _, err := callTool(t, name, args); err == nil {
			t.Fatalf("%s(%s) should fail validation", name, args)
		}
	}
}
