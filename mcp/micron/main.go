// SPDX-License-Identifier: 0BSD
// Command micron is a stdio MCP server for Micron markup:
// parse, lint, render to HTML/ANSI, extract links/headings, search,
// generate templates, and reference the syntax. Uses vendored
// micron-parser-go for offline builds.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"micron-parser-go/micron"

	"github.com/Quad4-Software/ai/mcp/micron/internal/mcp"
	"github.com/Quad4-Software/ai/mcp/micron/internal/templates"
)

const (
	maxSourceBytes     = 1 << 20 // 1 MiB
	maxQueryBytes      = 4 << 10 // 4 KiB
	maxSearchLimit     = 1000
	defaultTemplate    = "page"
	defaultSearchLimit = 20
)

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intArg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func checkSource(s string) error {
	if s == "" {
		return fmt.Errorf("missing required argument: source")
	}
	if len(s) > maxSourceBytes {
		return fmt.Errorf("source capped at %d bytes; got %d", maxSourceBytes, len(s))
	}
	return nil
}

// checkQuery bounds and sanitizes the search needle. Control bytes
// would only smear the snippet output, so reject NUL outright and cap
// the length; the rest is plain substring matching, which is safe.
func checkQuery(q string) error {
	if q == "" {
		return fmt.Errorf("missing required argument: query")
	}
	if len(q) > maxQueryBytes {
		return fmt.Errorf("query capped at %d bytes; got %d", maxQueryBytes, len(q))
	}
	if strings.IndexByte(q, 0) >= 0 {
		return fmt.Errorf("query must not contain NUL bytes")
	}
	return nil
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "parse_micron",
			Description: "Parse Micron markup into a structured JSON AST (blocks, inlines, colors, spans).",
			InputSchema: obj(map[string]any{"source": strArg("Micron source, max 1 MiB")}, "source"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Source == "" {
					return "", fmt.Errorf("missing required argument: source")
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				p := micron.Parser{}
				doc, _ := p.ParseWithDiagnostics(a.Source)
				b, err := doc.DocumentJSON()
				if err != nil {
					return "", err
				}
				return string(b), nil
			},
		},
		{
			Name:        "lint_micron",
			Description: "Check Micron markup for syntax problems and return diagnostics.",
			InputSchema: obj(map[string]any{"source": strArg("Micron source, max 1 MiB")}, "source"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Source == "" {
					return "", fmt.Errorf("missing required argument: source")
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				p := micron.Parser{}
				diags := p.Lint(a.Source)
				b, err := micron.DiagnosticsJSON(diags)
				if err != nil {
					return "", err
				}
				return string(b), nil
			},
		},
		{
			Name:        "render_html",
			Description: "Convert Micron markup to an HTML fragment.",
			InputSchema: obj(map[string]any{"source": strArg("Micron source, max 1 MiB")}, "source"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Source == "" {
					return "", fmt.Errorf("missing required argument: source")
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				p := micron.Parser{}
				return p.ConvertMicronToHTML(a.Source), nil
			},
		},
		{
			Name:        "render_ansi",
			Description: "Convert Micron markup to ANSI terminal output.",
			InputSchema: obj(map[string]any{"source": strArg("Micron source, max 1 MiB")}, "source"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Source == "" {
					return "", fmt.Errorf("missing required argument: source")
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				p := micron.Parser{}
				doc := p.Parse(a.Source)
				return p.RenderANSI(doc), nil
			},
		},
		{
			Name:        "extract_links",
			Description: "Extract all links from Micron markup (URL, label, source line, field names).",
			InputSchema: obj(map[string]any{"source": strArg("Micron source, max 1 MiB")}, "source"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Source == "" {
					return "", fmt.Errorf("missing required argument: source")
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				p := micron.Parser{}
				doc := p.Parse(a.Source)
				links := collectLinks(doc)
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "extract_headings",
			Description: "Extract section headings from Micron markup with depth and source line.",
			InputSchema: obj(map[string]any{"source": strArg("Micron source, max 1 MiB")}, "source"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Source == "" {
					return "", fmt.Errorf("missing required argument: source")
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				p := micron.Parser{}
				doc := p.Parse(a.Source)
				h := collectHeadings(doc)
				b, _ := json.MarshalIndent(h, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "search_micron",
			Description: "Search Micron source for a query and return matching line numbers and snippets.",
			InputSchema: obj(map[string]any{
				"source": strArg("Micron source, max 1 MiB"),
				"query":  strArg("text to find"),
				"limit":  intArg("max results, default 20"),
			}, "source", "query"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Source string `json:"source"`
					Query  string `json:"query"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil {
					return "", fmt.Errorf("invalid arguments: %v", err)
				}
				if err := checkSource(a.Source); err != nil {
					return "", err
				}
				if err := checkQuery(a.Query); err != nil {
					return "", err
				}
				if a.Limit <= 0 {
					a.Limit = defaultSearchLimit
				}
				if a.Limit > maxSearchLimit {
					a.Limit = maxSearchLimit
				}
				return searchMicron(a.Source, a.Query, a.Limit), nil
			},
		},
		{
			Name:          "generate_template",
			Description:   "Generate a starter Micron .mu template: page, color, form, table, or minimal.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"template": "page"}`)}},
			InputSchema: obj(map[string]any{
				"template": strArg("page, color, form, table, or minimal"),
			}, "template"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Template string `json:"template"`
				}
				// #nosec G104 -- empty/default handled downstream
				json.Unmarshal(args, &a)
				if a.Template == "" {
					a.Template = defaultTemplate
				}
				body, ok := templates.ByName[a.Template]
				if !ok {
					return "", fmt.Errorf("unknown template %q (try page, color, form, table, minimal)", a.Template)
				}
				return body, nil
			},
		},
		{
			Name:        "micron_reference",
			Description: "Return a Micron syntax quick reference.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return templates.Reference, nil
			},
		},
	}
}

type linkOut struct {
	URL    string   `json:"url"`
	Label  string   `json:"label"`
	Line   int      `json:"line"`
	Fields []string `json:"fields,omitempty"`
}

type headingOut struct {
	Text  string `json:"text"`
	Depth int    `json:"depth"`
	Line  int    `json:"line"`
}

func collectLinks(doc *micron.Document) []linkOut {
	out := make([]linkOut, 0, 8)
	for _, b := range doc.Blocks {
		for _, in := range b.Inlines {
			if in.Link != nil {
				out = append(out, linkOut{
					URL:    in.Link.URL,
					Label:  in.Link.Label,
					Line:   b.SourceLine,
					Fields: in.Link.Fields,
				})
			}
		}
	}
	return out
}

func collectHeadings(doc *micron.Document) []headingOut {
	out := make([]headingOut, 0, 8)
	for _, b := range doc.Blocks {
		if b.Kind != micron.BlockHeading {
			continue
		}
		var text strings.Builder
		for _, in := range b.Inlines {
			if in.Text != "" {
				text.WriteString(in.Text)
			}
		}
		out = append(out, headingOut{
			Text:  text.String(),
			Depth: b.Depth,
			Line:  b.SourceLine,
		})
	}
	return out
}

func searchMicron(source, query string, limit int) string {
	out := strings.Builder{}
	out.Grow(len(source))
	lines := strings.Split(source, "\n")
	q := strings.ToLower(query)
	found := 0
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), q) {
			fmt.Fprintf(&out, "%4d: %s\n", i+1, line)
			found++
			if found >= limit {
				more := 0
				for _, rest := range lines[i+1:] {
					if strings.Contains(strings.ToLower(rest), q) {
						more++
					}
				}
				if more > 0 {
					fmt.Fprintf(&out, "... %d more\n", more)
				}
				break
			}
		}
	}
	if found == 0 {
		return "no matches"
	}
	return out.String()
}

func main() {
	srv := mcp.NewServer("micron", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "micron:", err)
		os.Exit(1)
	}
}
