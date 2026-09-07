// SPDX-License-Identifier: 0BSD
// Package docs fetches, parses, caches, and searches documentation
// pages. Stdlib only, bounded memory.
package docs

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	maxBody      = 1 << 20 // 1 MiB per page
	cacheEntries = 32
	cacheTTL     = 10 * time.Minute
	fetchTimeout = 15 * time.Second
)

// Topic is a named documentation page.
type Topic struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Summary string `json:"summary"`
}

// Section is one heading-delimited part of a page.
type Section struct {
	Anchor string `json:"anchor"` // heading id or generated slug
	Name   string `json:"name"`
	start  int    // byte offset into page text
	end    int
}

type page struct {
	text     string
	sections []Section
}

// Source is a fixed set of topics fetched over HTTP with an LRU cache.
type Source struct {
	Name   string
	Topics []Topic

	// MarkdownURL, when set, returns a markdown export URL for a topic.
	// The source fetches it first and falls back to the HTML URL.
	MarkdownURL func(t Topic) string

	client  *http.Client
	fetchFn func(ctx context.Context, url string) (body, contentType string, err error)

	mu    sync.Mutex
	cache map[string]*page
	lru   []string // oldest first
	byID  map[string]Topic
}

func NewSource(name string, topics []Topic) *Source {
	s := &Source{
		Name:   name,
		Topics: topics,
		client: &http.Client{Timeout: fetchTimeout},
		cache:  make(map[string]*page),
		byID:   make(map[string]Topic, len(topics)),
	}
	for _, t := range topics {
		s.byID[t.ID] = t
	}
	return s
}

// List returns the topic index.
func (s *Source) List() []Topic { return s.Topics }

// TopicByID returns a topic or false.
func (s *Source) TopicByID(id string) (Topic, bool) {
	t, ok := s.byID[id]
	return t, ok
}

func (s *Source) load(ctx context.Context, id string) (*page, error) {
	t, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown topic %q (see list_topics)", id)
	}
	if p, ok := s.cached(t.URL); ok {
		return p, nil
	}
	fetch := s.fetchFn
	if fetch == nil {
		fetch = s.httpGet
	}
	if s.MarkdownURL != nil {
		mdURL := s.MarkdownURL(t)
		if raw, ct, err := fetch(ctx, mdURL); err == nil && strings.Contains(ct, "markdown") {
			p := parseMarkdown(raw)
			s.store(t.URL, p)
			return p, nil
		}
	}
	raw, ct, err := fetch(ctx, t.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", t.URL, err)
	}
	var p *page
	if strings.Contains(ct, "markdown") || strings.Contains(ct, "text/plain") {
		p = parseMarkdown(raw)
	} else {
		p = parseHTML(raw)
	}
	s.store(t.URL, p)
	return p, nil
}

// Get returns the full plain text of one topic page.
func (s *Source) Get(ctx context.Context, id string) (string, error) {
	p, err := s.load(ctx, id)
	if err != nil {
		return "", err
	}
	return p.text, nil
}

// Sections returns the heading index of a topic page.
func (s *Source) Sections(ctx context.Context, id string) ([]Section, error) {
	p, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	return p.sections, nil
}

// GetSection returns the text of one section by anchor or name
// (case-insensitive substring match).
func (s *Source) GetSection(ctx context.Context, id, section string) (string, error) {
	p, err := s.load(ctx, id)
	if err != nil {
		return "", err
	}
	sec := findSection(p, section)
	if sec == nil {
		var names []string
		for _, sc := range p.sections {
			names = append(names, sc.Anchor)
		}
		return "", fmt.Errorf("no section matching %q; available: %s", section, strings.Join(names, ", "))
	}
	return p.text[sec.start:sec.end], nil
}

