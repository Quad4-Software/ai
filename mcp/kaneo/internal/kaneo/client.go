// SPDX-License-Identifier: 0BSD
// Package kaneo implements a minimal client for the Kaneo REST API.
// It targets self-hosted and cloud instances alike.
//
// Token handling rules:
//   - the API key comes from KANEO_API_KEY or ~/.config/kaneo/config.json
//   - the key is never returned in tool output or error strings
//   - a config file holding a key must not be group/world readable
//   - write tools are opt-in: they only work when a key is configured
package kaneo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultAPI is the hosted Kaneo API. Self-hosted installs set
// KANEO_API_URL or apiUrl in the config file.
const DefaultAPI = "https://cloud.kaneo.app/api"

// Config holds connection settings. APIKey stays unexported-facing:
// methods use it for the Authorization header and never surface it.
type Config struct {
	APIURL      string `json:"apiUrl"`
	APIKey      string `json:"apiKey"`
	WorkspaceID string `json:"workspaceId"`
	ProjectID   string `json:"projectId"`

	// Source reports where the key came from: "env", "config", or "".
	Source string `json:"-"`
}

// ConfigPath returns ~/.config/kaneo/config.json.
func ConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "kaneo", "config.json"), nil
}

var getenv = os.Getenv

// LoadConfig resolves connection settings. Environment wins over the
// config file. A config file containing an API key must be
// owner-readable only; anything looser is rejected with guidance.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		APIURL:      strings.TrimSpace(getenv("KANEO_API_URL")),
		WorkspaceID: strings.TrimSpace(getenv("KANEO_WORKSPACE_ID")),
		ProjectID:   strings.TrimSpace(getenv("KANEO_PROJECT_ID")),
	}
	if k := strings.TrimSpace(getenv("KANEO_API_KEY")); k != "" {
		cfg.APIKey = k
		cfg.Source = "env"
	}

	p, err := ConfigPath()
	if err == nil {
		if data, err := os.ReadFile(p); err == nil { // #nosec G304 -- fixed config path under user config dir
			var file Config
			if json.Unmarshal(data, &file) == nil {
				if cfg.APIURL == "" {
					cfg.APIURL = file.APIURL
				}
				if cfg.WorkspaceID == "" {
					cfg.WorkspaceID = file.WorkspaceID
				}
				if cfg.ProjectID == "" {
					cfg.ProjectID = file.ProjectID
				}
				if cfg.APIKey == "" && file.APIKey != "" {
					if err := checkPerms(p); err != nil {
						return nil, err
					}
					cfg.APIKey = file.APIKey
					cfg.Source = "config"
				}
			}
		}
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPI
	}
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	return cfg, nil
}

// checkPerms refuses to load a key from a file others can read.
func checkPerms(p string) error {
	if runtime.GOOS == "windows" {
		return nil // ACLs differ; owner-only is the platform default
	}
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("config %s is readable by others; run: chmod 600 %s", p, p)
	}
	return nil
}

// Client talks to one Kaneo instance.
type Client struct {
	cfg  Config
	HTTP *http.Client
}

// NewClient builds a Client from resolved config. apiURL must be https
// unless it points at loopback (dev instances).
func NewClient(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.APIURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid api URL %q", cfg.APIURL)
	}
	if u.Scheme != "https" {
		h := u.Hostname()
		if !(u.Scheme == "http" && (h == "localhost" || h == "127.0.0.1" || h == "::1")) {
			return nil, fmt.Errorf("api URL must be https (http allowed for loopback only)")
		}
	}
	return &Client{cfg: cfg, HTTP: &http.Client{Timeout: 15 * time.Second}}, nil
}

// Configured reports whether an API key is available.
func (c *Client) Configured() bool { return c.cfg.APIKey != "" }

// ProjectID returns the default project when pid is empty.
func (c *Client) ProjectID(pid string) string {
	if pid != "" {
		return pid
	}
	return c.cfg.ProjectID
}

// WorkspaceID returns the default workspace when wid is empty.
func (c *Client) WorkspaceID(wid string) string {
	if wid != "" {
		return wid
	}
	return c.cfg.WorkspaceID
}

// do performs one API call. payload may be nil for GET/DELETE. When out
// is non-nil the response body is decoded into it. Error text never
// carries the API key.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, payload, out any) error {
	u := c.cfg.APIURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "kaneo-mcp")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 300 {
			msg = msg[:300] + "..."
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("kaneo %s %s: %d %s", method, path, resp.StatusCode, msg)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("kaneo %s %s: bad response json", method, path)
		}
	}
	return nil
}
