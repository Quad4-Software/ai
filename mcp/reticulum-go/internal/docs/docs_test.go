// SPDX-License-Identifier: 0BSD
package docs

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
)

const testPage = `<html><head><style>.x{color:red}</style><script>evil()</script></head>
<body><nav>menu junk</nav><main id="content"><h1 id="dest">Destinations</h1>
<p>A destination is an <b>endpoint</b> in Reticulum.</p>
<h2 id="ann">Announces</h2>
<p>Announces propagate through transport nodes.</p></main></body></html>`

func newTestSource(calls *atomic.Int32) *Source {
	s := NewSource("Test", []Topic{
		{ID: "a", Title: "Alpha", URL: "http://x/a", Summary: "destinations"},
		{ID: "b", Title: "Beta", URL: "http://x/b", Summary: "interfaces"},
	})
	s.fetchFn = func(_ context.Context, url string) (string, string, error) {
		calls.Add(1)
		if strings.HasSuffix(url, "/b") {
			return `# Beta

## UDP

udp interface on port 4242`, "text/markdown", nil
		}
		return testPage, "text/html", nil
	}
	return s
}

func TestGetAndCache(t *testing.T) {
	var calls atomic.Int32
	s := newTestSource(&calls)
	ctx := context.Background()

	got, err := s.Get(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "destination is an endpoint") {
		t.Fatalf("parse dropped content: %q", got)
	}
	if strings.Contains(got, "evil()") || strings.Contains(got, "menu junk") {
		t.Fatalf("script/nav not stripped: %q", got)
	}
	if _, err := s.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 fetch (cache hit), got %d", calls.Load())
	}
	if _, err := s.Get(ctx, "missing"); err == nil {
		t.Fatal("unknown id should error")
	}
}

func TestSections(t *testing.T) {
	var calls atomic.Int32
	s := newTestSource(&calls)
	ctx := context.Background()

	secs, err := s.Sections(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 2 || secs[0].Anchor != "dest" || secs[1].Anchor != "ann" {
		t.Fatalf("bad sections: %+v", secs)
	}
	sec, err := s.GetSection(ctx, "a", "ann")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sec, "propagate") || strings.Contains(sec, "endpoint") {
		t.Fatalf("section slice wrong: %q", sec)
	}
	// markdown topic sections
	secs, err = s.Sections(ctx, "b")
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 2 || secs[1].Anchor != "udp" {
		t.Fatalf("bad md sections: %+v", secs)
	}
	if _, err := s.GetSection(ctx, "a", "nope"); err == nil {
		t.Fatal("missing section should error")
	}
}

func TestSearch(t *testing.T) {
	var calls atomic.Int32
	s := newTestSource(&calls)
	out, err := s.Search(context.Background(), "transport", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "#ann") {
		t.Fatalf("search missed section anchor: %q", out)
	}
	if !strings.Contains(out, `get_section topic="a" section="ann"`) {
		t.Fatalf("missing get_section hint: %q", out)
	}
	out, err = s.Search(context.Background(), "zzzznotfound", 5)
	if err != nil || !strings.Contains(out, "no matches") {
		t.Fatalf("want no matches, got %q err %v", out, err)
	}
}

func TestFetchError(t *testing.T) {
	var calls atomic.Int32
	s := newTestSource(&calls)
	s.fetchFn = func(_ context.Context, _ string) (string, string, error) {
		return "", "", fmt.Errorf("network down")
	}
	if _, err := s.Get(context.Background(), "a"); err == nil {
		t.Fatal("fetch error should propagate")
	}
}