// FetchURL fetches a page by URL, restricted to hosts that appear in
// the topic index. Returns parsed plain text.
func (s *Source) FetchURL(ctx context.Context, url string) (string, error) {
	allowed := false
	for _, t := range s.Topics {
		if hostOf(t.URL) == hostOf(url) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("url %q is not on a documented host", url)
	}
	if p, ok := s.cached(url); ok {
		return p.text, nil
	}
	fetch := s.fetchFn
	if fetch == nil {
		fetch = s.httpGet
	}
	raw, ct, err := fetch(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	var p *page
	if strings.Contains(ct, "markdown") || strings.Contains(ct, "text/plain") {
		p = parseMarkdown(raw)
	} else {
		p = parseHTML(raw)
	}
	s.store(url, p)
	return p.text, nil
}

func hostOf(url string) string {
	if i := strings.Index(url, "://"); i >= 0 {
		url = url[i+3:]
	}
	if i := strings.IndexByte(url, '/'); i >= 0 {
		url = url[:i]
	}
	return strings.ToLower(url)
}

func findSection(p *page, q string) *Section {
	q = strings.ToLower(strings.TrimSpace(q))
	for i := range p.sections {
		if p.sections[i].Anchor == q {
			return &p.sections[i]
		}
	}
	for i := range p.sections {
		if strings.Contains(strings.ToLower(p.sections[i].Name), q) {
			return &p.sections[i]
		}
	}
	return nil
}

// Search scores every topic against query words, searching section by
// section so results point at the right part of a page.
func (s *Source) Search(ctx context.Context, query string, limit int) (string, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return "", fmt.Errorf("empty query")
	}

	type hit struct {
		t     Topic
		sec   *Section
		score int
		snip  string
	}
	var hits []hit
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 4)

	for _, t := range s.Topics {
		meta := strings.ToLower(t.Title + " " + t.Summary + " " + t.ID)
		metaScore := 0
		for _, w := range words {
			metaScore += 3 * strings.Count(meta, w)
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(t Topic, metaScore int) {
			defer wg.Done()
			defer func() { <-sem }()
			p, err := s.load(ctx, t.ID)
			if err != nil {
				return
			}
			var local []hit
			if len(p.sections) == 0 {
				if sc, sn := scoreText(p.text, words); sc > 0 {
					local = append(local, hit{t, nil, sc, sn})
				}
			} else {
				for i := range p.sections {
					sec := &p.sections[i]
					body := p.text[sec.start:sec.end]
					sc, sn := scoreText(sec.Name+" "+body, words)
					if sc > 0 {
						local = append(local, hit{t, sec, sc, sn})
					}
				}
			}
			mu.Lock()
			for _, h := range local {
				h.score += metaScore
				hits = append(hits, h)
			}
			mu.Unlock()
		}(t, metaScore)
	}
	wg.Wait()

	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	var b strings.Builder
	shown := 0
	for _, h := range hits {
		if shown >= limit || h.score == 0 {
			break
		}
		if h.sec != nil {
			fmt.Fprintf(&b, "## %s / %s (%s#%s)\n%s\nget_section topic=%q section=%q\n\n",
				h.t.Title, h.sec.Name, h.t.URL, h.sec.Anchor, h.snip, h.t.ID, h.sec.Anchor)
		} else {
			fmt.Fprintf(&b, "## %s (%s)\n%s\nget_topic %q for full text\n\n",
				h.t.Title, h.t.URL, h.snip, h.t.ID)
		}
		shown++
	}
	if shown == 0 {
		return "no matches in " + s.Name, nil
	}
	return b.String(), nil
}

func scoreText(body string, words []string) (int, string) {
	low := strings.ToLower(body)
	score := 0
	first := -1
	for _, w := range words {
		score += strings.Count(low, w)
		if i := strings.Index(low, w); i >= 0 && (first < 0 || i < first) {
			first = i
		}
	}
	if score == 0 || first < 0 {
		return 0, ""
	}
	start := max(first-120, 0)
	end := min(first+280, len(body))
	return score, strings.TrimSpace(body[start:end])
}

func (s *Source) httpGet(ctx context.Context, url string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "docs-mcp/1.0 (+local)")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	return string(b), resp.Header.Get("Content-Type"), err
}

func (s *Source) cached(url string) (*page, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.cache[url]
	return p, ok
}

func (s *Source) store(url string, p *page) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cache[url]; !ok {
		s.lru = append(s.lru, url)
	}
	s.cache[url] = p
	for len(s.lru) > cacheEntries {
		delete(s.cache, s.lru[0])
		s.lru = s.lru[1:]
	}
}

var _ = cacheTTL // reserved for future expiry

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// parseMarkdown splits markdown into text plus a heading index.
func parseMarkdown(raw string) *page {
	// strip YAML front matter
	if strings.HasPrefix(raw, "---") {
		if end := strings.Index(raw[3:], "\n---"); end >= 0 {
			raw = raw[3+end+4:]
		}
	}
	var b strings.Builder
	var secs []Section
	for line := range strings.SplitSeq(raw, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			name := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(strings.TrimLeft(trim, "#")), "#"))
			secs = append(secs, Section{Anchor: slug(name), Name: name})
			b.WriteString("\n\x00" + name + "\n\n")
			continue
		}
		// strip front matter and link targets, keep link text
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return finishPage(b.String(), secs)
}

