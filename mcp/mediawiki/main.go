// SPDX-License-Identifier: 0BSD
// Command mediawiki is a read-only MCP server for the MediaWiki
// Action API on one configured wiki. Anti-bot interstitials are routed
// around: Anubis proof-of-work is solved natively, Cloudflare and
// other JS challenges go through a FlareSolverr instance when
// MEDIAWIKI_FLARESOLVERR_URL is set.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Quad4-Software/ai/mcp/mediawiki/internal/mcp"
	"github.com/Quad4-Software/ai/mcp/mediawiki/internal/mediawiki"
)

// newClient lazily builds the API client so a missing config does not
// prevent tools/list.
var (
	clientOnce sync.Once
	client     *mediawiki.Client
	clientErr  error
)

func getClient() (*mediawiki.Client, error) {
	clientOnce.Do(func() {
		client, clientErr = mediawiki.NewClient(mediawiki.ConfigFromEnv())
	})
	return client, clientErr
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intArg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

// jout marshals a result map as indented JSON.
func jout(v map[string]any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "status",
			Description: "Probe the configured wiki: reachability, detected " +
				"anti-bot protection, active transport, and FlareSolverr " +
				"health. Use this first when other calls fail.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out := map[string]any{"api_url": c.APIURL()}
				f := c.Fetcher()
				fs := map[string]any{"configured": f.SolverConfigured()}
				if f.SolverConfigured() {
					if v, err := f.SolverHealth(ctx); err == nil {
						fs["reachable"] = true
						fs["version"] = v
					} else {
						fs["reachable"] = false
						fs["error"] = err.Error()
					}
				}
				out["flaresolverr"] = fs
				info, res, err := c.SiteInfo(ctx)
				if err != nil {
					out["reachable"] = false
					out["error"] = err.Error()
					if res != nil && res.Protection.Vendor != "" {
						out["protection"] = res.Protection
					}
					return jout(out)
				}
				out["reachable"] = true
				out["transport"] = res.Via
				if res.Protection.Vendor != "" {
					out["protection_bypassed"] = res.Protection
				}
				if q, ok := info["query"].(map[string]any); ok {
					if g, ok := q["general"].(map[string]any); ok {
						out["sitename"] = g["sitename"]
						out["generator"] = g["generator"]
						out["base"] = g["base"]
					}
				}
				return jout(out)
			},
		},
		{
			Name:        "search",
			Description: "Full-text search of wiki page titles and content.",
			InputSchema: obj(map[string]any{
				"query": strArg("search text, supports MediaWiki syntax such as intitle:, incategory:, or -exclusion"),
				"limit": intArg("max results, 1-50, default 10"),
			}, "query"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"query": "cargo", "limit": 5}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				hits, info, res, err := c.Search(ctx, a.Query, a.Limit)
				if err != nil {
					return "", err
				}
				return jout(map[string]any{
					"results": hits, "searchinfo": info, "transport": res.Via,
				})
			},
		},
		{
			Name: "page",
			Description: "Fetch a wiki page by title. format=text gives readable " +
				"plain text (default), wikitext gives raw markup, html gives " +
				"rendered HTML. Redirects are followed.",
			InputSchema: obj(map[string]any{
				"title":   strArg("page title, for example Main Page or Category:Foo"),
				"format":  strArg("text (default), wikitext, or html"),
				"section": intArg("optional section index to fetch only part of the page"),
			}, "title"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"title": "Main Page"}`)},
				{"arguments": json.RawMessage(`{"title": "Main Page", "format": "wikitext"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Title   string `json:"title"`
					Format  string `json:"format"`
					Section int    `json:"section"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Title == "" {
					return "", fmt.Errorf("missing required argument: title")
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, res, err := c.Page(ctx, a.Title, a.Format, a.Section)
				if err != nil {
					return "", err
				}
				out["transport"] = res.Via
				return jout(out)
			},
		},
		{
			Name:        "info",
			Description: "Page metadata: existence, length, last touched, URL, and category membership.",
			InputSchema: obj(map[string]any{
				"title": strArg("page title"),
			}, "title"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"title": "Main Page"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Title string `json:"title"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Title == "" {
					return "", fmt.Errorf("missing required argument: title")
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, res, err := c.Info(ctx, a.Title)
				if err != nil {
					return "", err
				}
				out["transport"] = res.Via
				return jout(out)
			},
		},
		{
			Name:        "backlinks",
			Description: "List pages that link to a title (what links here).",
			InputSchema: obj(map[string]any{
				"title": strArg("target page title"),
				"limit": intArg("max results, 1-500, default 50"),
			}, "title"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Title string `json:"title"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Title == "" {
					return "", fmt.Errorf("missing required argument: title")
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, res, err := c.Backlinks(ctx, a.Title, a.Limit)
				if err != nil {
					return "", err
				}
				return jout(map[string]any{
					"title": a.Title, "backlinks": out, "transport": res.Via,
				})
			},
		},
		{
			Name: "category",
			Description: "List members of a category. The name may omit the " +
				"Category: prefix. type filters to page, subcat, or file.",
			InputSchema: obj(map[string]any{
				"name":  strArg("category name, for example Vehicles"),
				"limit": intArg("max results, 1-500, default 50"),
				"type":  strArg("member filter: page, subcat, or file; empty lists all"),
			}, "name"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"name": "Maintenance", "type": "subcat"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Name  string `json:"name"`
					Limit int    `json:"limit"`
					Type  string `json:"type"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				c, err := getClient()
				if err != nil {
					return "", err
				}
				out, res, err := c.CategoryMembers(ctx, a.Name, a.Type, a.Limit)
				if err != nil {
					return "", err
				}
				return jout(map[string]any{
					"members": out, "transport": res.Via,
				})
			},
		},
	}
}

func main() {
	srv := mcp.NewServer("mediawiki", "0.1.0", tools(), nil)
	// Challenge solving can take tens of seconds.
	srv.ToolCallTimeout = 150 * time.Second
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mediawiki:", err)
		os.Exit(1)
	}
}
