// SPDX-License-Identifier: 0BSD
// Command meshchatx-mcp is a stdio MCP server exposing the MeshChatX
// documentation as searchable, section-aware tools, plus opt-in GitHub
// issue write tools for the MeshChatX tracker. Stdlib only.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Quad4-Software/ai/meshchatx-mcp/internal/docs"
	"github.com/Quad4-Software/ai/meshchatx-mcp/internal/mcp"
	"github.com/Quad4-Software/ai/meshchatx-mcp/internal/refactor"
	"github.com/Quad4-Software/ai/meshchatx-mcp/internal/scaffold"
)

var root string

// findRoot returns MCP_REPO_ROOT or the nearest .git ancestor of cwd.
// Empty string when no repo is found; local tools error gracefully.
func findRoot() string {
	if r := os.Getenv("MCP_REPO_ROOT"); r != "" {
		if p, err := filepath.Abs(r); err == nil {
			return p
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		up := filepath.Dir(dir)
		if up == dir {
			return ""
		}
		dir = up
	}
}

func needRoot() error {
	if root == "" {
		return fmt.Errorf("no repo root; set MCP_REPO_ROOT or run inside the repo")
	}
	return nil
}

const base = "https://meshchatx.com/docs/"

var topics = []docs.Topic{
	{ID: "overview", Title: "Overview", URL: base + "overview", Summary: "What MeshChatX is: LXMF messaging, LXST calls, NomadNet, relay chat, maps, Reticulum utilities"},
	{ID: "getting-started", Title: "Getting started", URL: base + "getting-started", Summary: "First-run setup, identity, connecting to the network"},
	{ID: "installation", Title: "Installation and setup", URL: base + "installation", Summary: "Install on desktop, headless servers, and mobile"},
	{ID: "building", Title: "Building from source and packaging", URL: base + "building", Summary: "Build MeshChatX from source, Electron packaging"},
	{ID: "development", Title: "Development", URL: base + "development", Summary: "Dev setup, contributing, codebase layout"},
	{ID: "architecture", Title: "Architecture and design", URL: base + "architecture", Summary: "Backend, frontend, IdentityContext, security model, storage"},
	{ID: "messaging", Title: "LXMF messaging", URL: base + "messaging", Summary: "LXMF send/receive, stamps, propagation nodes, attachments"},
	{ID: "audio-calls", Title: "Audio calls (LXST)", URL: base + "audio-calls", Summary: "LXST telephony, duplex, PTT, voicemail"},
	{ID: "nomad-network", Title: "Nomad Network and Mesh Server", URL: base + "nomad-network", Summary: "NomadNet browsing, Mesh Server pages, Micron, page nodes"},
	{ID: "interfaces", Title: "Reticulum interfaces", URL: base + "interfaces", Summary: "Interface config: AutoInterface, TCP, UDP, RNode, KISS, serial"},
	{ID: "tools", Title: "Tools and utilities", URL: base + "tools", Summary: "Built-in tools: rnstatus, rnpath, maps, files, terminal"},
	{ID: "identity-and-security", Title: "Identities, privacy, and security", URL: base + "identity-and-security", Summary: "Identity management, privacy mode, CSRF, auth, sandboxing"},
	{ID: "plugins", Title: "Plugins", URL: base + "plugins", Summary: "Plugin install, permissions, RSG signatures, runtimes"},
	{ID: "rns-link-api", Title: "RNS Link API", URL: base + "rns-link-api", Summary: "Generic RNS Link WebSocket transport for plugins and external apps"},
	{ID: "nomadmesh-pages", Title: "NomadNet page formats", URL: base + "nomadmesh-pages", Summary: "Authoring NomadNet pages: Micron, Markdown, allowed formats"},
	{ID: "raspberry-pi", Title: "Raspberry Pi", URL: base + "raspberry-pi", Summary: "Running MeshChatX on Raspberry Pi"},
	{ID: "android-termux", Title: "Android (Termux)", URL: base + "android-termux", Summary: "Running MeshChatX on Android via Termux"},
	{ID: "quest-sidequest", Title: "Meta Quest (SideQuest)", URL: base + "quest-sidequest", Summary: "Running MeshChatX on Meta Quest via SideQuest"},
	{ID: "linux-sandbox", Title: "Linux sandboxing", URL: base + "linux-sandbox", Summary: "Landlock sandboxing, MESHCHAT_LANDLOCK, SQLite temp_store"},
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools(src *docs.Source) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_topics",
			Description: "List MeshChatX documentation topics (id, title, URL, summary).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, err := json.MarshalIndent(src.List(), "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_topic",
			Description: "Fetch the full markdown text of a MeshChatX docs page by topic id.",
			InputSchema: obj(map[string]any{"id": strArg("topic id from list_topics")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				return src.Get(ctx, strings.TrimSpace(a.ID))
			},
		},
		{
			Name:        "list_sections",
			Description: "List the section headings of a MeshChatX docs page.",
			InputSchema: obj(map[string]any{"id": strArg("topic id from list_topics")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				secs, err := src.Sections(ctx, strings.TrimSpace(a.ID))
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(secs, "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_section",
			Description: "Fetch one section of a docs page by anchor or heading name. Cheaper than get_topic on large pages.",
			InputSchema: obj(map[string]any{
				"id":      strArg("topic id from list_topics"),
				"section": strArg("section anchor or heading substring from list_sections"),
			}, "id", "section"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string `json:"id"`
					Section string `json:"section"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" || a.Section == "" {
					return "", fmt.Errorf("missing required arguments: id, section")
				}
				return src.GetSection(ctx, strings.TrimSpace(a.ID), a.Section)
			},
		},
		{
			Name:        "search_docs",
			Description: "Full-text search across the MeshChatX docs. Returns ranked excerpts with topic and section anchors.",
			InputSchema: obj(map[string]any{
				"query": strArg("search terms"),
				"limit": map[string]any{"type": "integer", "description": "max results, default 5, max 20"},
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				return src.Search(ctx, a.Query, a.Limit)
			},
		},
		{
			Name:          "scaffold",
			Description:   "Generate MeshChatX-convention file templates: svelte-feature, svelte-component, backend-manager, ws-handler, plugin, oracle-test.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"kind": "svelte-feature", "name": "my-feature"}`)}},
			InputSchema: obj(map[string]any{
				"kind": map[string]any{"type": "string", "enum": scaffold.Kinds, "description": "what to scaffold"},
				"name": strArg("lowercase identifier, e.g. peer-map"),
			}, "kind", "name"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Kind string `json:"kind"`
					Name string `json:"name"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Kind == "" || a.Name == "" {
					return "", fmt.Errorf("missing required arguments: kind, name")
				}
				files, err := scaffold.Generate(a.Kind, a.Name)
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(files, "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "checks_for",
			Description: "Return the verification commands for a change surface: svelte, backend, plugin, ws, docs, i18n, electron, landlock.",
			InputSchema: obj(map[string]any{"surface": strArg("change surface")}, "surface"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Surface string `json:"surface"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Surface == "" {
					return "", fmt.Errorf("missing required argument: surface")
				}
				return scaffold.Checks(strings.ToLower(strings.TrimSpace(a.Surface)))
			},
		},
		{
			Name:        "god_files",
			Description: "Rank oversized files by lines, symbol count, and mixed concerns. Hunt split candidates here first.",
			InputSchema: obj(map[string]any{
				"path":  strArg("repo-relative dir, default meshchatx/src"),
				"limit": map[string]any{"type": "integer", "description": "max files, default 20"},
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				if err := needRoot(); err != nil {
					return "", err
				}
				var a struct {
					Path  string `json:"path"`
					Limit int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Path == "" {
					a.Path = "meshchatx/src"
				}
				if a.Limit <= 0 || a.Limit > 200 {
					a.Limit = 20
				}
				fs, err := refactor.GodFiles(root, a.Path, a.Limit)
				if err != nil {
					return "", err
				}
				if len(fs) == 0 {
					return "no god files found under " + a.Path, nil
				}
				b, err := json.MarshalIndent(fs, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "split_plan",
			Description:   "Inventory a god file: every top-level symbol with line ranges, grouped into suggested extraction targets. Use before splitting.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/meshchat.py"}`)}},
			InputSchema:   obj(map[string]any{"file": strArg("repo-relative file path")}, "file"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				if err := needRoot(); err != nil {
					return "", err
				}
				var a struct {
					File string `json:"file"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.File == "" {
					return "", fmt.Errorf("missing required argument: file")
				}
				groups, syms, err := refactor.SplitPlan(root, a.File)
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(map[string]any{
					"groups": groups, "symbols": syms,
				}, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "surface_snapshot",
			Description:   "Capture the observable surface of a file or dir (routes, WS types, i18n keys, emits, symbols, registry calls). Save the JSON before a refactor.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/meshchat.py", "save": "base"}`)}},
			InputSchema:   obj(map[string]any{"path": strArg("repo-relative file or dir")}, "path"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				if err := needRoot(); err != nil {
					return "", err
				}
				var a struct {
					Path string `json:"path"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" {
					return "", fmt.Errorf("missing required argument: path")
				}
				s, err := refactor.SurfaceSnapshot(root, a.Path)
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(s, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "surface_diff",
			Description:   "Compare a saved surface_snapshot against the current surface. Anything missing after a split is a lost feature.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"file": "meshchatx/meshchat.py", "save": "base"}`)}},
			InputSchema: obj(map[string]any{
				"path":   strArg("repo-relative file or dir"),
				"before": strArg("JSON array from a previous surface_snapshot"),
			}, "path", "before"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				if err := needRoot(); err != nil {
					return "", err
				}
				var a struct {
					Path   string `json:"path"`
					Before string `json:"before"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Path == "" || a.Before == "" {
					return "", fmt.Errorf("missing required arguments: path, before")
				}
				var before []string
				if err := json.Unmarshal([]byte(a.Before), &before); err != nil {
					return "", fmt.Errorf("before must be a JSON array: %w", err)
				}
				missing, added, err := refactor.SurfaceDiff(root, a.Path, before)
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(map[string]any{"missing": missing, "added": added}, "", "  ")
				return string(b), err
			},
		},
		{
			Name:          "feature_check",
			Description:   "Verify a frontend feature is fully wired: index.ts, register<Name>Feature export, registerAllFeatures call, routes with existing load() files, i18n keys, test file. Run after scaffold to catch unwired surfaces.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"id": "blocked"}`)}},
			InputSchema: obj(map[string]any{
				"id": strArg("feature dir name, e.g. blocked"),
			}, "id"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(a.ID) {
					return "", fmt.Errorf("invalid id; kebab-case dir name")
				}
				if err := needRoot(); err != nil {
					return "", err
				}
				return featureCheck(root, a.ID)
			},
		},
		{
			Name:          "route_map",
			Description:   "Every registered HTTP route (method, path, file, handler) plus whether the path or handler appears in tests/. Use before changing or adding API surface.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{}`)}},
			InputSchema: obj(map[string]any{
				"path": strArg("repo-relative dir to scan, default meshchatx/src/backend"),
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if err := needRoot(); err != nil {
					return "", err
				}
				routes, err := routeMap(root, a.Path)
				if err != nil {
					return "", err
				}
				untested := 0
				for _, r := range routes {
					if !r.Tested {
						untested++
					}
				}
				b, _ := json.MarshalIndent(map[string]any{
					"count": len(routes), "untested": untested, "routes": routes,
				}, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:          "impact",
			Description:   "Blast radius for a symbol: reference count and files, likely definition site, owning module per module-ownership.md, and probable test files. Use before renaming or changing behaviour.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"symbol": "hotswap_identity"}`)}},
			InputSchema: obj(map[string]any{
				"symbol": strArg("identifier, e.g. hotswap_identity"),
				"limit":  map[string]any{"type": "integer", "description": "max ref files, default 25"},
			}, "symbol"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Symbol == "" {
					return "", fmt.Errorf("missing required argument: symbol")
				}
				if err := needRoot(); err != nil {
					return "", err
				}
				if !regexp.MustCompile(`^[\w$.-]+$`).MatchString(a.Symbol) {
					return "", fmt.Errorf("invalid symbol")
				}
				if a.Limit <= 0 || a.Limit > 200 {
					a.Limit = 25
				}
				return impactReport(root, a.Symbol, a.Limit)
			},
		},
		{
			Name:        "fetch_page",
			Description: "Fetch any page on meshchatx.com as plain text or markdown (host allowlisted). Use for pages not in the topic index.",
			InputSchema: obj(map[string]any{"url": strArg("full URL on meshchatx.com")}, "url"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.URL == "" {
					return "", fmt.Errorf("missing required argument: url")
				}
				return src.FetchURL(ctx, strings.TrimSpace(a.URL))
			},
		},
	}
}

var impactCodeExts = map[string]bool{
	".py": true, ".js": true, ".ts": true, ".vue": true, ".svelte": true, ".go": true,
}

var impactDeclRe = regexp.MustCompile(`(?:def|func|function|class|export\s+\w+)\s+` +
	`[A-Za-z_$][A-Za-z0-9_$]*\s*[(<:=]`)

// impactReport computes the blast radius of a symbol: where it is
// defined, how many files reference it, who owns it, which tests cover
// it. Read-only; repo-wide scan.
func impactReport(root, sym string, limit int) (string, error) {
	nameRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(sym) + `\b`)
	defRe := regexp.MustCompile(`(def|func|class)\s+` + regexp.QuoteMeta(sym) +
		`|(?:function|const|let|export\s+(?:async\s+)?function)\s+` + regexp.QuoteMeta(sym))
	skip := map[string]bool{"node_modules": true, ".git": true, "dist": true,
		"vendor": true, "__pycache__": true, "storage": true, "build": true, "public": true, "assets": true}
	var refFiles []string
	var defFile, defLine string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !impactCodeExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil || len(data) > 2<<20 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		n := 0
		for i, line := range strings.Split(string(data), "\n") {
			if !nameRe.MatchString(line) {
				continue
			}
			n++
			if defFile == "" && defRe.MatchString(line) {
				defFile = filepath.ToSlash(rel)
				defLine = fmt.Sprintf("%d", i+1)
			}
		}
		if n > 0 {
			refFiles = append(refFiles, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(refFiles)
	var b strings.Builder
	fmt.Fprintf(&b, "symbol: %s\n", sym)
	if defFile != "" {
		fmt.Fprintf(&b, "definition: %s:%s\n", defFile, defLine)
	} else {
		b.WriteString("definition: not found (dynamic or external)\n")
	}
	fmt.Fprintf(&b, "referenced in %d file(s):\n", len(refFiles))
	for i, f := range refFiles {
		if i >= limit {
			fmt.Fprintf(&b, "  ... %d more\n", len(refFiles)-limit)
			break
		}
		b.WriteString("  " + f + "\n")
	}
	// owner hint from module-ownership.md rows
	if data, err := os.ReadFile(filepath.Join(root, ".agents", "module-ownership.md")); err == nil { // #nosec G304 -- path jailed to repo root or local config
		for line := range strings.SplitSeq(string(data), "\n") {
			if defFile != "" && strings.Contains(line, "|") {
				for cell := range strings.SplitSeq(line, "|") {
					c := strings.TrimSpace(strings.Trim(cell, " `"))
					if strings.HasPrefix(c, "meshchatx/") && strings.HasPrefix(defFile, strings.TrimSuffix(c, "*")) {
						fmt.Fprintf(&b, "owner hint: %s\n", strings.TrimSpace(line))
						break
					}
				}
			}
		}
	}
	// probable tests
	if defFile != "" {
		stem := strings.TrimSuffix(filepath.Base(defFile), filepath.Ext(defFile))
		d := filepath.ToSlash(filepath.Dir(defFile))
		var tests []string
		for _, c := range []string{
			"tests/backend/test_" + stem + ".py",
			d + "/test_" + stem + ".py",
			"tests/frontend/" + stem + ".test.js",
		} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err == nil {
				tests = append(tests, c)
			}
		}
		if len(tests) > 0 {
			b.WriteString("probable tests: " + strings.Join(tests, ", ") + "\n")
		}
	}
	_ = impactDeclRe
	return b.String(), nil
}

var routeRe = regexp.MustCompile(`@routes\.(get|post|put|delete|patch|head|options)\s*\(\s*["']([^"']+)["']`)

// RouteEntry is one HTTP route and its test coverage signal.
type RouteEntry struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Handler string `json:"handler"`
	Tested  bool   `json:"tested"` // path or handler name appears in tests/
}

// routeMap scans backend route registrations and cross-references
// tests/ for the path literal or handler name.
func routeMap(root, sub string) ([]RouteEntry, error) {
	base := filepath.Join(root, filepath.FromSlash("meshchatx/src/backend"))
	if sub != "" {
		base = filepath.Join(root, filepath.FromSlash(sub))
	}
	if !strings.HasPrefix(base, root) {
		return nil, fmt.Errorf("path escapes repo root")
	}
	var out []RouteEntry
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".py") {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			m := routeRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			handler := ""
			for j := i + 1; j < len(lines) && j < i+4; j++ {
				if hm := regexp.MustCompile(`(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)`).FindStringSubmatch(lines[j]); hm != nil {
					handler = hm[1]
					break
				}
			}
			out = append(out, RouteEntry{
				Method: strings.ToUpper(m[1]), Path: m[2],
				File: filepath.ToSlash(rel), Line: i + 1, Handler: handler,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// test coverage: any test file containing the path tail or handler name
	testDir := filepath.Join(root, "tests")
	var testSrc []string
	// tolerate walk errors; missing or unreadable tests dir is not fatal
	_ = filepath.WalkDir(testDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err == nil && len(data) < 4<<20 {
			testSrc = append(testSrc, string(data))
		}
		return nil
	})
	for i := range out {
		tail := strings.TrimPrefix(out[i].Path, "/api/v1")
		for _, src := range testSrc {
			if (out[i].Handler != "" && strings.Contains(src, out[i].Handler)) ||
				strings.Contains(src, out[i].Path) || strings.Contains(src, tail) {
				out[i].Tested = true
				break
			}
		}
	}
	return out, nil
}

func prompts(src *docs.Source) []mcp.Prompt {
	return []mcp.Prompt{
		{
			Name:        "docs_answer",
			Description: "Answer a MeshChatX question using only the fetched docs, citing the page.",
			Arguments:   []mcp.PromptArg{{Name: "question", Description: "the question to answer", Required: true}},
			Handle: func(args map[string]string) (string, error) {
				return "Use the search_docs, list_sections, and get_section tools to find the answer in the MeshChatX documentation. Answer only from what the docs say, and cite the docs page URL for each claim. If the docs do not cover it, say so instead of guessing.\n\nQuestion: " + args["question"], nil
			},
		},
	}
}

func main() {
	root = findRoot()
	src := docs.NewSource("MeshChatX Docs", topics)
	src.MarkdownURL = func(t docs.Topic) string { return t.URL + "/export/md" }
	srv := mcp.NewServer("meshchatx-mcp", "0.3.0", append(tools(src), issueTools()...), prompts(src))
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "meshchatx-mcp:", err)
		os.Exit(1)
	}
}

// featureCheck verifies a frontend feature module is fully wired:
// dir + index.ts exist, register<Name>Feature is exported and called
// from registerAllFeatures.ts, declared routes have load() files that
// exist, and i18n keys used inside the feature resolve in en.json.
func featureCheck(root, id string) (string, error) {
	featDir := filepath.Join(root, "meshchatx/src/frontend/features", id)
	var b strings.Builder
	fmt.Fprintf(&b, "feature check: %s\n", id)
	fail := func(format string, v ...any) { fmt.Fprintf(&b, "FAIL  "+format+"\n", v...) }
	ok := func(format string, v ...any) { fmt.Fprintf(&b, "ok    "+format+"\n", v...) }

	idxPath := filepath.Join(featDir, "index.ts")
	idxData, err := os.ReadFile(idxPath) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		fail("no %s", filepath.Join("features", id, "index.ts"))
		return b.String(), nil
	}
	ok("index.ts exists")

	// find the exported register function name
	regRe := regexp.MustCompile(`export\s+function\s+(register\w*Feature)\s*\(`)
	regName := ""
	if m := regRe.FindSubmatch(idxData); m != nil {
		regName = string(m[1])
		ok("exports %s", regName)
	} else {
		fail("index.ts exports no register<Name>Feature()")
	}

	// declared routes and their load() files
	routeRe := regexp.MustCompile(`path:\s*["']([^"']+)["']`)
	loadRe := regexp.MustCompile(`import\(\s*["'](\./[^"']+)["']`)
	var routes, missing []string
	for _, m := range routeRe.FindAllSubmatch(idxData, -1) {
		routes = append(routes, string(m[1]))
	}
	for _, m := range loadRe.FindAllSubmatch(idxData, -1) {
		f := filepath.Join(featDir, string(m[1]))
		if _, err := os.Stat(f); err != nil { // #nosec G703 -- path constructed under jailed root
			missing = append(missing, string(m[1]))
		}
	}
	if len(routes) > 0 {
		ok("routes: %s", strings.Join(routes, ", "))
	} else {
		fail("index.ts declares no routes")
	}
	if len(missing) > 0 {
		fail("load() files missing: %s", strings.Join(missing, ", "))
	}

	// registered in registerAllFeatures.ts
	allData, err := os.ReadFile(filepath.Join(root, "meshchatx/src/frontend/features/registerAllFeatures.ts")) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		fail("registerAllFeatures.ts unreadable")
	} else {
		all := string(allData)
		if regName != "" && strings.Contains(all, regName) {
			ok("registered in registerAllFeatures.ts")
		} else if regName != "" {
			fail("registerAllFeatures.ts does not import/call %s", regName)
		}
	}

	// i18n keys used inside the feature resolve in en.json
	keyRe := regexp.MustCompile(`["'\x60]([a-zA-Z0-9_]+(\.[a-zA-Z0-9_]+)+)["'\x60]`)
	var used []string
	// tolerate walk errors; missing or unreadable feature dir is not fatal
	_ = filepath.WalkDir(featDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
		if err == nil && len(data) < 1<<20 {
			for _, m := range keyRe.FindAllSubmatch(data, -1) {
				used = append(used, string(m[1]))
			}
		}
		return nil
	})
	enPath := filepath.Join(root, "meshchatx/src/frontend/locales/en.json")
	if enData, err := os.ReadFile(enPath); err == nil { // #nosec G304 -- path jailed to repo root or local config
		var raw map[string]any
		if json.Unmarshal(enData, &raw) == nil {
			flat := map[string]bool{}
			var walk func(m map[string]any, pre string)
			walk = func(m map[string]any, pre string) {
				for k, v := range m {
					key := k
					if pre != "" {
						key = pre + "." + k
					}
					if t, ok := v.(map[string]any); ok {
						walk(t, key)
					} else {
						flat[key] = true
					}
				}
			}
			walk(raw, "")
			var undef []string
			for _, k := range used {
				if strings.HasPrefix(k, id+".") || strings.HasPrefix(k, "feature."+id) {
					if !flat[k] && !contains(undef, k) {
						undef = append(undef, k)
					}
				}
			}
			if len(undef) > 0 {
				fail("i18n keys missing in en.json: %s", strings.Join(undef[:min(8, len(undef))], ", "))
			} else {
				ok("i18n keys resolve")
			}
		}
	}

	// a test file for the feature
	testRe := regexp.MustCompile(`(?:^|[-_/])` + regexp.QuoteMeta(id) + `[-_.]`)
	var match string
	// tolerate walk errors; missing tests dir is reported as "no test file"
	_ = filepath.WalkDir(filepath.Join(root, "tests"), func(p string, d os.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() && match == "" && testRe.MatchString(filepath.Base(p)) {
			match = p
		}
		return nil
	})
	if match != "" {
		rel, _ := filepath.Rel(root, match)
		ok("test file: %s", filepath.ToSlash(rel))
	} else {
		fail("no test file naming %q under tests/", id)
	}
	return b.String(), nil
}

func contains(s []string, v string) bool {
	return slices.Contains(s, v)
}
