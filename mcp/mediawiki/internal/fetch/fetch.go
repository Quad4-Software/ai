// SPDX-License-Identifier: 0BSD
// Package fetch is a bounded GET transport pinned to one wiki host.
// It detects anti-bot interstitials and routes around them: Anubis
// proof-of-work challenges are solved natively, everything else can go
// through a FlareSolverr instance when one is configured. Cookies and
// the solver-adopted User-Agent persist in an in-memory jar so a
// challenge only has to be solved once per process.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MaxBodyBytes caps one response body. Larger bodies are rejected
// rather than truncated.
const MaxBodyBytes = 8 << 20

// maxRounds bounds challenge-solve-and-retry cycles for one Get.
const maxRounds = 3

// DefaultUserAgent is sent until a solver supplies a clearance-bound UA.
const DefaultUserAgent = "mediawiki-mcp/0.1 (+https://github.com/Quad4-Software/ai)"

// Config configures a Fetcher. Secrets come from the environment and
// are never echoed in output or errors.
type Config struct {
	// UserAgent overrides DefaultUserAgent.
	UserAgent string
	// Cookies is a "name=value; name=value" list pre-seeded into the
	// jar for the wiki host, for example a solved cf_clearance.
	Cookies string
	// FlareSolverrURL points at a FlareSolverr instance, for example
	// http://localhost:8191. Empty disables the solver.
	FlareSolverrURL string
	// Timeout bounds each HTTP round trip. <= 0 means 30s.
	Timeout time.Duration
}

// Result is one fetched response.
type Result struct {
	Status int
	Header http.Header
	Body   []byte
	// URL is the final URL after redirects.
	URL string
	// Via records the transport: direct, anubis:<algorithm>, or
	// flaresolverr.
	Via string
	// Protection names the interstitial detected on the way, if any.
	Protection Protection
}

// Fetcher GETs URLs on one allowed host. It is safe for concurrent use.
type Fetcher struct {
	host         string
	hc           *http.Client
	hcNoRedirect *http.Client
	jar          *cookiejar.Jar
	fs           *flareSolverr

	mu     sync.Mutex
	ua     string
	redact []string
}

// NewFetcher validates base and cfg and returns a ready transport.
// base must be https; http is allowed for loopback only.
func NewFetcher(base string, cfg Config) (*Fetcher, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid MEDIAWIKI_BASE_URL %q", base)
	}
	if u.Scheme != "https" {
		h := u.Hostname()
		if !(u.Scheme == "http" && isLoopback(h)) {
			return nil, fmt.Errorf("MEDIAWIKI_BASE_URL must be https (http allowed for loopback only)")
		}
	}
	if u.User != nil {
		return nil, fmt.Errorf("MEDIAWIKI_BASE_URL must not embed credentials")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar: %w", err)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	tr := &http.Transport{}
	f := &Fetcher{
		host: strings.ToLower(u.Host),
		jar:  jar,
		hc:   &http.Client{Timeout: timeout, Jar: jar, Transport: tr},
		hcNoRedirect: &http.Client{Timeout: timeout, Jar: jar, Transport: tr,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		ua: cfg.UserAgent,
	}
	if f.ua == "" {
		f.ua = DefaultUserAgent
	}
	if cfg.FlareSolverrURL != "" {
		fs, err := newFlareSolverr(cfg.FlareSolverrURL)
		if err != nil {
			return nil, err
		}
		f.fs = fs
	}
	if err := f.seedCookies(cfg.Cookies, u); err != nil {
		return nil, err
	}
	return f, nil
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]"
}

// seedCookies parses a "a=b; c=d" list into the jar for the base host.
// Values are registered for redaction so transport errors can never
// echo them.
func (f *Fetcher) seedCookies(list string, base *url.URL) error {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	var cs []*http.Cookie
	for pair := range strings.SplitSeq(list, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok || name == "" {
			return fmt.Errorf("MEDIAWIKI_COOKIES: malformed pair %q (want name=value; ...)", pair)
		}
		// #nosec G124 -- replaying a user-supplied cookie client-side; Secure/HttpOnly are server-set attributes
		cs = append(cs, &http.Cookie{Name: name, Value: value})
		f.redact = append(f.redact, value)
	}
	// Host-only cookies: attach to the origin root so they ride every
	// request on this host.
	origin := *base
	origin.Path = "/"
	origin.RawQuery = ""
	origin.Fragment = ""
	f.jar.SetCookies(&origin, cs)
	return nil
}

// userAgent returns the current UA, which may have been replaced by a
// solver-issued one.
func (f *Fetcher) userAgent() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ua
}

// adoptSolution records solver-issued cookies and the matching UA so
// later direct requests carry a valid clearance.
func (f *Fetcher) adoptSolution(rawurl string, sol *fsSolution) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sol.UserAgent != "" {
		f.ua = sol.UserAgent
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return
	}
	var cs []*http.Cookie
	for _, c := range sol.Cookies {
		if c.Name == "" {
			continue
		}
		// #nosec G124 -- replaying solver-issued cookies client-side; flags belong to the origin server
		cs = append(cs, &http.Cookie{Name: c.Name, Value: c.Value, Path: c.Path})
		f.redact = append(f.redact, c.Value)
	}
	if len(cs) > 0 {
		f.jar.SetCookies(u, cs)
	}
}

