// SPDX-License-Identifier: 0BSD
package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const cfChallengeBody = `<!DOCTYPE html><html><head><title>Just a moment...</title>` +
	`<script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>` +
	`</head><body>Verify you are human</body></html>`

// newFetcher is a test helper building a Fetcher for an httptest host.
func newFetcher(t *testing.T, base string, cfg Config) *Fetcher {
	t.Helper()
	f, err := NewFetcher(base, cfg)
	if err != nil {
		t.Fatalf("NewFetcher: %v", err)
	}
	return f
}

func TestDirectPassthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()
	f := newFetcher(t, srv.URL, Config{})
	res, err := f.Get(context.Background(), srv.URL+"/api.php?action=query")
	if err != nil {
		t.Fatal(err)
	}
	if res.Via != "direct" || res.Status != 200 || !strings.Contains(string(res.Body), `"ok":true`) {
		t.Fatalf("unexpected result: via=%s status=%d", res.Via, res.Status)
	}
}

func TestHostJail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()
	f := newFetcher(t, srv.URL, Config{})
	if _, err := f.Get(context.Background(), "https://example.com/api.php"); err == nil ||
		!strings.Contains(err.Error(), "off the configured host") {
		t.Fatalf("expected host jail error, got %v", err)
	}
	if _, err := f.Get(context.Background(), "file:///etc/passwd"); err == nil {
		t.Fatal("expected scheme rejection")
	}
	if _, err := f.Get(context.Background(), "https://user:pw@"+srv.Listener.Addr().String()+"/x"); err == nil {
		t.Fatal("expected credential rejection")
	}
}

