// SPDX-License-Identifier: 0BSD
// Package kaneo implements a minimal client for the Kaneo REST API.
// It targets self-hosted and cloud instances alike.
//
// Token handling rules:
//   - the API key comes from KANEO_API_KEY or the OS keyring
//     (secret-tool / pass); it is never written to config.json
//   - the key is never returned in tool output or error strings
//   - a legacy plaintext key in config.json is migrated to the keyring
//     and stripped from the file on load
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
	"strings"
	"time"
)

// DefaultAPI is the hosted Kaneo API. Self-hosted installs set
// KANEO_API_URL or apiUrl in the config file.
const DefaultAPI = "https://cloud.kaneo.app/api"

// ProjectRef names one project so tools can take "melovian" instead
// of a raw id.
type ProjectRef struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
	WorkspaceID string `json:"workspaceId"`
	ProjectID   string `json:"projectId"`
}

// Config holds connection settings. APIKey is json:"-": it is loaded
// from env or the OS keyring and can never be serialized back out.
type Config struct {
	APIURL         string       `json:"apiUrl"`
	KeyBackend     string       `json:"keyBackend,omitempty"`
	WorkspaceID    string       `json:"workspaceId,omitempty"`
	ProjectID      string       `json:"projectId,omitempty"`
	DefaultProject string       `json:"defaultProject,omitempty"`
	Projects       []ProjectRef `json:"projects,omitempty"`

	APIKey string `json:"-"`

	// Source reports where the key came from: "env", "secret-service",
	// "pass", or "".
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
// config file. The key itself comes from env or the OS keyring; a
// plaintext apiKey left in config.json is migrated into the keyring
// and removed from the file. When no keyring exists the file key is
// still used so existing setups keep working, with a warning.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		APIURL:      strings.TrimSpace(getenv("KANEO_API_URL")),
		WorkspaceID: strings.TrimSpace(getenv("KANEO_WORKSPACE_ID")),
	}
	if k := strings.TrimSpace(getenv("KANEO_API_KEY")); k != "" {
		cfg.APIKey = k
		cfg.Source = "env"
	}
	if pid := strings.TrimSpace(getenv("KANEO_PROJECT_ID")); pid != "" {
		cfg.DefaultProject = pid
	}

	p, err := ConfigPath()
	var legacyKey string
	if err == nil {
		if data, err := os.ReadFile(p); err == nil { // #nosec G304 -- fixed config path under user config dir
			// decode as a map first so unknown keys survive rewrites
			var raw map[string]any
			if json.Unmarshal(data, &raw) == nil {
				legacyKey, _ = raw["apiKey"].(string)
			}
			var file Config
			if json.Unmarshal(data, &file) == nil {
				if cfg.APIURL == "" {
					cfg.APIURL = file.APIURL
				}
				cfg.KeyBackend = file.KeyBackend
				if cfg.WorkspaceID == "" {
					cfg.WorkspaceID = file.WorkspaceID
				}
				cfg.ProjectID = file.ProjectID
				if cfg.DefaultProject == "" {
					cfg.DefaultProject = file.DefaultProject
				}
				cfg.Projects = file.Projects
			}
		}
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPI
	}
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")

	if cfg.APIKey == "" {
		if k, src := LoadKey(cfg.APIURL); k != "" {
			cfg.APIKey = k
			cfg.Source = src
		}
	}

	if legacyKey != "" {
		if cfg.APIKey == "" {
			cfg.APIKey = legacyKey
			cfg.Source = "config-file (plaintext, deprecated)"
		}
		if err := migrateKey(p, cfg.APIURL, legacyKey); err != nil {
			fmt.Fprintf(os.Stderr, "kaneo: plaintext apiKey in %s could not be migrated to a keyring: %v\n", p, err)
		}
	}
	return cfg, nil
}

// migrateKey moves a plaintext config key into the OS keyring and
// rewrites the file without it, preserving every other field.
func migrateKey(path, apiURL, key string) error {
	backend := DetectBackend()
	if backend == BackendNone {
		return fmt.Errorf("no secret backend (install libsecret/secret-tool or pass)")
	}
	if err := StoreKey(backend, apiURL, key); err != nil {
		return err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- fixed config path
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	delete(raw, "apiKey")
	raw["keyBackend"] = backend
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "kaneo: migrated api key into %s and removed it from %s\n", backend, path)
	return nil
}

// SaveConfig writes the non-secret parts of cfg to the config file,
// preserving unknown fields already present. The key is never written.
func SaveConfig(path string, cfg *Config) error {
	raw := map[string]any{}
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- fixed config path
		_ = json.Unmarshal(data, &raw) // #nosec G104 -- corrupt file gets rewritten anyway
	}
	delete(raw, "apiKey")
	raw["apiUrl"] = cfg.APIURL
	if cfg.KeyBackend != "" {
		raw["keyBackend"] = cfg.KeyBackend
	}
	if cfg.WorkspaceID != "" {
		raw["workspaceId"] = cfg.WorkspaceID
	}
	if cfg.DefaultProject != "" {
		raw["defaultProject"] = cfg.DefaultProject
	} else if cfg.ProjectID != "" {
		raw["defaultProject"] = cfg.ProjectID
	}
	if len(cfg.Projects) > 0 {
		raw["projects"] = cfg.Projects
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
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

// ResolveProject maps a project name, slug, or id from the config's
// projects list to its id. Unknown values pass through as ids.
func (c *Config) ResolveProject(ref string) string {
	for _, p := range c.Projects {
		if strings.EqualFold(p.Name, ref) || p.Slug == ref || p.ProjectID == ref {
			return p.ProjectID
		}
	}
	return ref
}

// ProjectID resolves the default project when ref is empty. ref may be
// a configured project name, slug, or raw id.
func (c *Client) ProjectID(ref string) string {
	if ref != "" {
		return c.cfg.ResolveProject(ref)
	}
	if c.cfg.DefaultProject != "" {
		return c.cfg.ResolveProject(c.cfg.DefaultProject)
	}
	return c.cfg.ProjectID
}

// WorkspaceID resolves the default workspace when wid is empty. wid
// may be a workspace name matching a configured project entry, or a
// raw id.
func (c *Client) WorkspaceID(wid string) string {
	if wid != "" {
		for _, p := range c.cfg.Projects {
			if strings.EqualFold(p.Workspace, wid) {
				return p.WorkspaceID
			}
		}
		return wid
	}
	return c.cfg.WorkspaceID
}

// SetDefaultProject switches the session default project. ref may be a
// name, slug, or id.
func (c *Client) SetDefaultProject(ref string) { c.cfg.DefaultProject = ref }

// DefaultProject reports the resolved default project id.
func (c *Client) DefaultProject() string { return c.ProjectID("") }

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
