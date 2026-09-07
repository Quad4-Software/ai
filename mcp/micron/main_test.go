// SPDX-License-Identifier: 0BSD
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"micron-parser-go/micron"

	"github.com/Quad4-Software/ai/mcp/micron/internal/mcp"
	"github.com/Quad4-Software/ai/mcp/micron/internal/templates"
)

// callTool invokes a named tool's handler and fails the test when the
// tool is unknown.
func callTool(t *testing.T, name, args string) (string, error) {
	t.Helper()
	for _, tool := range tools() {
		if tool.Name == name {
			return tool.Handle(context.Background(), json.RawMessage(args))
		}
	}
	t.Fatalf("unknown tool %q", name)
	return "", nil
}

func mustCall(t *testing.T, name, args string) string {
	t.Helper()
	out, err := callTool(t, name, args)
	if err != nil {
		t.Fatalf("%s(%s): %v", name, args, err)
	}
	return out
}

const smokeSource = `>Welcome

Hello ` + "`[link`/page.mu]" + ` world ` + "`!bold`!" + `
>>Sub section
`

// --- unit tests: every tool -------------------------------------------------

func TestParseMicronTool(t *testing.T) {
	out := mustCall(t, "parse_micron", `{"source":">Title\n\nbody text\n"}`)
	var doc struct {
		Blocks []struct {
			Kind    int `json:"kind"`
			Depth   int `json:"depth"`
			Inlines []struct {
				Text string `json:"text"`
			} `json:"inlines"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("parse output is not JSON: %v", err)
	}
	if len(doc.Blocks) == 0 {
		t.Fatal("expected blocks")
	}
	if doc.Blocks[0].Kind != int(micron.BlockHeading) || doc.Blocks[0].Depth != 1 {
		t.Fatalf("first block should be a level-1 heading: %+v", doc.Blocks[0])
	}
}

func TestLintMicronTool(t *testing.T) {
	out := mustCall(t, "lint_micron", `{"source":"#!fg=zz\n>x\n"}`)
	var diags []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(out), &diags); err != nil {
		t.Fatalf("lint output is not JSON: %v", err)
	}
	if len(diags) == 0 || diags[0].Code != "header.fg_invalid" {
		t.Fatalf("expected header.fg_invalid diagnostic: %s", out)
	}
	// clean source yields an empty diagnostic list
	out = mustCall(t, "lint_micron", `{"source":"plain text\n"}`)
	if strings.TrimSpace(out) != "[]" && strings.TrimSpace(out) != "null" {
		t.Fatalf("clean lint should be empty: %s", out)
	}
}

func TestRenderHTMLTool(t *testing.T) {
	out := mustCall(t, "render_html", `{"source":"`+"`!hi`!"+`"}`)
	if !strings.Contains(out, "font-weight:bold") || !strings.Contains(out, "hi") {
		t.Fatalf("bold markup should render: %s", out)
	}
	if !strings.HasPrefix(out, `<div class="Mu-root"`) {
		t.Fatalf("output should be an HTML fragment: %s", out)
	}
}

func TestRenderANSITool(t *testing.T) {
	out := mustCall(t, "render_ansi", `{"source":"hello\n"}`)
	if !strings.Contains(out, "hello") {
		t.Fatalf("ANSI output should contain text: %q", out)
	}
	out = mustCall(t, "render_ansi", `{"source":"`+"`!hi`!"+`"}`)
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("bold should emit ANSI escapes: %q", out)
	}
}

func TestExtractLinksTool(t *testing.T) {
	out := mustCall(t, "extract_links", `{"source":"`+"`[home`/index.mu] and `[docs`https://x.mu]"+`"}`)
	var links []linkOut
	if err := json.Unmarshal([]byte(out), &links); err != nil {
		t.Fatalf("links output is not JSON: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links: %s", out)
	}
	if links[0].Label != "home" || !strings.Contains(links[0].URL, "/index.mu") {
		t.Fatalf("bad first link: %+v", links[0])
	}
	if links[0].Line != 1 {
		t.Fatalf("line should be 1: %+v", links[0])
	}
}

func TestExtractHeadingsTool(t *testing.T) {
	out := mustCall(t, "extract_headings", `{"source":">Top\n\ntext\n>>Sub\n>>>Deep\n"}`)
	var heads []headingOut
	if err := json.Unmarshal([]byte(out), &heads); err != nil {
		t.Fatalf("headings output is not JSON: %v", err)
	}
	if len(heads) != 3 || heads[0].Text != "Top" || heads[1].Depth != 2 || heads[2].Depth != 3 {
		t.Fatalf("bad headings: %s", out)
	}
	if heads[2].Line != 5 {
		t.Fatalf("deep heading should be on line 5: %+v", heads[2])
	}
}

func TestSearchMicronTool(t *testing.T) {
	src := "alpha\nbeta\ngamma\nbeta again\nlast beta\n"
	a, _ := json.Marshal(map[string]any{"source": src, "query": "beta", "limit": 2})
	out := mustCall(t, "search_micron", string(a))
	if !strings.Contains(out, "beta") || !strings.Contains(out, "... 1 more") {
		t.Fatalf("search should cap and count the rest: %q", out)
	}
	a, _ = json.Marshal(map[string]any{"source": src, "query": "BETA"})
	out = mustCall(t, "search_micron", string(a))
	if !strings.Contains(out, "beta") {
		t.Fatalf("search should be case-insensitive: %q", out)
	}
	a, _ = json.Marshal(map[string]any{"source": src, "query": "zzz"})
	if out2 := mustCall(t, "search_micron", string(a)); out2 != "no matches" {
		t.Fatalf("miss should report no matches: %q", out2)
	}
	// huge limit clamps instead of erroring
	a, _ = json.Marshal(map[string]any{"source": src, "query": "beta", "limit": 1 << 30})
	if out = mustCall(t, "search_micron", string(a)); !strings.Contains(out, "beta") {
		t.Fatalf("clamped limit should still search: %q", out)
	}
}

func TestGenerateTemplateTool(t *testing.T) {
	for name, want := range map[string]string{
		"page": templates.Page, "color": templates.Color, "form": templates.Form,
		"table": templates.Table, "minimal": templates.Minimal,
	} {
		a, _ := json.Marshal(map[string]any{"template": name})
		if out := mustCall(t, "generate_template", string(a)); out != want {
			t.Fatalf("template %q mismatch", name)
		}
	}
	if out := mustCall(t, "generate_template", `{}`); out != templates.Page {
		t.Fatal("empty template should default to page")
	}
	if _, err := callTool(t, "generate_template", `{"template":"bogus"}`); err == nil {
		t.Fatal("unknown template should error")
	}
	// every embedded template should itself parse cleanly
	p := micron.Parser{}
	for name, body := range templates.ByName {
		if diags := p.Lint(body); len(diags) > 0 {
			t.Fatalf("template %q has diagnostics: %v", name, diags)
		}
	}
}

func TestMicronReferenceTool(t *testing.T) {
	out := mustCall(t, "micron_reference", `null`)
	if out != templates.Reference || !strings.Contains(out, "Quick Reference") {
		t.Fatalf("bad reference output: %q", out[:80])
	}
}

// --- input validation --------------------------------------------------------

func TestMissingAndOversizedArgs(t *testing.T) {
	big := strings.Repeat("x", maxSourceBytes+1)
	for _, tool := range []string{"parse_micron", "lint_micron", "render_html", "render_ansi", "extract_links", "extract_headings"} {
		if _, err := callTool(t, tool, `{}`); err == nil {
			t.Fatalf("%s should reject missing source", tool)
		}
		a, _ := json.Marshal(map[string]any{"source": big})
		if _, err := callTool(t, tool, string(a)); err == nil {
			t.Fatalf("%s should reject >1 MiB source", tool)
		}
	}
	if _, err := callTool(t, "search_micron", `{"source":"x","query":""}`); err == nil {
		t.Fatal("empty query should error")
	}
	if _, err := callTool(t, "search_micron", `{"source":"x"}`); err == nil {
		t.Fatal("missing query should error")
	}
	bigQ := strings.Repeat("q", maxQueryBytes+1)
	a, _ := json.Marshal(map[string]any{"source": "x", "query": bigQ})
	if _, err := callTool(t, "search_micron", string(a)); err == nil {
		t.Fatal("oversized query should error")
	}
	a, _ = json.Marshal(map[string]any{"source": "x", "query": "a\x00b"})
	if _, err := callTool(t, "search_micron", string(a)); err == nil {
		t.Fatal("NUL in query should error")
	}
	if _, err := callTool(t, "parse_micron", `{"source":123}`); err == nil {
		t.Fatal("non-string source should error")
	}
	if _, err := callTool(t, "parse_micron", `not json`); err == nil {
		t.Fatal("malformed JSON args should error")
	}
}

// --- smoke test --------------------------------------------------------------

func TestSmokeEndToEnd(t *testing.T) {
	srv := mcp.NewServer("micron", "test", tools(), nil)
	var out strings.Builder
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"render_html","arguments":{"source":">Hi\\n"}}}` + "\n" +
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"extract_links","arguments":{"source":"` +
		"`[a`/x.mu]" + `"}}}` + "\n"
	if err := srv.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 responses, got %d: %s", len(lines), out.String())
	}
	var res []map[string]any
	for _, l := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("bad response line: %v", err)
		}
		res = append(res, m)
	}
	init := res[0]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-06-18" {
		t.Fatalf("version negotiation: %v", init)
	}
	tl := res[1]["result"].(map[string]any)["tools"].([]any)
	if len(tl) != len(tools()) {
		t.Fatalf("tools/list should expose %d tools", len(tools()))
	}
	html := res[2]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(html, "Hi") || !strings.Contains(html, "#777") {
		t.Fatalf("render_html over the wire: %s", html)
	}
	links := res[3]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(links, "/x.mu") {
		t.Fatalf("extract_links over the wire: %s", links)
	}
}