func TestBlockedNoSolver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(403)
		fmt.Fprint(w, cfChallengeBody)
	}))
	defer srv.Close()
	f := newFetcher(t, srv.URL, Config{})
	_, err := f.Get(context.Background(), srv.URL+"/api.php")
	if err == nil {
		t.Fatal("expected blocked error")
	}
	for _, want := range []string{"cloudflare", "managed-challenge", "MEDIAWIKI_FLARESOLVERR_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

// anubisServer is a minimal Anubis emulation: challenge page with an
// embedded puzzle, a pass-challenge endpoint that validates the PoW
// like lib/challenge/proofofwork does, and a protected api.php.
type anubisServer struct {
	*httptest.Server
	randomData  string
	challengeID string
	difficulty  int
	method      string // fast or metarefresh
	solved      chan struct{}
}

func newAnubisServer(t *testing.T, method string, difficulty int) *anubisServer {
	a := &anubisServer{
		randomData:  "abcdef0123456789",
		challengeID: "018e6d40-0000-7000-8000-testchallenge",
		difficulty:  difficulty,
		method:      method,
		solved:      make(chan struct{}, 1),
	}
	a.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/.within.website/x/cmd/anubis/api/pass-challenge"):
			if r.URL.Query().Get("id") != a.challengeID {
				http.Error(w, "bad id", 400)
				return
			}
			switch a.method {
			case "fast":
				nonce := r.URL.Query().Get("nonce")
				response := r.URL.Query().Get("response")
				if r.URL.Query().Get("elapsedTime") == "" {
					http.Error(w, "missing elapsedTime", 400)
					return
				}
				sum := sha256.Sum256([]byte(a.randomData + nonce))
				want := hex.EncodeToString(sum[:])
				if response != want || !strings.HasPrefix(response, strings.Repeat("0", a.difficulty)) {
					http.Error(w, "bad pow", 403)
					return
				}
			case "metarefresh":
				if r.URL.Query().Get("challenge") != a.randomData {
					http.Error(w, "bad challenge", 403)
					return
				}
			}
			http.SetCookie(w, &http.Cookie{Name: anubisCookie, Value: "jwt-token", Path: "/"})
			http.Redirect(w, r, r.URL.Query().Get("redir"), http.StatusFound)
			select {
			case a.solved <- struct{}{}:
			default:
			}
		case strings.HasPrefix(r.URL.Path, "/api.php"):
			c, err := r.Cookie(anubisCookie)
			if err != nil || c.Value != "jwt-token" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<html><head>`+
					`<script id="anubis_challenge" type="application/json">`+
					`{"challenge":{"id":%q,"randomData":%q,"method":%q,"issuedAt":%q},`+
					`"rules":{"algorithm":%q,"difficulty":%d}}</script>`+
					`<script id="anubis_base_prefix" type="application/json">""</script>`+
					`</head><body>Making sure you are not a bot</body></html>`,
					a.challengeID, a.randomData, a.method,
					time.Now().Add(-time.Hour).Format(time.RFC3339Nano),
					a.method, a.difficulty)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"batchcomplete":true,"query":{"general":{"sitename":"Test Wiki"}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(a.Server.Close)
	return a
}

func TestAnubisPoWEndToEnd(t *testing.T) {
	a := newAnubisServer(t, "fast", 3)
	f := newFetcher(t, a.URL, Config{})
	res, err := f.Get(context.Background(), a.URL+"/api.php?action=query")
	if err != nil {
		t.Fatal(err)
	}
	if res.Via != "anubis:fast" {
		t.Fatalf("via = %q, want anubis:fast", res.Via)
	}
	if !strings.Contains(string(res.Body), "Test Wiki") {
		t.Fatalf("unexpected body %s", res.Body[:min(200, len(res.Body))])
	}
	select {
	case <-a.solved:
	case <-time.After(5 * time.Second):
		t.Fatal("pass-challenge was never called")
	}
}

func TestAnubisMetarefreshEndToEnd(t *testing.T) {
	a := newAnubisServer(t, "metarefresh", 1)
	f := newFetcher(t, a.URL, Config{})
	res, err := f.Get(context.Background(), a.URL+"/api.php")
	if err != nil {
		t.Fatal(err)
	}
	if res.Via != "anubis:metarefresh" {
		t.Fatalf("via = %q", res.Via)
	}
}

func TestAnubisUnsupportedFallsToSolverOrErrors(t *testing.T) {
	a := newAnubisServer(t, "preact", 1)
	f := newFetcher(t, a.URL, Config{})
	_, err := f.Get(context.Background(), a.URL+"/api.php")
	if err == nil || !strings.Contains(err.Error(), "preact") {
		t.Fatalf("expected unsolvable-method error, got %v", err)
	}
}

// flareServer emulates POST /v1 with sessions.create and request.get.
func flareServer(t *testing.T, solution fsSolution) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == http.MethodGet {
			fmt.Fprint(w, `{"msg":"FlareSolverr is ready!","version":"3.3.21","userAgent":"FlareUA"}`)
			return
		}
		var req struct {
			Cmd     string `json:"cmd"`
			Session string `json:"session"`
			URL     string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		switch req.Cmd {
		case "sessions.create":
			fmt.Fprint(w, `{"status":"ok","message":"Session created","session":"mediawiki-mcp"}`)
		case "request.get":
			if req.URL == "" {
				fmt.Fprint(w, `{"status":"error","message":"url mandatory"}`)
				return
			}
			out := map[string]any{"status": "ok", "message": "", "solution": solution}
			_ = json.NewEncoder(w).Encode(out)
		default:
			fmt.Fprintf(w, `{"status":"error","message":"cmd %q invalid"}`, req.Cmd)
		}
	}))
}

func TestFlareSolverrCookieHarvest(t *testing.T) {
	// Wiki unlocks when the cf_clearance cookie and solver UA arrive.
	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("cf_clearance")
		if err == nil && c.Value == "cleared" && r.UserAgent() == "FlareUA/9.9" {
			fmt.Fprint(w, `{"unlocked":true}`)
			return
		}
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(403)
		fmt.Fprint(w, cfChallengeBody)
	}))
	defer wiki.Close()
	fs := flareServer(t, fsSolution{
		Status:    200,
		URL:       wiki.URL + "/api.php",
		Response:  cfChallengeBody,
		UserAgent: "FlareUA/9.9",
		Cookies:   []fsCookie{{Name: "cf_clearance", Value: "cleared", Path: "/"}},
	})
	defer fs.Close()

	f := newFetcher(t, wiki.URL, Config{FlareSolverrURL: fs.URL})
	res, err := f.Get(context.Background(), wiki.URL+"/api.php?action=query")
	if err != nil {
		t.Fatal(err)
	}
	if res.Via != "flaresolverr" || !strings.Contains(string(res.Body), `"unlocked":true`) {
		t.Fatalf("via=%s body=%s", res.Via, res.Body)
	}
	if f.userAgent() != "FlareUA/9.9" {
		t.Fatalf("solver UA not adopted: %q", f.userAgent())
	}
}

func TestFlareSolverrBodyFallback(t *testing.T) {
	// Wiki keeps challenging (clearance bound to solver egress IP), so
	// the solver-rendered JSON-in-pre body is returned instead.
	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(403)
		fmt.Fprint(w, cfChallengeBody)
	}))
	defer wiki.Close()
	fs := flareServer(t, fsSolution{
		Status:   200,
		Response: `<html><head></head><body><pre>{&quot;data&quot;:1}</pre></body></html>`,
	})
	defer fs.Close()

	f := newFetcher(t, wiki.URL, Config{FlareSolverrURL: fs.URL})
	res, err := f.Get(context.Background(), wiki.URL+"/api.php")
	if err != nil {
		t.Fatal(err)
	}
	if res.Via != "flaresolverr" {
		t.Fatalf("via=%s", res.Via)
	}
	var m map[string]any
	if json.Unmarshal(res.Body, &m) != nil || m["data"] != float64(1) {
		t.Fatalf("body not unwrapped JSON: %s", res.Body)
	}
}

func TestFlareSolverrUnreachable(t *testing.T) {
	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(403)
		fmt.Fprint(w, cfChallengeBody)
	}))
	defer wiki.Close()
	dead := httptest.NewServer(http.NotFoundHandler())
	fsURL := dead.URL
	dead.Close() // nothing listening now
	f := newFetcher(t, wiki.URL, Config{FlareSolverrURL: fsURL})
	_, err := f.Get(context.Background(), wiki.URL+"/api.php")
	if err == nil || !strings.Contains(err.Error(), "flaresolverr failed") {
		t.Fatalf("expected solver failure, got %v", err)
	}
}

func TestSeededCookies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		if err == nil && c.Value == "abc123" {
			fmt.Fprint(w, `{"ok":true}`)
			return
		}
		http.Error(w, "no cookie", 403)
	}))
	defer srv.Close()
	f := newFetcher(t, srv.URL, Config{Cookies: "session=abc123; other=x"})
	res, err := f.Get(context.Background(), srv.URL+"/api.php")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 200 {
		t.Fatalf("status %d", res.Status)
	}
}

func TestSolvePoWCorrect(t *testing.T) {
	ctx := context.Background()
	nonce, sum, err := solvePoW(ctx, "deadbeef", 3)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte("deadbeef" + strconv.FormatUint(nonce, 10)))
	if hex.EncodeToString(want[:]) != sum {
		t.Fatal("reported hash does not match sha256(data+nonce)")
	}
	if !strings.HasPrefix(sum, "000") {
		t.Fatalf("sum %s lacks 3 leading zeros", sum)
	}
}

func TestSolvePoWContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _, err := solvePoW(ctx, "deadbeef", 60) // unreachable difficulty
	if err == nil {
		t.Fatal("expected ctx error")
	}
}

func TestZeroPrefix(t *testing.T) {
	if !zeroPrefix([]byte{0, 0, 0x0f}, 5) {
		t.Fatal("5 zero nibbles expected")
	}
	if zeroPrefix([]byte{0, 0, 0x1f}, 5) {
		t.Fatal("only 4 zero nibbles")
	}
	if !zeroPrefix([]byte{0, 0, 0xff}, 4) {
		t.Fatal("4 zero nibbles expected")
	}
}

func TestUnwrapBrowserBody(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"a":1}`, `{"a":1}`},
		{`<html><body><pre>{&quot;a&quot;:1}</pre></body></html>`, `{"a":1}`},
		{`<html><body>no pre</body></html>`, `<html><body>no pre</body></html>`},
	}
	for _, tc := range cases {
		if got := string(unwrapBrowserBody(tc.in)); got != tc.want {
			t.Fatalf("unwrap(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
