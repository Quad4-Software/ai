// SPDX-License-Identifier: 0BSD
package issues

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCleanProse(t *testing.T) {
	in := "Flags are not languages — they break down; also see `language-icons`.\n\n```\nkeep — this; `code`\n```\n"
	got := CleanProse(in)
	prose, _, _ := strings.Cut(got, "\n\n")
	if strings.Contains(prose, "—") || strings.Contains(prose, ";") {
		t.Fatalf("prose not cleaned: %q", got)
	}
	if !strings.Contains(got, "keep — this; `code`") {
		t.Fatalf("fenced block was modified: %q", got)
	}
	if strings.Contains(got, "`language-icons`") {
		t.Fatalf("inline backticks kept: %q", got)
	}
}

func TestTitle(t *testing.T) {
	got, err := Title("feature", "Language picker")
	if err != nil {
		t.Fatal(err)
	}
	if got != "[Feature]: Language picker" {
		t.Fatalf("got %q", got)
	}
	got, err = Title("bug", "[Bug]: already prefixed")
	if err != nil || got != "[Bug]: already prefixed" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := Title("nope", "x"); err == nil {
		t.Fatal("want error for unknown kind")
	}
	if _, err := Title("bug", "   "); err == nil {
		t.Fatal("want error for empty title")
	}
}

func TestBuildBody(t *testing.T) {
	body, err := BuildBody("feature", map[string]string{
		"version":      "N/A",
		"os":           "All platforms",
		"problem":      "Flags are not languages.",
		"proposal":     "Use native names keyed on BCP 47.\n\n- Native names\n- One generic glyph",
		"alternatives": "flag-icons, rejected",
		"context":      "BCP 47 model packs can be reused.",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{
		"### MeshChatX version", "### Operating system", "### Problem or use case",
		"### Proposed solution", "### Alternatives considered", "### Additional context",
	} {
		if !strings.Contains(body, h) {
			t.Fatalf("missing %s in:\n%s", h, body)
		}
	}
	if _, err := BuildBody("bug", map[string]string{"os": "Linux"}); err == nil ||
		!strings.Contains(err.Error(), "missing required") {
		t.Fatalf("want missing-required error, got %v", err)
	}
	if _, err := BuildBody("bug", map[string]string{
		"version": "1", "os": "Linux", "install": "Docker",
		"description": "x", "reproduction": "y", "bogus": "z",
	}); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("want unknown-field error, got %v", err)
	}
}

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := New("owner/repo", "tok", srv.URL)
	return c, srv
}

func TestCreate(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repos/owner/repo/issues" {
			t.Fatalf("bad request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatal("missing auth header")
		}
		var p map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &p)
		if p["title"] != "[Bug]: x" {
			t.Fatalf("bad title %v", p["title"])
		}
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"number": 5, "title": "[Bug]: x", "state": "open", "html_url": "u"}`)
	})
	it, err := c.Create(context.Background(), "[Bug]: x", "body", []string{"bug"})
	if err != nil {
		t.Fatal(err)
	}
	if it.Number != 5 || it.HTMLURL != "u" {
		t.Fatalf("got %+v", it)
	}
}

func TestUpdateClose(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/repos/owner/repo/issues/7" {
			t.Fatalf("bad request %s %s", r.Method, r.URL.Path)
		}
		var p map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &p)
		if p["state"] != "closed" || p["state_reason"] != "completed" {
			t.Fatalf("bad patch %v", p)
		}
		io.WriteString(w, `{"number": 7, "state": "closed", "html_url": "u"}`)
	})
	closed, reason := "closed", "completed"
	it, err := c.Update(context.Background(), 7, Patch{State: &closed, StateReason: &reason})
	if err != nil || it.State != "closed" {
		t.Fatalf("got %+v err %v", it, err)
	}
}

func TestAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"message": "Not Found"}`)
	})
	if _, err := c.Get(context.Background(), 1); err == nil ||
		!strings.Contains(err.Error(), "Not Found") {
		t.Fatalf("want api error, got %v", err)
	}
}

func TestRepoValidation(t *testing.T) {
	c := New("../../etc", "tok", "http://x")
	if _, err := c.Get(context.Background(), 1); err == nil {
		t.Fatal("want repo validation error")
	}
}

func TestNewClientFromEnvNoToken(t *testing.T) {
	old := getenv
	defer func() { getenv = old }()
	getenv = func(string) string { return "" }
	if _, err := NewClientFromEnv(); err == nil {
		t.Fatal("want error without token")
	}
	getenv = func(k string) string {
		if k == "GH_TOKEN" {
			return "x"
		}
		return ""
	}
	c, err := NewClientFromEnv()
	if err != nil || c.Repo != DefaultRepo {
		t.Fatalf("got %+v err %v", c, err)
	}
}

func TestLinkReferences(t *testing.T) {
	in := "Key everything on BCP 47 tags. Use LXMF and an RNode over KISS.\n" +
		"BCP 47 again on a second line is not re-linked.\n" +
		"See https://reticulum.network/ for context, the URL stays bare.\n" +
		"```\ncode mentions BCP 47 untouched\n```\n"
	got := LinkReferences(in)
	if !strings.Contains(got, "[BCP 47](https://www.rfc-editor.org/rfc/rfc5646)") {
		t.Fatalf("BCP 47 not linked:\n%s", got)
	}
	if !strings.Contains(got, "[LXMF](https://github.com/markqvist/LXMF)") {
		t.Fatalf("LXMF not linked:\n%s", got)
	}
	if !strings.Contains(got, "[RNode](https://unsigned.io/rnode/)") {
		t.Fatalf("RNode not linked:\n%s", got)
	}
	// one link per term per document
	if strings.Count(got, "rfc5646") != 1 {
		t.Fatalf("BCP 47 linked more than once:\n%s", got)
	}
	if strings.Contains(got, "code mentions [BCP 47]") {
		t.Fatalf("fenced code was linked:\n%s", got)
	}
	// a URL containing the term text is not rewritten
	if !strings.Contains(got, "See https://reticulum.network/ for context") {
		t.Fatalf("bare URL was rewritten:\n%s", got)
	}

	// term already used as link text is left alone everywhere
	in2 := "LXMF is great. [LXMF](https://example.com) already linked."
	got2 := LinkReferences(in2)
	if strings.Contains(got2, "markqvist/LXMF") {
		t.Fatalf("already-linked term was re-linked:\n%s", got2)
	}
}