// --- edge cases --------------------------------------------------------------

func TestEdgeInputs(t *testing.T) {
	// empty source is rejected by validation but the parser itself copes
	if _, err := callTool(t, "parse_micron", `{"source":""}`); err == nil {
		t.Fatal("empty source should be a validation error")
	}
	p := micron.Parser{}
	if doc := p.Parse(""); doc == nil || len(doc.Blocks) == 0 {
		t.Fatal("parser should return a document for empty input")
	}
	// exactly at the cap still works
	at := strings.Repeat("a", maxSourceBytes)
	a, _ := json.Marshal(map[string]any{"source": at})
	if _, err := callTool(t, "parse_micron", string(a)); err != nil {
		t.Fatalf("1 MiB source should parse: %v", err)
	}
	// malformed markup and unmatched tags must not panic or error
	for _, s := range []string{
		"`!unclosed bold",
		"`[link with no url or bracket",
		"````` ``` ` ` `",
		">>>>>>deep\n>>>>>>>\n",
		"`t\n| unclosed table\n",
		"#!fg=\n#!bg=xyz\n#!unknown=1\n",
		"`<<unclosed field\n",
		"`[a`bad scheme" + "\x00" + "x]\n",
		"\xff\xfe binary \x00 soup \x7f\n",
		"line\r\nwindows\r\nline\r",
	} {
		a, _ := json.Marshal(map[string]any{"source": s})
		if _, err := callTool(t, "parse_micron", string(a)); err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		if out, err := callTool(t, "render_html", string(a)); err != nil {
			t.Fatalf("render %q: %v", s, err)
		} else if strings.Count(out, "<") != strings.Count(out, ">") {
			t.Fatalf("unbalanced angle brackets for %q: %s", s, out)
		}
		if _, err := callTool(t, "lint_micron", string(a)); err != nil {
			t.Fatalf("lint %q: %v", s, err)
		}
	}
}

