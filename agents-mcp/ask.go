package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Quad4-Software/ai/agents-mcp/internal/mcp"
)

type fuzzyHit struct {
	path  string
	line  int
	text  string
	score int
}

var wordRe = regexp.MustCompile("[a-zA-Z0-9]{3,}")

var askTool = mcp.Tool{
	Name:        "ask",
	Description: "Answer a question by returning the most relevant .agents skills and docs. Fuzzy-matches the question against skill names, descriptions, headings and contents, then returns the top-matching skill contents plus matching snippets from conventions and AGENTS.md.",
	InputSchema: obj(map[string]any{
		"question": strArg("the question you want answered, e.g. 'how do I configure an RNode'"),
		"limit":    map[string]any{"type": "integer", "description": "max number of matching skills and snippets, default 8"},
	}, "question"),
	Handle: func(_ context.Context, args json.RawMessage) (string, error) {
		var a struct {
			Question string `json:"question"`
			Limit    int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Question == "" {
			return "", fmt.Errorf("missing required argument: question")
		}
		if a.Limit <= 0 || a.Limit > 20 {
			a.Limit = 8
		}
		return askContext(a.Question, a.Limit)
	},
}

var fuzzySearchTool = mcp.Tool{
	Name:        "fuzzy_search",
	Description: "Fuzzy, token-based search across .agents skills, conventions and AGENTS.md. Returns ranked file:line snippets that best match the query terms.",
	InputSchema: obj(map[string]any{
		"query": strArg("free-text search terms, e.g. 'propagation node stamp'"),
		"limit": map[string]any{"type": "integer", "description": "max snippets, default 20"},
	}, "query"),
	Handle: func(_ context.Context, args json.RawMessage) (string, error) {
		var a struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
			return "", fmt.Errorf("missing required argument: query")
		}
		if a.Limit <= 0 || a.Limit > 100 {
			a.Limit = 20
		}
		return fuzzySearch(a.Query, a.Limit)
	},
}

func tokenWords(s string) map[string]bool {
	words := map[string]bool{}
	for _, w := range wordRe.FindAllString(s, -1) {
		words[strings.ToLower(w)] = true
	}
	return words
}

func fuzzySearch(query string, limit int) (string, error) {
	words := tokenWords(query)
	var hits []fuzzyHit
	for _, sub := range []string{".agents", "docs", "AGENTS.md"} {
		base, err := jailed(sub)
		if err != nil {
			continue
		}
		st, err := os.Stat(base)
		if err != nil {
			continue
		}
		if st.IsDir() {
			walk := func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				if !strings.HasSuffix(d.Name(), ".md") && !strings.HasSuffix(d.Name(), ".mdc") {
					return nil
				}
				data, err := os.ReadFile(p) // #nosec G122 G304 -- read-only walk; path comes from WalkDir itself
				if err != nil || len(data) > maxFile {
					return nil
				}
				rel, _ := filepath.Rel(root, p)
				relToks := tokenWords(rel)
				for i, line := range strings.Split(string(data), string(rune(10))) {
					low := strings.ToLower(line)
					score := 0
					for w := range words {
						if strings.Contains(low, w) {
							score++
						}
					}
					if score == 0 {
						continue
					}
					for w := range words {
						if relToks[w] {
							score += 2
						}
						if strings.HasPrefix(strings.TrimSpace(line), "#") && strings.Contains(low, w) {
							score += 3
						}
					}
					hits = append(hits, fuzzyHit{filepath.ToSlash(rel), i + 1, strings.TrimSpace(line), score})
				}
				return nil
			}
			filepath.WalkDir(base, walk) // #nosec G104 -- error tolerated; empty/default is handled downstream
		} else {
			data, _ := os.ReadFile(base) // #nosec G304 -- path jailed to repo root or local config
			for i, line := range strings.Split(string(data), string(rune(10))) {
				low := strings.ToLower(line)
				score := 0
				for w := range words {
					if strings.Contains(low, w) {
						score++
					}
				}
				if score > 0 {
					hits = append(hits, fuzzyHit{sub, i + 1, strings.TrimSpace(line), score})
				}
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	if len(hits) == 0 {
		return "no matches", nil
	}
	var b strings.Builder
	for _, h := range hits {
		fmt.Fprintf(&b, "%s:%d: %s", h.path, h.line, h.text)
		fmt.Fprintln(&b)
	}
	return b.String(), nil
}

func askContext(question string, limit int) (string, error) {
	skillJSON, err := skillFor(question, 3)
	if err != nil {
		return "", err
	}
	var matches []SkillMatch
	if err := json.Unmarshal([]byte(skillJSON), &matches); err != nil {
		return "", err
	}
	var b strings.Builder
	if len(matches) > 0 {
		fmt.Fprintf(&b, "Relevant skills for %q:", question)
		fmt.Fprintln(&b)
		for _, m := range matches {
			fmt.Fprintf(&b, "- %s (%s)", m.Name, m.Path)
			fmt.Fprintln(&b)
		}
		fmt.Fprintln(&b)
	}
	for _, m := range matches {
		data, err := readJailed(m.Path)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "=== %s ===", m.Path)
		fmt.Fprintln(&b)
		fmt.Fprint(&b, data)
		fmt.Fprintln(&b)
		fmt.Fprintln(&b)
	}
	snippets, err := fuzzySearch(question, limit)
	if err != nil {
		return b.String(), err
	}
	fmt.Fprintf(&b, "=== related snippets ===")
	fmt.Fprintln(&b)
	fmt.Fprint(&b, snippets)
	fmt.Fprintln(&b)
	return b.String(), nil
}
