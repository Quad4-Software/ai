// SPDX-License-Identifier: 0BSD
package mediawiki

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// wikiFixture emulates api.php with canned action responses.
func wikiFixture(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch {
		case q.Get("action") == "query" && q.Get("meta") == "siteinfo":
			fmt.Fprint(w, `{"query":{"general":{"sitename":"Fixture Wiki","generator":"MediaWiki 1.43.0","base":"https://x/wiki/Main_Page"}}}`)
		case q.Get("list") == "search":
			if q.Get("srsearch") == "fail" {
				fmt.Fprint(w, `{"error":{"code":"srsearch","info":"broken"}}`)
				return
			}
			fmt.Fprint(w, `{"query":{"searchinfo":{"totalhits":2},"search":[`+
				`{"title":"Foo","snippet":"<b>Foo</b> bar &amp; baz","size":10,"wordcount":2,"timestamp":"t"},`+
				`{"title":"Bar","snippet":"plain","size":5,"wordcount":1,"timestamp":"t"}]}}`)
		case q.Get("action") == "parse":
			if q.Get("prop") == "wikitext|displaytitle|revid|categories" {
				fmt.Fprint(w, `{"parse":{"title":"Foo","revid":42,"wikitext":"== Head =="}}`)
				return
			}
			fmt.Fprint(w, `{"parse":{"title":"Foo","revid":42,`+
				`"text":"<p>Hello <b>world</b></p><script>evil()</script><p>Second &amp; more</p>",`+
				`"categories":[{"sortkey":"","category":"Things"}]}}`)
		case q.Get("list") == "backlinks":
			fmt.Fprint(w, `{"query":{"backlinks":[{"pageid":1,"title":"A"},{"pageid":2,"title":"B"}]}}`)
		case q.Get("list") == "categorymembers":
			if q.Get("cmtitle") != "Category:Vehicles" {
				fmt.Fprint(w, `{"error":{"code":"invalidcategory","info":"bad cat"}}`)
				return
			}
			fmt.Fprint(w, `{"query":{"categorymembers":[{"title":"Car","type":"page"}]}}`)
		case q.Get("prop") == "info|categories":
			fmt.Fprint(w, `{"query":{"pages":[{"pageid":7,"title":"Foo","length":100,`+
				`"touched":"2026-01-01T00:00:00Z","fullurl":"https://x/wiki/Foo",`+
				`"categories":[{"ns":14,"title":"Category:Things"}]}]}}`)
		default:
			fmt.Fprint(w, `{"error":{"code":"unknown_action","info":"unhandled"}}`)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestSearch(t *testing.T) {
	c, _ := wikiFixture(t)
	hits, info, res, err := c.Search(context.Background(), "foo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if res.Via != "direct" || len(hits) != 2 {
		t.Fatalf("hits=%v via=%s", hits, res.Via)
	}
	if hits[0]["snippet"] != "Foo bar & baz" {
		t.Fatalf("snippet not stripped: %q", hits[0]["snippet"])
	}
	si, _ := info.(map[string]any)
	if si["totalhits"] != float64(2) {
		t.Fatalf("searchinfo %v", si)
	}
}

func TestSearchAPIError(t *testing.T) {
	c, _ := wikiFixture(t)
	_, _, _, err := c.Search(context.Background(), "fail", 5)
	if err == nil || !strings.Contains(err.Error(), "srsearch: broken") {
		t.Fatalf("expected api error, got %v", err)
	}
}

func TestPageTextAndWikitext(t *testing.T) {
	c, _ := wikiFixture(t)
	out, _, err := c.Page(context.Background(), "Foo", "text", 0)
	if err != nil {
		t.Fatal(err)
	}
	content := out["content"].(string)
	if strings.Contains(content, "evil()") || strings.Contains(content, "<") {
		t.Fatalf("html/script leaked: %q", content)
	}
	if !strings.Contains(content, "Hello world") || !strings.Contains(content, "Second & more") {
		t.Fatalf("unexpected content %q", content)
	}
	wt, _, err := c.Page(context.Background(), "Foo", "wikitext", 0)
	if err != nil {
		t.Fatal(err)
	}
	if wt["content"] != "== Head ==" {
		t.Fatalf("wikitext = %v", wt["content"])
	}
	if _, _, err := c.Page(context.Background(), "Foo", "bogus", 0); err == nil {
		t.Fatal("bad format should error")
	}
}

func TestInfo(t *testing.T) {
	c, _ := wikiFixture(t)
	out, _, err := c.Info(context.Background(), "Foo")
	if err != nil {
		t.Fatal(err)
	}
	if out["pageid"] != float64(7) {
		t.Fatalf("info %v", out)
	}
	cats, _ := out["categories"].([]string)
	if len(cats) != 1 || cats[0] != "Category:Things" {
		t.Fatalf("categories %v", out["categories"])
	}
}

func TestBacklinksAndCategory(t *testing.T) {
	c, _ := wikiFixture(t)
	bl, _, err := c.Backlinks(context.Background(), "Foo", 10)
	if err != nil || len(bl) != 2 {
		t.Fatalf("backlinks %v %v", bl, err)
	}
	cm, _, err := c.CategoryMembers(context.Background(), "Vehicles", "page", 10)
	if err != nil || len(cm) != 1 || cm[0]["title"] != "Car" {
		t.Fatalf("category %v %v", cm, err)
	}
	if _, _, err := c.CategoryMembers(context.Background(), "Vehicles", "bogus", 10); err == nil {
		t.Fatal("bad type should error")
	}
}

func TestNoBaseURL(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil ||
		!strings.Contains(err.Error(), "MEDIAWIKI_BASE_URL") {
		t.Fatalf("expected env error, got %v", err)
	}
}

func TestAPIURLOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/w/api.php" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"query":{"general":{"sitename":"W"}}}`)
	}))
	defer srv.Close()
	c, err := NewClient(Config{BaseURL: srv.URL, APIURL: srv.URL + "/w/api.php"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SiteInfo(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestStripHTML(t *testing.T) {
	in := `<p>A <a href="/wiki/B">link</a></p><style>x{}</style><ul><li>one</li><li>two</li></ul>`
	out := stripHTML(in)
	if strings.Contains(out, "x{}") || strings.Contains(out, "<") {
		t.Fatalf("leak: %q", out)
	}
	if !strings.Contains(out, "A link") || !strings.Contains(out, "one") {
		t.Fatalf("out %q", out)
	}
	_ = url.Values{} // keep import
}