// --- property tests -----------------------------------------------------------

func TestParseDocumentJSONRoundTrip(t *testing.T) {
	corpus := []string{
		smokeSource,
		templates.Page, templates.Form, templates.Table,
		"#!fg=abc\n#!bg=123\n>T\n",
		"`!b`! `*i`* `_u`_ `c`l`r`a\n",
	}
	p := micron.Parser{}
	for _, src := range corpus {
		doc, diags := p.ParseWithDiagnostics(src)
		b, err := doc.DocumentJSON()
		if err != nil {
			t.Fatalf("DocumentJSON: %v", err)
		}
		var decoded micron.Document
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatalf("DocumentJSON output does not decode: %v", err)
		}
		if len(decoded.Blocks) != len(doc.Blocks) {
			t.Fatalf("block count lost in round-trip: %d != %d", len(decoded.Blocks), len(doc.Blocks))
		}
		for i, bl := range doc.Blocks {
			got := decoded.Blocks[i]
			if got.Kind != bl.Kind || got.SourceLine != bl.SourceLine || got.Depth != bl.Depth || got.Span != bl.Span {
				t.Fatalf("block %d changed in round-trip: %+v != %+v", i, got, bl)
			}
			if len(got.Inlines) != len(bl.Inlines) {
				t.Fatalf("block %d inline count changed", i)
			}
		}
		if len(diags) > 0 && len(decoded.Diagnostics) != len(diags) {
			t.Fatal("diagnostics lost in round-trip")
		}
	}
}

