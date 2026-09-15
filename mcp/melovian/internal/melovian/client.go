// SPDX-License-Identifier: 0BSD
// Package melovian is a bounded HTTP client for the Melovian API.
// It authenticates with a username and password over the app session
// endpoint, keeps the session cookie in memory only, and redacts
// credentials from every error it returns.
package melovian

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// MaxBodyBytes caps one API response. The client rejects larger bodies
// rather than truncating, so callers never see half a JSON document.
const MaxBodyBytes = 4 << 20 // 4 MiB

// Config holds connection settings. Secrets come from the environment
// and are never echoed in tool output or errors.
type Config struct {
	// URL is the base Melovian URL, for example https://melovian.home.
	// https is required; http is allowed for loopback only.
	URL string
	// Username and Password authenticate against /api/auth/login when
	// the instance has auth enabled. Empty means unauthenticated access
	// to whatever the instance allows.
	Username string
	Password string
	// SearXNGURL optionally enables the web_search tool.
	SearXNGURL string
}

// ConfigFromEnv reads MELOVIAN_* environment variables.
func ConfigFromEnv() Config {
	return Config{
		URL:        strings.TrimSpace(os.Getenv("MELOVIAN_URL")),
		Username:   strings.TrimSpace(os.Getenv("MELOVIAN_USERNAME")),
		Password:   os.Getenv("MELOVIAN_PASSWORD"),
		SearXNGURL: strings.TrimSpace(os.Getenv("MELOVIAN_SEARXNG_URL")),
	}
}

// Client talks to one Melovian instance.
type Client struct {
	cfg  Config
	http *http.Client
	base *url.URL
}

// NewClient validates cfg and returns a ready client. The session is
// established lazily on the first authenticated request so a missing
// server does not prevent tools/list.
func NewClient(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("MELOVIAN_URL is not set; point it at a Melovian instance such as http://localhost:4533")
	}
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid MELOVIAN_URL %q", cfg.URL)
	}
	if u.Scheme != "https" {
		h := u.Hostname()
		if !(u.Scheme == "http" && (h == "localhost" || h == "127.0.0.1" || h == "::1")) {
			return nil, fmt.Errorf("MELOVIAN_URL must be https (http allowed for loopback only)")
		}
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar: %w", err)
	}
	return &Client{
		cfg:  cfg,
		base: u,
		http: &http.Client{Timeout: 30 * time.Second, Jar: jar},
	}, nil
}

// authed reports whether credentials were configured.
func (c *Client) authed() bool { return c.cfg.Username != "" }

// login performs the session login. The cookie lands in the jar and is
// never read back or exposed.
func (c *Client) login(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"username": c.cfg.Username,
		"password": c.cfg.Password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.endpoint("/api/auth/login"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login: %w", redactErr(err, c.cfg))
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxBodyBytes))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login: Melovian returned %s", resp.Status)
	}
	return nil
}

// endpoint builds an absolute URL under the configured base. Segments
// are path-escaped individually.
func (c *Client) endpoint(p string, query ...url.Values) string {
	u := *c.base
	u.Path = path.Join(u.Path, p)
	if len(query) > 0 && query[0] != nil {
		u.RawQuery = query[0].Encode()
	}
	return u.String()
}

// api runs one request and decodes the JSON body into out. out may be
// nil for endpoints whose body is not needed. On 401 with credentials
// configured it logs in once and retries.
func (c *Client) api(ctx context.Context, method, p string, query url.Values, in, out any) error {
	body, err := c.apiRaw(ctx, method, p, query, in)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s %s: malformed response: %w", method, p, err)
	}
	return nil
}

func (c *Client) apiRaw(ctx context.Context, method, p string, query url.Values, in any) ([]byte, error) {
	b, code, err := c.do(ctx, method, p, query, in)
	if err != nil {
		return nil, err
	}
	if code == http.StatusUnauthorized && c.authed() {
		if err := c.login(ctx); err != nil {
			return nil, err
		}
		b, code, err = c.do(ctx, method, p, query, in)
		if err != nil {
			return nil, err
		}
	}
	if code < 200 || code > 299 {
		return nil, fmt.Errorf("%s %s: Melovian returned %s", method, p, apiStatus(code, b))
	}
	return b, nil
}

func (c *Client) do(ctx context.Context, method, p string, query url.Values, in any) ([]byte, int, error) {
	var rdr io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return nil, 0, fmt.Errorf("encode request: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(p, query), rdr)
	if err != nil {
		return nil, 0, err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s %s: %w", method, p, redactErr(err, c.cfg))
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%s %s: read body: %w", method, p, err)
	}
	if len(b) > MaxBodyBytes {
		return nil, resp.StatusCode, fmt.Errorf("%s %s: response exceeds %d byte cap", method, p, MaxBodyBytes)
	}
	return b, resp.StatusCode, nil
}

// apiStatus extracts the error field from a Melovian error body when
// present, falling back to the status text. The body is already
// bounded, and the extracted message is capped again for safety.
func apiStatus(code int, body []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		msg := e.Error
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return fmt.Sprintf("%d %s", code, msg)
	}
	return strconv.Itoa(code)
}

// redactErr strips configured secrets from an error string so a
// reflected URL or header can never leak credentials.
func redactErr(err error, cfg Config) error {
	msg := err.Error()
	for _, s := range []string{cfg.Password, cfg.Username} {
		if s != "" {
			msg = strings.ReplaceAll(msg, s, "[redacted]")
		}
	}
	return fmt.Errorf("%s", msg)
}

// Get issues an authenticated GET and decodes the JSON body.
func (c *Client) Get(ctx context.Context, p string, query url.Values, out any) error {
	return c.api(ctx, http.MethodGet, p, query, nil, out)
}

// Post issues an authenticated POST with a JSON body.
func (c *Client) Post(ctx context.Context, p string, in, out any) error {
	return c.api(ctx, http.MethodPost, p, nil, in, out)
}

// Put issues an authenticated PUT with a JSON body.
func (c *Client) Put(ctx context.Context, p string, in, out any) error {
	return c.api(ctx, http.MethodPut, p, nil, in, out)
}

// Delete issues an authenticated DELETE.
func (c *Client) Delete(ctx context.Context, p string, out any) error {
	return c.api(ctx, http.MethodDelete, p, nil, nil, out)
}
