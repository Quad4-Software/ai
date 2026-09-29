// SPDX-License-Identifier: 0BSD
// Package mediawiki is a read-only client for the MediaWiki Action
// API running over the challenge-aware fetch transport. It targets
// exactly one wiki: the configured base host.
package mediawiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Quad4-Software/ai/mcp/mediawiki/internal/fetch"
)

// Config holds connection settings sourced from the environment.
type Config struct {
	// BaseURL is the wiki root, for example https://wiki.example.org.
	// https is required; http is allowed for loopback only.
	BaseURL string
	// APIURL optionally overrides the Action API endpoint. Default is
	// BaseURL + /api.php.
	APIURL string
	// UserAgent overrides the default request UA.
	UserAgent string
	// Cookies pre-seeds the jar, for example a solved cf_clearance.
	Cookies string
	// FlareSolverrURL enables the remote browser solver.
	FlareSolverrURL string
	// Timeout bounds each HTTP round trip in seconds. <= 0 means 30.
	Timeout int
}

// ConfigFromEnv reads MEDIAWIKI_* variables.
func ConfigFromEnv() Config {
	timeout, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("MEDIAWIKI_TIMEOUT")))
	solver := os.Getenv("MEDIAWIKI_FLARESOLVERR_URL")
	if solver == "" {
		solver = os.Getenv("FLARESOLVERR_URL")
	}
	return Config{
		BaseURL:         os.Getenv("MEDIAWIKI_BASE_URL"),
		APIURL:          os.Getenv("MEDIAWIKI_API_URL"),
		UserAgent:       os.Getenv("MEDIAWIKI_USER_AGENT"),
		Cookies:         os.Getenv("MEDIAWIKI_COOKIES"),
		FlareSolverrURL: solver,
		Timeout:         timeout,
	}
}

// Client queries one MediaWiki instance.
type Client struct {
	base string
	api  string
	f    *fetch.Fetcher
}