func TestConvertEqualsParseThenRender(t *testing.T) {
	corpus := []string{
		smokeSource, templates.Page, templates.Color, templates.Minimal,
		"plain\n---\n-=\n-*\n",
		">H\n>>H2\n`!bold`! `[l`/u.mu]\n",
		"#!fg=abc\n#!bg=123\n>x\n",
		"\x00\xff binary <<< >>>\n",
	}
	p := micron.Parser{}
	for _, src := range corpus {
		a := p.ConvertMicronToHTML(src)
		b := p.RenderHTML(p.Parse(src))
		if a != b {
			t.Fatalf("ConvertMicronToHTML and Parse+RenderHTML diverged for %q\nA=%s\nB=%s", src, a, b)
		}
	}
}

func TestLintMatchesParseDiagnostics(t *testing.T) {
	corpus := []string{
		"#!fg=zz\n#!bg=1\n>x\n",
		"`!unclosed\n`t\n| t |\n",
		"clean\n",
		"#!bg=zzzzzzzz\n",
		strings.Repeat("`", 50),
		"`[x`]\n",
		"#!fg=abcdef\n#!fg=abc\n#!fg=\n",
	}
	p := micron.Parser{}
	for _, src := range corpus {
		lint := p.Lint(src)
		_, diags := p.ParseWithDiagnostics(src)
		if len(lint) != len(diags) {
			t.Fatalf("Lint vs ParseWithDiagnostics count differs for %q: %d vs %d", src, len(lint), len(diags))
		}
		for i := range lint {
			if lint[i] != diags[i] {
				t.Fatalf("diagnostic %d differs for %q: %+v vs %+v", i, src, lint[i], diags[i])
			}
		}
	}
}

// --- race: concurrent Serve calls on one Server -------------------------------

