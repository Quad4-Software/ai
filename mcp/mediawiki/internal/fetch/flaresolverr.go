// SPDX-License-Identifier: 0BSD
// FlareSolverr client. Speaks the POST /v1 command API: sessions are
// created lazily so a solved Cloudflare clearance survives between
// request.get calls on the FlareSolverr side as well.
package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// fsMaxTimeout is the per-request solve budget handed to FlareSolverr.
const fsMaxTimeout = 90000 // ms

type flareSolverr struct {
	base   *url.URL
	hc     *http.Client
	once   sync.Once
	sess   string
	sesErr error
}

type fsCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

type fsSolution struct {
	Status    int        `json:"status"`
	URL       string     `json:"url"`
	Response  string     `json:"response"`
	UserAgent string     `json:"userAgent"`
	Cookies   []fsCookie `json:"cookies"`
}

type fsResponse struct {
	Status   string      `json:"status"`
	Message  string      `json:"message"`
	Session  string      `json:"session"`
	Version  string      `json:"version"`
	Solution *fsSolution `json:"solution"`
}

// newFlareSolverr validates the endpoint URL. https is required;
// http is allowed for loopback only, matching the wiki base rule.
func newFlareSolverr(rawurl string) (*flareSolverr, error) {
	u, err := url.Parse(strings.TrimSpace(rawurl))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid MEDIAWIKI_FLARESOLVERR_URL %q", rawurl)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil, fmt.Errorf("MEDIAWIKI_FLARESOLVERR_URL must be https (http allowed for loopback only)")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	return &flareSolverr{
		base: u,
		hc:   &http.Client{Timeout: time.Duration(fsMaxTimeout)*time.Millisecond + 30*time.Second},
	}, nil
}

// call posts one command to /v1 and returns the decoded envelope.
func (f *flareSolverr) call(ctx context.Context, payload map[string]any) (*fsResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.base.String()+"/v1", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flaresolverr: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("flaresolverr: read body: %w", err)
	}
	if len(b) > MaxBodyBytes {
		return nil, fmt.Errorf("flaresolverr: response exceeds %d byte cap", MaxBodyBytes)
	}
	var r fsResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("flaresolverr: malformed response (is %s a flaresolverr?)", f.base.Host)
	}
	if r.Status != "ok" {
		msg := r.Message
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return nil, fmt.Errorf("flaresolverr: %s", msg)
	}
	return &r, nil
}

// session returns a persistent browser session id, creating it once.
// A session keeps cleared cookies alive between solves.
func (f *flareSolverr) session(ctx context.Context) string {
	f.once.Do(func() {
		r, err := f.call(ctx, map[string]any{
			"cmd":     "sessions.create",
			"session": "mediawiki-mcp",
		})
		if err != nil || r.Session == "" {
			f.sesErr = err
			return
		}
		f.sess = r.Session
	})
	return f.sess
}

// get runs request.get for url through the shared session.
func (f *flareSolverr) get(ctx context.Context, rawurl string) (*fsSolution, error) {
	payload := map[string]any{
		"cmd":        "request.get",
		"url":        rawurl,
		"maxTimeout": fsMaxTimeout,
	}
	if s := f.session(ctx); s != "" {
		payload["session"] = s
		payload["session_ttl_minutes"] = 30
	}
	r, err := f.call(ctx, payload)
	if err != nil {
		return nil, err
	}
	if r.Solution == nil {
		return nil, fmt.Errorf("flaresolverr: ok status but no solution object")
	}
	return r.Solution, nil
}

// health calls GET / for the version banner. Used by the status tool.
func (f *flareSolverr) health(ctx context.Context) (version string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base.String()+"/", nil)
	if err != nil {
		return "", err
	}
	resp, err := f.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var r struct {
		Version   string `json:"version"`
		Msg       string `json:"msg"`
		UserAgent string `json:"userAgent"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&r); err != nil {
		return "", fmt.Errorf("not a flaresolverr at %s", f.base.Host)
	}
	return r.Version, nil
}

// unwrapBrowserBody strips the DOM wrapper a headless browser adds
// around bare JSON responses: <html><body><pre>payload</pre>.
func unwrapBrowserBody(s string) []byte {
	trim := strings.TrimSpace(s)
	if !strings.HasPrefix(trim, "<") {
		return []byte(s)
	}
	lower := strings.ToLower(trim)
	i := strings.Index(lower, "<pre")
	if i < 0 {
		return []byte(s)
	}
	open := strings.Index(lower[i:], ">")
	if open < 0 {
		return []byte(s)
	}
	start := i + open + 1
	end := strings.LastIndex(lower, "</pre>")
	if end <= start {
		return []byte(s)
	}
	return []byte(html.UnescapeString(trim[start:end]))
}