var blockTags = map[string]bool{
	"script": true, "style": true, "nav": true, "footer": true,
	"header": true, "noscript": true, "svg": true, "form": true,
	"select": true, "button": true, "aside": true,
}

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

var mainRe = regexp.MustCompile(`(?is)<(main|article|div)[^>]*(role="main"|class="[^"]*body[^"]*"|id="main"|id="content")[^>]*>`)

// parseHTML extracts the main content, strips markup, and indexes headings.
func parseHTML(raw string) *page {
	if m := mainRe.FindStringIndex(raw); m != nil {
		raw = raw[m[0]:]
	}
	var b strings.Builder
	b.Grow(len(raw) / 2)
	var secs []Section
	i := 0
	for i < len(raw) {
		c := raw[i]
		if c == '<' {
			j := strings.IndexByte(raw[i:], '>')
			if j < 0 {
				break
			}
			tag := raw[i+1 : i+j]
			i += j + 1
			name := tagName(tag)
			if blockTags[name] {
				if skip := skipTag(raw[i:], name); skip >= 0 {
					i += skip
				}
				continue
			}
			if headingTags[name] && tag[0] != '/' {
				head, adv := readUntil(raw[i:], "</"+name)
				head = strings.TrimSpace(strings.TrimRight(stripTags(head), "¶# \t"))
				if head != "" {
					anchor := attrValue(tag, "id")
					if anchor == "" {
						anchor = slug(head)
					}
					secs = append(secs, Section{Anchor: anchor, Name: head})
					b.WriteString("\n\x00" + head + "\n\n")
				}
				i += adv
				continue
			}
			switch name {
			case "p", "div", "br", "li", "ul", "ol", "tr", "section", "article", "table", "pre", "blockquote", "dt", "dd":
				b.WriteByte('\n')
			}
			continue
		}
		if c == '&' {
			if j := strings.IndexByte(raw[i:], ';'); j > 0 && j < 12 {
				b.WriteString(html.UnescapeString(raw[i : i+j+1]))
				i += j + 1
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return finishPage(b.String(), secs)
}

// finishPage collapses whitespace, resolves heading sentinel positions
// into section offsets, and strips the sentinels.
func finishPage(raw string, secs []Section) *page {
	text := collapseWS(raw)
	var b strings.Builder
	b.Grow(len(text))
	si := 0
	for i := 0; i < len(text); i++ {
		if text[i] == 0 {
			if si < len(secs) {
				if si > 0 {
					secs[si-1].end = b.Len()
				}
				secs[si].start = b.Len()
				si++
			}
			continue
		}
		b.WriteByte(text[i])
	}
	if si > 0 {
		secs[si-1].end = b.Len()
	}
	return &page{text: b.String(), sections: secs}
}

// readUntil returns text before the close tag and the offset past it.
func readUntil(rest, close string) (string, int) {
	idx := strings.Index(strings.ToLower(rest), close)
	if idx < 0 {
		return rest, len(rest)
	}
	end := strings.IndexByte(rest[idx:], '>')
	if end < 0 {
		return rest[:idx], idx + len(close)
	}
	return rest[:idx], idx + end + 1
}

func stripTags(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '<' {
			if j := strings.IndexByte(s[i:], '>'); j >= 0 {
				i += j + 1
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func attrValue(tag, attr string) string {
	key := attr + `="`
	_, after, ok := strings.Cut(tag, key)
	if !ok {
		return ""
	}
	rest := after
	before0, _, ok0 := strings.Cut(rest, "\"")
	if !ok0 {
		return ""
	}
	return before0
}

func tagName(tag string) string {
	tag = strings.TrimLeft(tag, "/ ")
	end := len(tag)
	for i, r := range tag {
		if unicode.IsSpace(r) || r == '/' {
			end = i
			break
		}
	}
	return strings.ToLower(tag[:end])
}

// skipTag returns the offset past the matching close tag, or -1.
func skipTag(rest, name string) int {
	close := "</" + name
	low := strings.ToLower(rest)
	idx := strings.Index(low, close)
	if idx < 0 {
		return -1
	}
	end := strings.IndexByte(low[idx:], '>')
	if end < 0 {
		return -1
	}
	return idx + end + 1
}

func collapseWS(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	newlines := 0
	for _, r := range s {
		switch {
		case r == '\n':
			if newlines < 2 {
				b.WriteByte('\n')
			}
			newlines++
			space = false
		case unicode.IsSpace(r):
			space = true
		default:
			if space && newlines == 0 {
				b.WriteByte(' ')
			}
			space = false
			newlines = 0
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
