// SPDX-License-Identifier: 0BSD
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Quad4-Software/ai/mcp/melovian/internal/melovian"
)

// fakeMelovian serves the endpoints the client touches and records the
// auth flow.
type fakeMelovian struct {
	*httptest.Server
	loginCalls int
	gotCookie  bool
}

func newFakeMelovian(t *testing.T) *fakeMelovian {
	t.Helper()
	f := &fakeMelovian{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		f.loginCalls++
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Username != "u" || body.Password != "p" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "melovian_session", Value: "sess-1", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	authed := func(w http.ResponseWriter, r *http.Request) bool {
		c, err := r.Cookie("melovian_session")
		if err != nil || c.Value != "sess-1" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return false
		}
		f.gotCookie = true
		return true
	}
	mux.HandleFunc("/api/local-music/search", func(w http.ResponseWriter, r *http.Request) {
		if !authed(w, r) {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"artists": []any{}, "albums": []any{},
			"songs": []map[string]any{{"id": "tr-1", "title": "X", "artist": "Y"}},
		})
	})
	mux.HandleFunc("/api/local-music/songs/tr-1", func(w http.ResponseWriter, r *http.Request) {
		if !authed(w, r) {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "tr-1", "title": "X", "artist": "Y",
			"album": "Z", "albumId": "al-1", "duration": 200000,
		})
	})
	mux.HandleFunc("/api/music/playlists", func(w http.ResponseWriter, r *http.Request) {
		if !authed(w, r) {
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "pl-1", "name": "Mix"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"playlists": []any{}})
	})
	mux.HandleFunc("/api/music/playlists/pl-1/tracks", func(w http.ResponseWriter, r *http.Request) {
		if !authed(w, r) {
			return
		}
		var body struct {
			Tracks []map[string]any `json:"tracks"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Tracks) != 1 || body.Tracks[0]["trackTitle"] != "X" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad tracks"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"saved": true})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Server.Close)
	return f
}

// resetClient swaps the package-level lazy client for a test client.
// The once is consumed so getClient never re-initializes from env.
func resetClient(t *testing.T, c *melovian.Client) {
	t.Helper()
	client = c
	clientErr = nil
	clientOnce = sync.Once{}
	clientOnce.Do(func() {})
	t.Cleanup(func() { clientOnce = sync.Once{}; client = nil; clientErr = nil })
}

func toolByName(t *testing.T, name string) func(context.Context, json.RawMessage) (string, error) {
	t.Helper()
	for _, tool := range tools() {
		if tool.Name == name {
			return tool.Handle
		}
	}
	t.Fatalf("no tool %q", name)
	return nil
}

func TestSearchAuthenticatesAndRetries(t *testing.T) {
	f := newFakeMelovian(t)
	c, err := melovian.NewClient(melovian.Config{URL: f.URL, Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	resetClient(t, c)
	out, err := toolByName(t, "search")(context.Background(), json.RawMessage(`{"q":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "tr-1") {
		t.Fatalf("unexpected output: %s", out)
	}
	if f.loginCalls != 1 || !f.gotCookie {
		t.Fatalf("login=%d cookie=%v", f.loginCalls, f.gotCookie)
	}
}

func TestPlaylistSetTracksResolvesIDs(t *testing.T) {
	f := newFakeMelovian(t)
	c, err := melovian.NewClient(melovian.Config{URL: f.URL, Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	resetClient(t, c)
	out, err := toolByName(t, "playlist_set_tracks")(context.Background(),
		json.RawMessage(`{"playlistId":"pl-1","trackIds":["tr-1"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"saved": true`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSearchMissingQuery(t *testing.T) {
	resetClient(t, nil)
	if _, err := toolByName(t, "search")(context.Background(), json.RawMessage(`{}`)); err == nil ||
		!strings.Contains(err.Error(), "q") {
		t.Fatalf("want missing-arg error naming q, got %v", err)
	}
}

func TestValidIDRejectsTraversal(t *testing.T) {
	for _, bad := range []string{"../x", "a/b", "a b", "", "x.y"} {
		if err := validID("trackId", bad); err == nil {
			t.Fatalf("want rejection for %q", bad)
		}
	}
	if err := validID("trackId", "tr-1_ABC"); err != nil {
		t.Fatal(err)
	}
}

func TestNewClientURLRules(t *testing.T) {
	if _, err := melovian.NewClient(melovian.Config{}); err == nil {
		t.Fatal("empty URL should error")
	}
	if _, err := melovian.NewClient(melovian.Config{URL: "http://evil.example"}); err == nil {
		t.Fatal("non-loopback http should error")
	}
	if _, err := melovian.NewClient(melovian.Config{URL: "http://127.0.0.1:4533"}); err != nil {
		t.Fatal(err)
	}
}

func TestWebSearchUnconfigured(t *testing.T) {
	c, err := melovian.NewClient(melovian.Config{URL: "http://127.0.0.1:4533"})
	if err != nil {
		t.Fatal(err)
	}
	resetClient(t, c)
	if _, err := toolByName(t, "web_search")(context.Background(), json.RawMessage(`{"q":"x"}`)); err == nil ||
		!strings.Contains(err.Error(), "MELOVIAN_SEARXNG_URL") {
		t.Fatalf("want disabled error naming the env var, got %v", err)
	}
}

func TestScaffoldExtension(t *testing.T) {
	out, err := toolByName(t, "scaffold_extension")(context.Background(),
		json.RawMessage(`{"id":"mood-lights","name":"Mood Lights"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"melovian-extension.json", "script.ts", "mood-lights.css", "CHANGELOG.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in %s", want, out)
		}
	}
	for _, bad := range []string{"Bad_ID", "a", "x--y", "-lead"} {
		if _, err := toolByName(t, "scaffold_extension")(context.Background(),
			json.RawMessage(`{"id":"`+bad+`"}`)); err == nil {
			t.Fatalf("want rejection for %q", bad)
		}
	}
}
