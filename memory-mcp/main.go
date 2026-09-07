// SPDX-License-Identifier: 0BSD
// Command memory-mcp is a write-capable MCP server that stores, searches,
// and recalls short notes, people, destinations, tasks, learned facts, and
// web/doc snippets for agents. All data lives in one JSONL file under a
// configured directory; paths are jailed, content and result counts are
// bounded, and secrets-style patterns are redacted from output.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Quad4-Software/ai/memory-mcp/internal/mcp"
)

const (
	maxContentBytes = 64 << 10 // 64 KiB per memory
	maxTags         = 20
	maxTagLen       = 64
	maxResults      = 50
	defaultResults  = 10
)

var (
	memoryDir  string
	memoryFile string
	memMu      sync.Mutex

	secretRe = regexp.MustCompile(`(?i)(token|key|password|secret|private)\s*[:=]\s*\S+`)
)

func init() {
	memoryDir = os.Getenv("MEMORY_DIR")
	if memoryDir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = "."
		}
		memoryDir = filepath.Join(home, ".local", "share", "ai-memory")
	}
	memoryDir = filepath.Clean(memoryDir)
	memoryFile = filepath.Join(memoryDir, "memories.jsonl")
}

// Memory is one stored item.
type Memory struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Source  string   `json:"source,omitempty"`
	Created string   `json:"created"`
	Updated string   `json:"updated"`
}

func validType(t string) string {
	switch t {
	case "note", "person", "destination", "task", "not-do", "doc", "snippet":
		return t
	}
	return ""
}

func newID(content string) string {
	h := sha256.Sum256([]byte(content + time.Now().UTC().String()))
	randBytes := make([]byte, 6)
	_, _ = rand.Read(randBytes) // #nosec G104 -- best-effort entropy; collision resistance from SHA-256
	return hex.EncodeToString(h[:6]) + hex.EncodeToString(randBytes)
}

func ensureStore() error {
	if err := os.MkdirAll(memoryDir, 0o700); err != nil { // #nosec G301 -- per-user data dir, owner-only
		return fmt.Errorf("cannot create memory dir: %w", err)
	}
	if _, err := os.Stat(memoryFile); err == nil {
		return nil
	}
	f, err := os.OpenFile(memoryFile, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("cannot create memory file: %w", err)
	}
	return f.Close()
}

