// SPDX-License-Identifier: 0BSD
// Package community fetches community sources: rns.recipes forum,
// unsigned.io posts, and markqvist/Reticulum GitHub discussions.
// Read-only HTTP with a small in-memory cache.
package community

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	nurl "net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	maxBody  = 1 << 20
	timeout  = 15 * time.Second
	cacheTTL = 10 * time.Minute
)

var client = &http.Client{Timeout: timeout}

type cacheEnt struct {
	body string
	at   time.Time
}

var (
	cacheMu sync.Mutex
	cache   = map[string]cacheEnt{}
)

func get(ctx context.Context, url string) (string, error) {
	cacheMu.Lock()
	if e, ok := cache[url]; ok && time.Since(e.at) < cacheTTL {
		cacheMu.Unlock()
		return e.body, nil
	}
	cacheMu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mcp/rns/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %s for %s", resp.Status, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", err
	}
	body := string(b)
	cacheMu.Lock()
	cache[url] = cacheEnt{body, time.Now()}
	cacheMu.Unlock()
	return body, nil
}

var (
	anchorRe = regexp.MustCompile(`(?s)<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	tagRe    = regexp.MustCompile(`<[^>]+>`)
	wsRe     = regexp.MustCompile(`\s+`)
	entities = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&nbsp;", " ")
)

func clean(s string) string {
	return wsRe.ReplaceAllString(strings.TrimSpace(entities.Replace(tagRe.ReplaceAllString(s, " "))), " ")
}

// cleanMD formats a markdown export while preserving paragraph breaks.
func cleanMD(s string) string {
	var b strings.Builder
	for raw := range strings.SplitSeq(s, "\n") {
		line := strings.TrimSpace(entities.Replace(tagRe.ReplaceAllString(raw, " ")))
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

// relToAbs expands rns.recipes relative image and link targets to absolute URLs.
func relToAbs(s string) string {
	return strings.ReplaceAll(s, "](/", "](https://rns.recipes/")
}

// Link is a parsed anchor.
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// UnsignedPosts lists articles from unsigned.io index.
func UnsignedPosts(ctx context.Context) ([]Link, error) {
	body, err := get(ctx, "https://unsigned.io/")
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, m := range anchorRe.FindAllStringSubmatch(body, -1) {
		href, title := m[1], clean(m[2])
		if strings.Contains(href, "articles/") && title != "" {
			if strings.HasPrefix(href, "./") {
				href = "https://unsigned.io/" + href[2:]
			} else if strings.HasPrefix(href, "/") {
				href = "https://unsigned.io" + href
			}
			out = append(out, Link{title, href})
		}
	}
	return out, nil
}

// ForumCategories lists rns.recipes forum categories.
func ForumCategories(ctx context.Context) ([]Link, error) {
	body, err := get(ctx, "https://rns.recipes/forum")
	if err != nil {
		return nil, err
	}
	var out []Link
	seen := map[string]bool{}
	for _, m := range anchorRe.FindAllStringSubmatch(body, -1) {
		href, title := m[1], clean(m[2])
		href = strings.TrimPrefix(href, "https://rns.recipes")
		if strings.HasPrefix(href, "/forum/") && !strings.Contains(strings.TrimPrefix(href, "/forum/"), "/") {
			if title != "" && !seen[href] {
				seen[href] = true
				out = append(out, Link{title, "https://rns.recipes" + href})
			}
		}
	}
	return out, nil
}

// ForumThreads lists thread links under a category path like
// /forum/general or a bare name.
func ForumThreads(ctx context.Context, category string, limit int) ([]Link, error) {
	cat := strings.Trim(category, "/")
	cat = strings.TrimPrefix(cat, "https://rns.recipes/")
	if !strings.HasPrefix(cat, "forum/") {
		cat = "forum/" + cat
	}
	if !regexp.MustCompile(`^forum/[a-z0-9-]+$`).MatchString(cat) {
		return nil, fmt.Errorf("category must be a forum section name, got %q", category)
	}
	body, err := get(ctx, "https://rns.recipes/"+cat)
	if err != nil {
		return nil, err
	}
	var out []Link
	seen := map[string]bool{}
	prefix := "/" + cat + "/"
	for _, m := range anchorRe.FindAllStringSubmatch(body, -1) {
		href, title := m[1], clean(m[2])
		href = strings.TrimPrefix(href, "https://rns.recipes")
		if strings.HasPrefix(href, prefix) && title != "" && !seen[href] &&
			!strings.HasSuffix(href, "/new") && !strings.HasSuffix(href, "feed.xml") {
			seen[href] = true
			out = append(out, Link{title, "https://rns.recipes" + href})
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// ReadPage fetches a community page and returns readable text.
// For rns.recipes forum threads it tries the markdown export endpoint so
// images and links survive. Allowed hosts: rns.recipes, unsigned.io, github.com.
func ReadPage(ctx context.Context, url string) (string, error) {
	low := strings.ToLower(url)
	ok := strings.HasPrefix(low, "https://rns.recipes/forum/") ||
		strings.HasPrefix(low, "https://unsigned.io/") ||
		strings.HasPrefix(low, "https://github.com/markqvist/Reticulum/discussions/")
	if !ok {
		return "", fmt.Errorf("url not on a community host: rns.recipes forum, unsigned.io, or markqvist/Reticulum discussions")
	}
	u, err := nurl.Parse(url)
	if err != nil {
		return "", err
	}
	body := ""
	if strings.HasPrefix(low, "https://rns.recipes/forum/") && !strings.HasSuffix(u.Path, "/export.md") {
		exp := *u
		exp.Path = strings.TrimSuffix(exp.Path, "/") + "/export.md"
		if b, err := get(ctx, exp.String()); err == nil {
			body = b
		}
	}
	if body == "" {
		body, err = get(ctx, url)
		if err != nil {
			return "", err
		}
	}
	if strings.HasSuffix(u.Path, "/export.md") {
		return relToAbs(cleanMD(body)), nil
	}
	return clean(body), nil
}

// Discussion titles on GitHub are anchors with discussion-title class.
var ghDiscussRe = regexp.MustCompile(`<a[^>]*href="(/markqvist/Reticulum/discussions/\d+)"[^>]*class="[^"]*discussion-title[^"]*"[^>]*>(.*?)</a>`)

// GitHubDiscussions lists recent discussion titles and links.
func GitHubDiscussions(ctx context.Context, limit int) ([]Link, error) {
	body, err := get(ctx, "https://github.com/markqvist/Reticulum/discussions")
	if err != nil {
		return nil, err
	}
	var out []Link
	seen := map[string]bool{}
	for _, m := range ghDiscussRe.FindAllStringSubmatch(body, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, Link{clean(m[2]), "https://github.com" + m[1]})
			if len(out) >= limit {
				break
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no discussion links parsed; GitHub markup may have changed")
	}
	return out, nil
}

var rssItemRe = regexp.MustCompile(`(?s)<item>.*?<title>(.*?)</title>.*?<link>(.*?)</link>`)

// ForumLatest parses the forum RSS feed of newest threads.
func ForumLatest(ctx context.Context, limit int) ([]Link, error) {
	body, err := get(ctx, "https://rns.recipes/forum/feed.xml")
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, m := range rssItemRe.FindAllStringSubmatch(body, -1) {
		out = append(out, Link{clean(m[1]), strings.TrimSpace(m[2])})
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no feed items parsed")
	}
	return out, nil
}

// GitHubSearch queries issues/PRs in markqvist/Reticulum via the REST
// search API (10 req/min unauthenticated; discussions are GraphQL-only).
func GitHubSearch(ctx context.Context, query string, limit int) ([]Link, error) {
	if !regexp.MustCompile(`^[\w .:/@-]{1,120}$`).MatchString(query) {
		return nil, fmt.Errorf("invalid query")
	}
	url := fmt.Sprintf("https://api.github.com/search/issues?q=repo:markqvist/Reticulum+%s&per_page=%d",
		strings.ReplaceAll(strings.TrimSpace(query), " ", "+"), limit)
	body, err := get(ctx, url)
	if err != nil {
		return nil, err
	}
	var res struct {
		Items []struct {
			Title   string `json:"title"`
			HTMLURL string `json:"html_url"`
		} `json:"items"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		return nil, err
	}
	if res.Message != "" {
		return nil, fmt.Errorf("GitHub: %s", res.Message)
	}
	var out []Link
	for _, it := range res.Items {
		out = append(out, Link{it.Title, it.HTMLURL})
	}
	return out, nil
}

// ForumSearch filters thread titles across categories by regex.
func ForumSearch(ctx context.Context, query string, limit int) ([]Link, error) {
	re, err := regexp.Compile("(?i)" + query)
	if err != nil {
		return nil, fmt.Errorf("bad regex: %w", err)
	}
	cats, err := ForumCategories(ctx)
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, c := range cats {
		threads, err := ForumThreads(ctx, c.URL, 50)
		if err != nil {
			continue
		}
		for _, t := range threads {
			if re.MatchString(t.Title) || re.MatchString(t.URL) {
				out = append(out, t)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}
