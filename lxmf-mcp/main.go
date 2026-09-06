// SPDX-License-Identifier: 0BSD
// Command lxmf-mcp reads the local Reticulum instance state
// (~/.reticulum): sanitized config, storage inventory, identity names,
// and decoded known destinations. Read-only. Private key material is
// never returned: identity files are listed by name only, and config
// secrets are redacted. Stdlib only; python3 is used solely to decode
// msgpack storage files.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Quad4-Software/ai/lxmf-mcp/internal/mcp"
	"github.com/Quad4-Software/ai/lxmf-mcp/internal/mpack"
)

var configDir string

var secretKeys = regexp.MustCompile(`(?i)key|secret|pass|token|ifac|networkname`)

func init() {
	if v := os.Getenv("MCP_RNS_CONFIG"); v != "" {
		configDir = v
		return
	}
	home, _ := os.UserHomeDir()
	configDir = filepath.Join(home, ".reticulum")
}

type destRow struct {
	Dest      string  `json:"dest"`
	FirstSeen float64 `json:"first_seen"`
	LastSeen  float64 `json:"last_seen"`
	Name      string  `json:"name,omitempty"`
}

var printable = regexp.MustCompile(`[\x20-\x7e]{4,}`)

func decodeKnownDestinations(path string, limit int) ([]destRow, error) {
	f, err := os.Open(path) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := mpack.Decode(f)
	if err != nil {
		return nil, err
	}
	m, ok := raw.(map[any]any)
	if !ok {
		return nil, fmt.Errorf("unexpected store format %T", raw)
	}
	var rows []destRow
	for k, v := range m {
		var row destRow
		switch kk := k.(type) {
		case string:
			row.Dest = hex.EncodeToString([]byte(kk))
		case []byte:
			row.Dest = hex.EncodeToString(kk)
		default:
			row.Dest = fmt.Sprint(k)
		}
		arr, _ := v.([]any)
		if len(arr) > 0 {
			row.FirstSeen, _ = arr[0].(float64)
		}
		if len(arr) > 4 {
			row.LastSeen, _ = arr[4].(float64)
		}
		if len(arr) > 3 {
			if nb, ok := arr[3].([]byte); ok {
				if runs := printable.FindAllString(string(nb), -1); len(runs) > 0 {
					row.Name = runs[0]
					for _, s := range runs {
						if len(s) > len(row.Name) {
							row.Name = s
						}
					}
				}
			} else if ns, ok := arr[3].(string); ok {
				row.Name = ns
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].LastSeen > rows[j].LastSeen })
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

// parseINI returns section -> []lines with secret values redacted.
func parseINI(path string) (string, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for line := range strings.SplitSeq(string(b), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") {
			continue
		}
		if strings.HasPrefix(t, "[") {
			out.WriteString(t + "\n")
			continue
		}
		if i := strings.IndexByte(t, '='); i > 0 {
			k := strings.TrimSpace(t[:i])
			v := strings.TrimSpace(t[i+1:])
			if secretKeys.MatchString(k) {
				v = "[redacted]"
			}
			fmt.Fprintf(&out, "    %s = %s\n", k, v)
		}
	}
	return out.String(), nil
}

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:          "store_config",
			Description:   "Sanitized view of the local Reticulum config (sections and interfaces; secrets redacted).",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{}`)}},
			InputSchema:   obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return parseINI(filepath.Join(configDir, "config"))
			},
		},
		{
			Name:        "store_inventory",
			Description: "Inventory of ~/.reticulum storage: per-entry size and file counts.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				var b strings.Builder
				fmt.Fprintf(&b, "config dir: %s\n", configDir)
				for _, sub := range []string{"", "storage", "identities", "interfaces"} {
					base := filepath.Join(configDir, sub)
					ents, err := os.ReadDir(base)
					if err != nil {
						continue
					}
					for _, e := range ents {
						p := filepath.Join(base, e.Name())
						var size int64
						count := 1
						if e.IsDir() {
							count = 0
							filepath.WalkDir(p, func(_ string, d os.DirEntry, err error) error { // #nosec G104 -- error tolerated; empty/default is handled downstream
								if err == nil && !d.IsDir() {
									count++
									if fi, err := d.Info(); err == nil {
										size += fi.Size()
									}
								}
								return nil
							})
						} else if fi, err := e.Info(); err == nil {
							size = fi.Size()
						}
						rel := filepath.Join(sub, e.Name())
						fmt.Fprintf(&b, "%-44s %10d B  %d files\n", rel, size, count)
					}
				}
				return b.String(), nil
			},
		},
		{
			Name:        "store_identities",
			Description: "List identity names/dirs under ~/.reticulum/identities and storage/identities. Contents are never read.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				var b strings.Builder
				for _, sub := range []string{"identities", filepath.Join("storage", "identities")} {
					base := filepath.Join(configDir, sub)
					ents, err := os.ReadDir(base)
					if err != nil {
						continue
					}
					fmt.Fprintf(&b, "%s:\n", sub)
					for _, e := range ents {
						fmt.Fprintf(&b, "  %s\n", e.Name())
					}
				}
				if b.Len() == 0 {
					return "no identities found", nil
				}
				return b.String(), nil
			},
		},
		{
			Name:          "store_destinations",
			Description:   "Decode the known_destinations store: destination hash, display name/app data, first and last seen. Sorted by most recent.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"limit": 20}`)}},
			InputSchema: obj(map[string]any{
				"limit": strArg("max entries, default 25, digits only"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Limit string `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				lim := 25
				if a.Limit != "" {
					n, err := strconv.Atoi(a.Limit)
					if err != nil || n < 1 || n > 500 {
						return "", fmt.Errorf("invalid limit %q", a.Limit)
					}
					lim = n
				}
				path := filepath.Join(configDir, "storage", "known_destinations")
				rows, err := decodeKnownDestinations(path, lim)
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(rows, "", "  ")
				return string(b), err
			},
		},
	}
}

func main() {
	srv := mcp.NewServer("lxmf-mcp", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lxmf-mcp:", err)
		os.Exit(1)
	}
}