// NewClient validates cfg and wires the transport. Network access is
// lazy so a missing wiki never breaks tools/list.
func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		return nil, fmt.Errorf("MEDIAWIKI_BASE_URL is not set; point it at a wiki such as https://wiki.example.org")
	}
	f, err := fetch.NewFetcher(base, fetch.Config{
		UserAgent:       cfg.UserAgent,
		Cookies:         cfg.Cookies,
		FlareSolverrURL: cfg.FlareSolverrURL,
		Timeout:         time.Duration(cfg.Timeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	api := strings.TrimSpace(cfg.APIURL)
	if api == "" {
		api = strings.TrimSuffix(base, "/") + "/api.php"
	}
	return &Client{base: strings.TrimSuffix(base, "/"), api: api, f: f}, nil
}

// APIURL exposes the resolved api.php endpoint for diagnostics.
func (c *Client) APIURL() string { return c.api }

// Fetcher exposes the transport for the status tool.
func (c *Client) Fetcher() *fetch.Fetcher { return c.f }

// call issues one API GET and returns the decoded JSON object plus the
// transport result. MediaWiki error envelopes become Go errors.
func (c *Client) call(ctx context.Context, q url.Values) (map[string]any, *fetch.Result, error) {
	q.Set("format", "json")
	q.Set("formatversion", "2")
	res, err := c.f.Get(ctx, c.api+"?"+q.Encode())
	if err != nil {
		return nil, res, err
	}
	if res.Status < 200 || res.Status > 299 {
		return nil, res, fmt.Errorf("wiki returned status %d", res.Status)
	}
	var m map[string]any
	if err := json.Unmarshal(res.Body, &m); err != nil {
		return nil, res, fmt.Errorf("wiki returned non-JSON (status %d); the api endpoint may be wrong, check MEDIAWIKI_API_URL", res.Status)
	}
	if e, ok := m["error"].(map[string]any); ok {
		return nil, res, fmt.Errorf("wiki api error %s: %s", e["code"], e["info"])
	}
	return m, res, nil
}

// clamp bounds a caller-supplied limit.
func clamp(v, lo, hi, def int) int {
	if v <= 0 {
		return def
	}
	if v < lo {
		return lo
	}
	return min(v, hi)
}

// SiteInfo fetches general site metadata for the status tool.
func (c *Client) SiteInfo(ctx context.Context) (map[string]any, *fetch.Result, error) {
	return c.call(ctx, url.Values{
		"action": {"query"}, "meta": {"siteinfo"}, "siprop": {"general"},
	})
}

// Search runs a full-text search and returns normalized hits plus the
// transport result.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]map[string]any, any, *fetch.Result, error) {
	m, res, err := c.call(ctx, url.Values{
		"action":   {"query"},
		"list":     {"search"},
		"srsearch": {query},
		"srlimit":  {strconv.Itoa(clamp(limit, 1, 50, 10))},
	})
	if err != nil {
		return nil, nil, res, err
	}
	q, _ := m["query"].(map[string]any)
	var hits []map[string]any
	for _, it := range listOf(q["search"]) {
		e, _ := it.(map[string]any)
		hits = append(hits, map[string]any{
			"title":     e["title"],
			"snippet":   stripHTML(str(e["snippet"])),
			"size":      e["size"],
			"wordcount": e["wordcount"],
			"timestamp": e["timestamp"],
		})
	}
	return hits, q["searchinfo"], res, nil
}

// Page fetches page content. format is text, wikitext, or html.
// section is an optional section index. Redirects are followed.
func (c *Client) Page(ctx context.Context, title, format string, section int) (map[string]any, *fetch.Result, error) {
	q := url.Values{
		"action":    {"parse"},
		"page":      {title},
		"redirects": {"1"},
	}
	wantText := format == "" || format == "text"
	switch format {
	case "", "text":
		q.Set("prop", "text|displaytitle|revid|categories")
	case "wikitext":
		q.Set("prop", "wikitext|displaytitle|revid|categories")
	case "html":
		q.Set("prop", "text|displaytitle|revid|categories")
	default:
		return nil, nil, fmt.Errorf("invalid format %q; want text, wikitext, or html", format)
	}
	if section > 0 {
		q.Set("section", strconv.Itoa(section))
	}
	m, res, err := c.call(ctx, q)
	if err != nil {
		return nil, res, err
	}
	p, _ := m["parse"].(map[string]any)
	if len(p) == 0 {
		return nil, res, fmt.Errorf("no parse result for %q", title)
	}
	out := map[string]any{
		"title": p["title"],
		"revid": p["revid"],
	}
	if cats := listOf(p["categories"]); len(cats) > 0 {
		var names []string
		for _, it := range cats {
			if e, ok := it.(map[string]any); ok {
				names = append(names, str(e["category"]))
			}
		}
		out["categories"] = names
	}
	if format == "wikitext" {
		out["format"] = "wikitext"
		out["content"] = str(p["wikitext"])
	} else {
		content := str(p["text"])
		if wantText {
			out["format"] = "text"
			out["content"] = stripHTML(content)
		} else {
			out["format"] = "html"
			out["content"] = content
		}
	}
	return out, res, nil
}

// Info fetches page metadata and category membership.
func (c *Client) Info(ctx context.Context, title string) (map[string]any, *fetch.Result, error) {
	m, res, err := c.call(ctx, url.Values{
		"action":  {"query"},
		"prop":    {"info|categories"},
		"titles":  {title},
		"inprop":  {"url|touched|protection"},
		"cllimit": {"max"},
	})
	if err != nil {
		return nil, res, err
	}
	q, _ := m["query"].(map[string]any)
	pages := listOf(q["pages"])
	if len(pages) == 0 {
		return nil, res, fmt.Errorf("no page record for %q", title)
	}
	p, _ := pages[0].(map[string]any)
	out := map[string]any{
		"title":   p["title"],
		"pageid":  p["pageid"],
		"missing": p["missing"] != nil || false,
		"length":  p["length"],
		"touched": p["touched"],
		"url":     p["fullurl"],
	}
	var cats []string
	for _, it := range listOf(p["categories"]) {
		if e, ok := it.(map[string]any); ok {
			cats = append(cats, str(e["title"]))
		}
	}
	out["categories"] = cats
	return out, res, nil
}

// Backlinks lists pages linking to title.
func (c *Client) Backlinks(ctx context.Context, title string, limit int) ([]map[string]any, *fetch.Result, error) {
	m, res, err := c.call(ctx, url.Values{
		"action":        {"query"},
		"list":          {"backlinks"},
		"bltitle":       {title},
		"bllimit":       {strconv.Itoa(clamp(limit, 1, 500, 50))},
		"blfilterredir": {"nonredirects"},
	})
	if err != nil {
		return nil, res, err
	}
	q, _ := m["query"].(map[string]any)
	var out []map[string]any
	for _, it := range listOf(q["backlinks"]) {
		e, _ := it.(map[string]any)
		out = append(out, map[string]any{"title": e["title"], "pageid": e["pageid"]})
	}
	return out, res, nil
}

// CategoryMembers lists members of a category. name is prefixed with
// "Category:" when it lacks a namespace. cmtype filters page, subcat,
// or file members; empty lists all.
func (c *Client) CategoryMembers(ctx context.Context, name, cmtype string, limit int) ([]map[string]any, *fetch.Result, error) {
	if !strings.Contains(name, ":") {
		name = "Category:" + name
	}
	q := url.Values{
		"action":  {"query"},
		"list":    {"categorymembers"},
		"cmtitle": {name},
		"cmlimit": {strconv.Itoa(clamp(limit, 1, 500, 50))},
	}
	switch cmtype {
	case "", "page", "subcat", "file":
		if cmtype != "" {
			q.Set("cmtype", cmtype)
		}
	default:
		return nil, nil, fmt.Errorf("invalid type %q; want page, subcat, or file", cmtype)
	}
	m, res, err := c.call(ctx, q)
	if err != nil {
		return nil, res, err
	}
	qres, _ := m["query"].(map[string]any)
	var out []map[string]any
	for _, it := range listOf(qres["categorymembers"]) {
		e, _ := it.(map[string]any)
		out = append(out, map[string]any{
			"title": e["title"],
			"type":  e["type"],
		})
	}
	return out, res, nil
}

// helpers tolerate formatversion quirks across wiki versions.
func listOf(v any) []any {
	l, _ := v.([]any)
	return l
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