func TestConcurrentServe(t *testing.T) {
	srv := mcp.NewServer("micron", "test", tools(), nil)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			var in strings.Builder
			for i := range 20 {
				fmt.Fprintf(&in, `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"render_html","arguments":{"source":"x %d"}}}`+"\n", i, g)
				in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"tasks/list","params":{}}` + "\n")
			}
			var out strings.Builder
			if err := srv.Serve(context.Background(), strings.NewReader(in.String()), &out); err != nil {
				t.Errorf("serve: %v", err)
			}
			if got := strings.Count(out.String(), "\n"); got != 40 {
				t.Errorf("goroutine %d: expected 40 responses, got %d", g, got)
			}
		}(g)
	}
	wg.Wait()
}

func TestConcurrentToolCallsSameStream(t *testing.T) {
	// hammer the shared helpers through tools/call on one connection
	srv := mcp.NewServer("micron", "test", tools(), nil)
	var in strings.Builder
	for i := range 50 {
		fmt.Fprintf(&in, `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"parse_micron","arguments":{"source":">h%d"}}}`+"\n", i, i)
	}
	var out strings.Builder
	if err := srv.Serve(context.Background(), strings.NewReader(in.String()), &out); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out.String(), "\n"); got != 50 {
		t.Fatalf("expected 50 responses, got %d", got)
	}
}

// --- fuzz ---------------------------------------------------------------------

var fuzzSeeds = []string{
	">Heading\n\ntext `!bold`! `[l`/x.mu]\n",
	"#!fg=abc\n#!bg=def\n>x\n",
	"`t\n| a | b |\n`t\n",
	"`<<name`>> `[`?f=`v\n",
	"`!`*`_``c`l`r`a`Faaa`f`B222`b\n",
	">>>>>>deep\n",
	"\x00\xff\xfe\x7f binary\n",
	"`[unclosed `!unclosed `t\n| table\n",
	"a\rb\rc\n",
	strings.Repeat("x", 4096) + "\n",
	"⧖partial `[]x]\n",
	"`[`?name=`value `[go`/y.mu|`?a|`?b]\n",
}

func FuzzParseMicron(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > maxSourceBytes {
			t.Skip("beyond tool cap")
		}
		p := micron.Parser{}
		doc, diags := p.ParseWithDiagnostics(src)
		if doc == nil {
			t.Fatal("nil document")
		}
		// every span must stay inside the source
		for _, b := range doc.Blocks {
			if b.Span.Start < 0 || b.Span.End > len(src) || b.Span.Start > b.Span.End {
				t.Fatalf("block span %v outside %d-byte source", b.Span, len(src))
			}
			for _, in := range b.Inlines {
				if in.Span.Start < 0 || in.Span.End > len(src) {
					t.Fatalf("inline span %v outside source", in.Span)
				}
			}
		}
		for _, d := range diags {
			if d.Span.Start < 0 || d.Span.End > len(src) {
				t.Fatalf("diagnostic span %v outside source", d.Span)
			}
		}
		b, err := doc.DocumentJSON()
		if err != nil {
			t.Fatalf("DocumentJSON: %v", err)
		}
		var decoded micron.Document
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatalf("DocumentJSON does not decode: %v", err)
		}
		if len(decoded.Blocks) != len(doc.Blocks) {
			t.Fatal("block count changed through JSON round-trip")
		}
		// Lint must agree with ParseWithDiagnostics
		if got := len(p.Lint(src)); got != len(diags) {
			t.Fatalf("Lint returned %d diags, ParseWithDiagnostics %d", got, len(diags))
		}
	})
}

// forbiddenHTML lists fragments that must never appear in rendered output.
// Scheme checks are anchored to attribute position (=\"...) because the
// nomadnetwork:// safety prefix legitimately embeds e.g. "javascript:"
// inside a prefixed href.
var forbiddenHTML = []string{
	"<script", "</script", "<iframe", "<object", "<embed", "<svg",
	"<base", "<form", "<meta", "onerror=", "onload=", "onclick=",
	`="javascript:`, `="vbscript:`, `="file:`, `="data:`,
}

func FuzzRenderHTML(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s)
	}
	f.Add("\x00\x01\x02<script>alert(1)</script>`[x`javascript:alert(1)]")
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > maxSourceBytes {
			t.Skip("beyond tool cap")
		}
		p := micron.Parser{}
		out := p.ConvertMicronToHTML(src)
		lower := strings.ToLower(out)
		for _, bad := range forbiddenHTML {
			if strings.Contains(lower, bad) {
				t.Fatalf("forbidden %q in output for source %q: %s", bad, src, out)
			}
		}
		if strings.Count(out, "<") != strings.Count(out, ">") {
			t.Fatalf("unbalanced angle brackets for %q: %s", src, out)
		}
		// both render paths must agree
		if other := p.RenderHTML(p.Parse(src)); other != out {
			t.Fatalf("render paths diverged for %q", src)
		}
		// ANSI path must not panic either
		_ = p.RenderANSI(p.Parse(src))
	})
}

// --- benchmarks -----------------------------------------------------------------

var benchSource = strings.Repeat(
	">Section\n\nParagraph with `!bold`! and `[a link`/some/where.mu] plus `Fabc`Fcolor`f.\n>>Sub\n---\n",
	200)

func BenchmarkParse(b *testing.B) {
	p := micron.Parser{}
	for b.Loop() {
		p.Parse(benchSource)
	}
}

func BenchmarkRenderHTML(b *testing.B) {
	p := micron.Parser{}
	for b.Loop() {
		p.ConvertMicronToHTML(benchSource)
	}
}

func BenchmarkSearch(b *testing.B) {
	for b.Loop() {
		searchMicron(benchSource, "link", defaultSearchLimit)
	}
}
