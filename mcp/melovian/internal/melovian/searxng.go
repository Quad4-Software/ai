// SPDX-License-Identifier: 0BSD
package melovian

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearXNGResult is one organic web result.
type SearXNGResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content,omitempty"`
	Engine  string `json:"engine,omitempty"`
}

// WebSearch queries a configured SearXNG instance. The endpoint is
// env-gated: without MELOVIAN_SEARXNG_URL the tool reports as disabled
// so agents know not to retry.
func (c *Client) WebSearch(ctx context.Context, q string, limit int) (any, error) {
	if q == "" {
		return nil, fmt.Errorf("missing required argument: q")
	}
	if c.cfg.SearXNGURL == "" {
		return nil, fmt.Errorf("web search is not configured; set MELOVIAN_SEARXNG_URL to a SearXNG instance")
	}
	u, err := url.Parse(c.cfg.SearXNGURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid MELOVIAN_SEARXNG_URL %q", c.cfg.SearXNGURL)
	}
	if u.Scheme != "https" {
		h := u.Hostname()
		if !(u.Scheme == "http" && (h == "localhost" || h == "127.0.0.1" || h == "::1")) {
			return nil, fmt.Errorf("MELOVIAN_SEARXNG_URL must be https (http allowed for loopback only)")
		}
	}
	v := url.Values{
		"q":        {q},
		"format":   {"json"},
		"language": {"en"},
	}
	u.RawQuery = v.Encode()

	httpc := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searxng: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("searxng: read body: %w", err)
	}
	if len(body) > MaxBodyBytes {
		return nil, fmt.Errorf("searxng: response exceeds %d byte cap", MaxBodyBytes)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searxng: returned %s", resp.Status)
	}

	var parsed struct {
		Results []SearXNGResult `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("searxng: malformed response: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}
	if len(parsed.Results) > limit {
		parsed.Results = parsed.Results[:limit]
	}
	for i := range parsed.Results {
		if len(parsed.Results[i].Content) > 500 {
			parsed.Results[i].Content = strings.TrimSpace(parsed.Results[i].Content[:500]) + "..."
		}
	}
	return map[string]any{
		"query":   q,
		"results": parsed.Results,
		"count":   len(parsed.Results),
	}, nil
}