func loadMemories() ([]Memory, error) {
	_ = ensureStore()
	f, err := os.Open(memoryFile) // #nosec G304 -- path is fixed under jailed dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Memory
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var m Memory
		if err := json.Unmarshal(line, &m); err != nil {
			continue // skip corrupted line
		}
		out = append(out, m)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func saveMemories(mem []Memory) error {
	if err := os.MkdirAll(memoryDir, 0o700); err != nil { // #nosec G301 -- owner-only per-user data
		return err
	}
	tmp := memoryFile + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G302,G304 -- owner-only data, path jailed to MEMORY_DIR
	if err != nil {
		return err
	}
	for _, m := range mem {
		b, err := json.Marshal(m)
		if err != nil {
			_ = f.Close()
			return err
		}
		if _, err := f.Write(b); err != nil {
			_ = f.Close()
			return err
		}
		if _, err := f.WriteString("\n"); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, memoryFile)
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(t))
		t = strings.ReplaceAll(t, " ", "-")
		if t == "" || len(t) > maxTagLen || seen[t] {
			continue
		}
		if len(out) >= maxTags {
			break
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func redact(s string) string {
	return secretRe.ReplaceAllString(s, "${1}: <redacted>")
}

func jsonMem(m Memory, preview int) string {
	if preview > 0 && len(m.Content) > preview {
		m.Content = m.Content[:preview] + "..."
	}
	m.Content = redact(m.Content)
	b, _ := json.Marshal(m)
	return string(b)
}

func scoreMemory(query string, m Memory) int {
	q := strings.ToLower(query)
	words := strings.Fields(q)
	hay := strings.ToLower(m.Content + " " + strings.Join(m.Tags, " ") + " " + m.Source + " " + m.ID)
	score := 0
	if strings.Contains(hay, q) {
		score += 30
	}
	for _, w := range words {
		if w == "" {
			continue
		}
		if strings.Contains(strings.ToLower(m.Content), w) {
			score += 10
		}
		for _, t := range m.Tags {
			if strings.Contains(t, w) {
				score += 20
			}
		}
		if strings.Contains(strings.ToLower(m.Source), w) {
			score += 5
		}
		if strings.Contains(strings.ToLower(m.ID), w) {
			score += 15
		}
		if levenshtein(strings.ToLower(m.ID), w) <= 2 && len(w) > 2 {
			score += 8
		}
	}
	return score
}

func levenshtein(a, b string) int {
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = curr
	}
	return prev[len(b)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func resultLimit(l int) int {
	if l <= 0 {
		return defaultResults
	}
	if l > maxResults {
		return maxResults
	}
	return l
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intArg(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func arrayArg(items map[string]any, desc string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": items}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "remember",
			Write:       true,
			Description: "Store a memory. Use for notes, people, RNS destinations, tasks, things to avoid, or snippets from docs/web searches.",
			InputSchema: obj(map[string]any{
				"content": strArg("The text to remember. Max 64 KiB."),
				"type":    strArg("Kind of memory: note, person, destination, task, not-do, doc, snippet. Default note."),
				"tags":    arrayArg(strArg("Tag words, max 20"), "Optional tags to improve recall."),
				"source":  strArg("Optional URL, doc path, or reference."),
			}, "content"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"content":"rns://06a54b505bb67b25ef3f8097e8001edc/public/LXMFy","type":"destination","tags":["lxmfy","public"],"source":"user prompt"}`)},
				{"arguments": json.RawMessage(`{"content":"Use go test -race before release","type":"task","tags":["workflow","release"]}`)},
			},
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Content string   `json:"content"`
					Type    string   `json:"type"`
					Tags    []string `json:"tags"`
					Source  string   `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Content == "" {
					return "", fmt.Errorf("missing required argument: content")
				}
				if len(a.Content) > maxContentBytes {
					return "", fmt.Errorf("content capped at %d bytes; got %d", maxContentBytes, len(a.Content))
				}
				if a.Type == "" {
					a.Type = "note"
				}
				if validType(a.Type) == "" {
					return "", fmt.Errorf("invalid type %q; valid: note, person, destination, task, not-do, doc, snippet", a.Type)
				}
				now := time.Now().UTC().Format(time.RFC3339)
				m := Memory{
					ID:      newID(a.Content + a.Source + now),
					Type:    a.Type,
					Content: a.Content,
					Tags:    normalizeTags(a.Tags),
					Source:  a.Source,
					Created: now,
					Updated: now,
				}
				memMu.Lock()
				defer memMu.Unlock()
				mem, err := loadMemories()
				if err != nil {
					return "", err
				}
				mem = append(mem, m)
				if err := saveMemories(mem); err != nil {
					return "", err
				}
				return fmt.Sprintf("remembered %s as %s", m.Type, m.ID), nil
			},
		},
		{
			Name:        "recall",
			Description: "Fuzzy-search memories. Matches content, tags, source, and id. Returns ranked results.",
			InputSchema: obj(map[string]any{
				"query": strArg("Search terms. Empty returns recent items."),
				"type":  strArg("Optional filter by memory type."),
				"tags":  arrayArg(strArg("Optional tags; all must match."), "Filter by tags."),
				"limit": intArg("Max results, default 10, max 50."),
			}, "query"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"query":"LXMFy destination","limit":5}`)},
				{"arguments": json.RawMessage(`{"query":"release","tags":["workflow"],"limit":3}`)},
			},
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string   `json:"query"`
					Type  string   `json:"type"`
					Tags  []string `json:"tags"`
					Limit int      `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil {
					return "", fmt.Errorf("invalid arguments: %v", err)
				}
				a.Tags = normalizeTags(a.Tags)
				if a.Type != "" && validType(a.Type) == "" {
					return "", fmt.Errorf("invalid type %q; valid: note, person, destination, task, not-do, doc, snippet", a.Type)
				}
				memMu.Lock()
				mem, err := loadMemories()
				memMu.Unlock()
				if err != nil {
					return "", err
				}
				type scored struct {
					score int
					mem   Memory
				}
				var out []scored
				for _, m := range mem {
					if a.Type != "" && m.Type != a.Type {
						continue
					}
					if !tagsMatch(a.Tags, m.Tags) {
						continue
					}
					score := 0
					if strings.TrimSpace(a.Query) != "" {
						score = scoreMemory(a.Query, m)
					}
					if score > 0 || strings.TrimSpace(a.Query) == "" {
						out = append(out, scored{score, m})
					}
				}
				sort.Slice(out, func(i, j int) bool {
					if out[i].score != out[j].score {
						return out[i].score > out[j].score
					}
					return out[i].mem.Created > out[j].mem.Created
				})
				limit := resultLimit(a.Limit)
				if len(out) > limit {
					out = out[:limit]
				}
				var lines []string
				for _, s := range out {
					lines = append(lines, jsonMem(s.mem, 512))
				}
				if len(lines) == 0 {
					return "no memories matched", nil
				}
				return strings.Join(lines, "\n"), nil
			},
		},
		{
			Name:        "memory_by_id",
			Description: "Return a single memory by id.",
			InputSchema: obj(map[string]any{
				"id": strArg("Memory id."),
			}, "id"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				memMu.Lock()
				mem, err := loadMemories()
				memMu.Unlock()
				if err != nil {
					return "", err
				}
				for _, m := range mem {
					if m.ID == a.ID {
						return jsonMem(m, 0), nil
					}
				}
				return "", fmt.Errorf("memory not found: %s", a.ID)
			},
		},
		{
			Name:        "update_memory",
			Write:       true,
			Description: "Update an existing memory's content, type, tags, or source by id.",
			InputSchema: obj(map[string]any{
				"id":      strArg("Memory id to update."),
				"content": strArg("New content. Optional, leave empty to keep."),
				"type":    strArg("New type. Optional."),
				"tags":    arrayArg(strArg("New tags. Optional; replaces existing tags."), "Replace tags."),
				"source":  strArg("New source. Optional."),
			}, "id"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string   `json:"id"`
					Content string   `json:"content"`
					Type    string   `json:"type"`
					Tags    []string `json:"tags"`
					Source  string   `json:"source"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				if a.Type != "" && validType(a.Type) == "" {
					return "", fmt.Errorf("invalid type %q; valid: note, person, destination, task, not-do, doc, snippet", a.Type)
				}
				memMu.Lock()
				defer memMu.Unlock()
				mem, err := loadMemories()
				if err != nil {
					return "", err
				}
				for i, m := range mem {
					if m.ID != a.ID {
						continue
					}
					if a.Content != "" {
						if len(a.Content) > maxContentBytes {
							return "", fmt.Errorf("content capped at %d bytes", maxContentBytes)
						}
						m.Content = a.Content
					}
					if a.Type != "" {
						m.Type = a.Type
					}
					if len(a.Tags) > 0 || (a.Tags != nil && len(a.Tags) == 0) {
						m.Tags = normalizeTags(a.Tags)
					}
					if a.Source != "" {
						m.Source = a.Source
					}
					m.Updated = time.Now().UTC().Format(time.RFC3339)
					mem[i] = m
					if err := saveMemories(mem); err != nil {
						return "", err
					}
					return fmt.Sprintf("updated %s", m.ID), nil
				}
				return "", fmt.Errorf("memory not found: %s", a.ID)
			},
		},
		{
			Name:        "forget",
			Write:       true,
			Description: "Delete one or more memories by id.",
			InputSchema: obj(map[string]any{
				"ids": arrayArg(strArg("Memory ids to delete."), "Memory ids."),
			}, "ids"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					IDs []string `json:"ids"`
				}
				if err := json.Unmarshal(args, &a); err != nil || len(a.IDs) == 0 {
					return "", fmt.Errorf("missing required argument: ids")
				}
				seen := make(map[string]bool)
				for _, id := range a.IDs {
					seen[id] = true
				}
				memMu.Lock()
				defer memMu.Unlock()
				mem, err := loadMemories()
				if err != nil {
					return "", err
				}
				var keep []Memory
				for _, m := range mem {
					if !seen[m.ID] {
						keep = append(keep, m)
					}
				}
				if len(keep) == len(mem) {
					return "no matching memories deleted", nil
				}
				if err := saveMemories(keep); err != nil {
					return "", err
				}
				return fmt.Sprintf("forgot %d memory(s)", len(mem)-len(keep)), nil
			},
		},
		{
			Name:        "list_memory",
			Description: "List memories with optional type/tag filters and pagination.",
			InputSchema: obj(map[string]any{
				"type":   strArg("Optional memory type."),
				"tags":   arrayArg(strArg("Optional tags; all must match."), "Filter by tags."),
				"limit":  intArg("Max results, default 10, max 50."),
				"offset": intArg("Skip first N results."),
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Type   string   `json:"type"`
					Tags   []string `json:"tags"`
					Limit  int      `json:"limit"`
					Offset int      `json:"offset"`
				}
				_ = json.Unmarshal(args, &a)
				a.Tags = normalizeTags(a.Tags)
				if a.Type != "" && validType(a.Type) == "" {
					return "", fmt.Errorf("invalid type %q; valid: note, person, destination, task, not-do, doc, snippet", a.Type)
				}
				memMu.Lock()
				mem, err := loadMemories()
				memMu.Unlock()
				if err != nil {
					return "", err
				}
				sort.Slice(mem, func(i, j int) bool { return mem[i].Created > mem[j].Created })
				var filtered []Memory
				for _, m := range mem {
					if a.Type != "" && m.Type != a.Type {
						continue
					}
					if !tagsMatch(a.Tags, m.Tags) {
						continue
					}
					filtered = append(filtered, m)
				}
				start := a.Offset
				if start > len(filtered) {
					start = len(filtered)
				}
				limit := resultLimit(a.Limit)
				end := start + limit
				if end > len(filtered) {
					end = len(filtered)
				}
				var lines []string
				for _, m := range filtered[start:end] {
					lines = append(lines, jsonMem(m, 512))
				}
				if len(lines) == 0 {
					return "no memories", nil
				}
				return fmt.Sprintf("showing %d-%d of %d\n%s", start+1, end, len(filtered), strings.Join(lines, "\n")), nil
			},
		},
		{
			Name:        "memory_tags",
			Description: "List all tags with counts.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				memMu.Lock()
				mem, err := loadMemories()
				memMu.Unlock()
				if err != nil {
					return "", err
				}
				counts := make(map[string]int)
				for _, m := range mem {
					for _, t := range m.Tags {
						counts[t]++
					}
				}
				var tags []string
				for t := range counts {
					tags = append(tags, t)
				}
				sort.Strings(tags)
				var lines []string
				for _, t := range tags {
					lines = append(lines, fmt.Sprintf("%s: %d", t, counts[t]))
				}
				if len(lines) == 0 {
					return "no tags", nil
				}
				return strings.Join(lines, "\n"), nil
			},
		},
		{
			Name:        "memory_status",
			Description: "Return the number of stored memories and the data directory.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				memMu.Lock()
				mem, err := loadMemories()
				memMu.Unlock()
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%d memory(s) in %s", len(mem), memoryDir), nil
			},
		},
	}
}

func tagsMatch(required, have []string) bool {
	haveSet := make(map[string]bool)
	for _, h := range have {
		haveSet[h] = true
	}
	for _, r := range required {
		if !haveSet[r] {
			return false
		}
	}
	return true
}

func main() {
	_ = ensureStore()
	srv := mcp.NewServer("memory-mcp", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "memory-mcp:", err)
		os.Exit(1)
	}
}