// Get fetches rawurl, solving anti-bot interstitials as needed.
// rawurl must live on the configured host; anything else is rejected
// so the transport cannot be used as an open proxy.
func (f *Fetcher) Get(ctx context.Context, rawurl string) (*Result, error) {
	if err := f.checkURL(rawurl); err != nil {
		return nil, err
	}
	var last *Result
	var lastErr error
	via := "direct"
	for range maxRounds {
		res, err := f.roundtrip(ctx, rawurl)
		if err != nil {
			return nil, err
		}
		res.Via = via
		res.Protection = classify(res.Status, res.Header, res.Body)
		if res.Protection.Vendor == "" {
			return res, nil
		}
		last = res
		p := res.Protection
		if p.Vendor == "anubis" {
			algo, err := f.solveAnubis(ctx, res, rawurl)
			if err == nil {
				via = "anubis:" + algo
				continue
			}
			lastErr = err
		}
		if f.fs != nil {
			return f.viaFlareSolverr(ctx, rawurl, res)
		}
		return res, blockedError(rawurl, p, lastErr)
	}
	return last, fmt.Errorf("%s still presents a %s %s after %d attempts",
		rawurl, last.Protection.Vendor, last.Protection.Kind, maxRounds)
}

// checkURL enforces the single-host jail.
func (f *Fetcher) checkURL(rawurl string) error {
	u, err := url.Parse(rawurl)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url %q", rawurl)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("url %q: scheme must be http or https", rawurl)
	}
	if u.User != nil {
		return fmt.Errorf("url %q: embedded credentials are not allowed", rawurl)
	}
	if strings.ToLower(u.Host) != f.host {
		return fmt.Errorf("url %q is off the configured host %q", rawurl, f.host)
	}
	return nil
}

// roundtrip performs one GET with the current UA and shared jar. The
// Go transport negotiates gzip itself, which also satisfies Anubis
// deployments that reject clients not advertising gzip support.
func (f *Fetcher) roundtrip(ctx context.Context, rawurl string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent())
	req.Header.Set("Accept", "application/json, text/html;q=0.9, */*;q=0.8")
	resp, err := f.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", rawurl, f.cleanErr(err))
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("get %s: read body: %w", rawurl, err)
	}
	if len(b) > MaxBodyBytes {
		return nil, fmt.Errorf("get %s: response exceeds %d byte cap", rawurl, MaxBodyBytes)
	}
	return &Result{
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   b,
		URL:    resp.Request.URL.String(),
	}, nil
}

// viaFlareSolverr solves the page through a remote browser, harvests
// the issued cookies and UA into the jar, and retries directly. When
// the retried request is still challenged (clearance bound to the
// solver egress IP) the solver-rendered body is used instead.
func (f *Fetcher) viaFlareSolverr(ctx context.Context, rawurl string, blocked *Result) (*Result, error) {
	sol, err := f.fs.get(ctx, rawurl)
	if err != nil {
		return blocked, fmt.Errorf("%s blocked by %s %s; flaresolverr failed: %w",
			rawurl, blocked.Protection.Vendor, blocked.Protection.Kind, f.cleanErr(err))
	}
	f.adoptSolution(rawurl, sol)
	res, err := f.roundtrip(ctx, rawurl)
	if err == nil {
		res.Protection = classify(res.Status, res.Header, res.Body)
		if res.Protection.Vendor == "" {
			res.Via = "flaresolverr"
			return res, nil
		}
	}
	// Clearance did not transfer; use the solver-rendered body. Browser
	// DOM wraps bare JSON in a <pre> element, which is stripped here.
	return &Result{
		Status:     sol.Status,
		Body:       unwrapBrowserBody(sol.Response),
		URL:        rawurl,
		Via:        "flaresolverr",
		Protection: blocked.Protection,
	}, nil
}

// SolverConfigured reports whether a FlareSolverr endpoint is set.
func (f *Fetcher) SolverConfigured() bool { return f.fs != nil }

// SolverHealth pings the solver index endpoint and returns its version.
func (f *Fetcher) SolverHealth(ctx context.Context) (string, error) {
	if f.fs == nil {
		return "", fmt.Errorf("MEDIAWIKI_FLARESOLVERR_URL is not set")
	}
	return f.fs.health(ctx)
}

// blockedError explains what stopped the request and how to fix it.
func blockedError(rawurl string, p Protection, cause error) error {
	hint := "configure MEDIAWIKI_FLARESOLVERR_URL (for example http://localhost:8191) " +
		"or pass solved cookies via MEDIAWIKI_COOKIES with a matching MEDIAWIKI_USER_AGENT"
	if p.Kind == "block" {
		hint = "this is a deny rule, not a solvable challenge; an allowlist change on the wiki side is required"
	}
	msg := fmt.Sprintf("%s blocked by %s %s", rawurl, p.Vendor, p.Kind)
	if p.Detail != "" {
		msg += " (" + p.Detail + ")"
	}
	if cause != nil {
		msg += "; native solver failed: " + cause.Error()
	}
	return fmt.Errorf("%s; %s", msg, hint)
}

// cleanErr strips seeded cookie values from an error string.
func (f *Fetcher) cleanErr(err error) error {
	msg := err.Error()
	f.mu.Lock()
	for _, s := range f.redact {
		if s != "" {
			msg = strings.ReplaceAll(msg, s, "[redacted]")
		}
	}
	f.mu.Unlock()
	return fmt.Errorf("%s", msg)
}
