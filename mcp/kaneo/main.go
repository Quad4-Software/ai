// SPDX-License-Identifier: 0BSD
// Command kaneo is a stdio MCP server for the Kaneo project management
// API (cloud or self-hosted). Read tools work with no key on public
// projects; write tools need an API key from KANEO_API_KEY or
// ~/.config/kaneo/config.json. The key is never printed. Run
// "kaneo setup" to store a key interactively.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quad4-Software/ai/mcp/kaneo/internal/kaneo"
	"github.com/Quad4-Software/ai/mcp/kaneo/internal/mcp"
)

var client *kaneo.Client
var cfg *kaneo.Config

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

// needKey gates write tools on a configured API key.
func needKey() error {
	if client == nil || !client.Configured() {
		return fmt.Errorf("no Kaneo API key configured; run `kaneo setup` or set KANEO_API_KEY")
	}
	return nil
}

func str(args json.RawMessage, names ...string) (map[string]string, error) {
	var a map[string]string
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid arguments: %v", err)
	}
	for _, n := range names {
		if strings.TrimSpace(a[n]) == "" {
			return nil, fmt.Errorf("missing required argument: %s", n)
		}
	}
	return a, nil
}

func marshal(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return string(b), err
}

func tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "auth_status",
			Description: "Report whether a Kaneo API key is configured and where it came from. The key itself is never shown.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				return marshal(map[string]any{
					"configured":  client.Configured(),
					"source":      cfg.Source,
					"apiUrl":      cfg.APIURL,
					"workspaceId": cfg.WorkspaceID,
					"projectId":   cfg.ProjectID,
				})
			},
		},
		{
			Name:        "list_projects",
			Description: "List projects in a workspace.",
			InputSchema: obj(map[string]any{
				"workspaceId": strArg("workspace id, default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					WorkspaceID string `json:"workspaceId"`
				}
				_ = json.Unmarshal(args, &a) // #nosec G104 -- optional arg
				ps, err := client.ListProjects(ctx, a.WorkspaceID)
				if err != nil {
					return "", err
				}
				return marshal(ps)
			},
		},
		{
			Name:        "get_board",
			Description: "Fetch a project board: columns plus every task in each column (id, number, title, priority, position). Descriptions are omitted; use get_task for those.",
			InputSchema: obj(map[string]any{
				"projectId": strArg("project id, default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				b, err := client.GetBoard(ctx, a.ProjectID)
				if err != nil {
					return "", err
				}
				// compact view: drop descriptions to bound output
				type col struct {
					Slug  string      `json:"slug"`
					Name  string      `json:"name"`
					Tasks []taskBrief `json:"tasks"`
				}
				out := map[string]any{"id": b.ID, "name": b.Name, "slug": b.Slug}
				var cols []col
				for _, c := range b.Columns {
					cc := col{Slug: c.Slug, Name: c.Name}
					for _, t := range c.Tasks {
						cc.Tasks = append(cc.Tasks, brief(t))
					}
					cols = append(cols, cc)
				}
				out["columns"] = cols
				return marshal(out)
			},
		},
		{
			Name:        "get_task",
			Description: "Fetch one task with full description, labels, and assignee.",
			InputSchema: obj(map[string]any{"id": strArg("task id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				t, err := client.GetTask(ctx, a["id"])
				if err != nil {
					return "", err
				}
				return marshal(t)
			},
		},
		{
			Name:        "find_tasks",
			Description: "Search board tasks by title substring. Use for dedupe before create_task.",
			InputSchema: obj(map[string]any{
				"query":     strArg("case-insensitive substring of the title"),
				"projectId": strArg("project id, default from config"),
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "query")
				if err != nil {
					return "", err
				}
				b, err := client.GetBoard(ctx, a["projectId"])
				if err != nil {
					return "", err
				}
				q := strings.ToLower(a["query"])
				var hits []taskBrief
				for _, c := range b.Columns {
					for _, t := range c.Tasks {
						if strings.Contains(strings.ToLower(t.Title), q) {
							tb := brief(t)
							tb.Column = c.Slug
							hits = append(hits, tb)
						}
					}
				}
				return marshal(hits)
			},
		},
		{
			Name:        "list_labels",
			Description: "List workspace labels.",
			InputSchema: obj(map[string]any{
				"workspaceId": strArg("workspace id, default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					WorkspaceID string `json:"workspaceId"`
				}
				_ = json.Unmarshal(args, &a)
				ls, err := client.ListLabels(ctx, a.WorkspaceID)
				if err != nil {
					return "", err
				}
				return marshal(ls)
			},
		},
		{
			Name:        "get_task_comments",
			Description: "List comments on a task.",
			InputSchema: obj(map[string]any{"taskId": strArg("task id")}, "taskId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "taskId")
				if err != nil {
					return "", err
				}
				cs, err := client.GetComments(ctx, a["taskId"])
				if err != nil {
					return "", err
				}
				return marshal(cs)
			},
		},
		{
			Name:        "create_task",
			Description: "Create a task in a project. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"title":       strArg("task title"),
				"description": strArg("markdown body"),
				"priority":    strArg("no-priority|low|medium|high|urgent"),
				"status":      strArg("column slug, default to-do"),
				"dueDate":     strArg("ISO date-time, optional"),
				"projectId":   strArg("project id, default from config"),
			}, "title"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"title": "Add RSS feed", "priority": "medium"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "title")
				if err != nil {
					return "", err
				}
				t, err := client.CreateTask(ctx, a["projectId"], a["title"],
					a["description"], a["priority"], a["status"], a["dueDate"])
				if err != nil {
					return "", err
				}
				return marshal(t)
			},
		},
		{
			Name:        "update_task",
			Description: "Update fields of a task. Only fields you pass are changed. To clear a field pass a single space for description or use the API. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":          strArg("task id"),
				"title":       strArg("new title"),
				"description": strArg("new markdown body"),
				"priority":    strArg("no-priority|low|medium|high|urgent"),
				"status":      strArg("column slug"),
				"dueDate":     strArg("ISO date-time"),
			}, "id"),
			InputExamples: []map[string]any{
				{"arguments": json.RawMessage(`{"id": "abc", "status": "in-progress", "priority": "high"}`)},
			},
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				t, err := client.UpdateTask(ctx, a["id"], kaneo.Patch{
					Title: a["title"], Description: a["description"],
					Priority: a["priority"], Status: a["status"], DueDate: a["dueDate"],
				})
				if err != nil {
					return "", err
				}
				return marshal(t)
			},
		},
		{
			Name:        "move_task",
			Description: "Move a task to another column, or to another project when destinationProjectId is given. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":                   strArg("task id"),
				"status":               strArg("destination column slug: backlog|to-do|in-progress|in-review|done|cancelled"),
				"destinationProjectId": strArg("optional, move across projects"),
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.MoveTask(ctx, a["id"], a["status"], a["destinationProjectId"]); err != nil {
					return "", err
				}
				return "moved", nil
			},
		},
		{
			Name:        "add_comment",
			Description: "Post a comment on a task. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"taskId":  strArg("task id"),
				"content": strArg("comment markdown"),
			}, "taskId", "content"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "taskId", "content")
				if err != nil {
					return "", err
				}
				if err := client.AddComment(ctx, a["taskId"], a["content"]); err != nil {
					return "", err
				}
				return "comment added", nil
			},
		},
		{
			Name:        "set_label",
			Description: "Attach or detach a label on a task. Label may be an id or an exact name. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"taskId":      strArg("task id"),
				"label":       strArg("label id or exact name"),
				"attach":      map[string]any{"type": "boolean", "description": "default true; false detaches"},
				"workspaceId": strArg("workspace id for name lookup, default from config"),
			}, "taskId", "label"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					TaskID      string `json:"taskId"`
					Label       string `json:"label"`
					Attach      *bool  `json:"attach"`
					WorkspaceID string `json:"workspaceId"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.TaskID == "" || a.Label == "" {
					return "", fmt.Errorf("missing required arguments: taskId, label")
				}
				attach := a.Attach == nil || *a.Attach
				if err := client.SetLabel(ctx, a.TaskID, a.Label, a.WorkspaceID, attach); err != nil {
					return "", err
				}
				if attach {
					return "label attached", nil
				}
				return "label detached", nil
			},
		},
		{
			Name:        "delete_task",
			Description: "Permanently delete a task. Destructive; confirm with the user first. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{"id": strArg("task id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.DeleteTask(ctx, a["id"]); err != nil {
					return "", err
				}
				return "deleted", nil
			},
		},
	}
}

// taskBrief is the compact task shape list endpoints return.
type taskBrief struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Column   string `json:"column,omitempty"`
}

func brief(t kaneo.Task) taskBrief {
	return taskBrief{ID: t.ID, Number: t.Number, Title: t.Title,
		Status: t.Status, Priority: t.Priority}
}

// setup stores connection settings in ~/.config/kaneo/config.json with
// owner-only permissions. The key is read from the terminal with echo
// disabled where the platform supports it, so it never lands in shell
// history, logs, or tool output.
func setup() error {
	cfg := kaneo.Config{APIURL: kaneo.DefaultAPI}
	reader := bufio.NewReader(os.Stdin)
	ask := func(label, cur string) string {
		if cur != "" {
			fmt.Fprintf(os.Stderr, "%s [%s]: ", label, cur)
		} else {
			fmt.Fprintf(os.Stderr, "%s: ", label)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return cur
		}
		return line
	}

	cfg.APIURL = ask("API URL", getenvOr("KANEO_API_URL", kaneo.DefaultAPI))
	cfg.WorkspaceID = ask("Workspace ID", getenvOr("KANEO_WORKSPACE_ID", ""))
	cfg.ProjectID = ask("Default project ID", getenvOr("KANEO_PROJECT_ID", ""))

	fmt.Fprint(os.Stderr, "API key: ")
	key, err := readSecret(reader)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("empty key, nothing written")
	}
	cfg.APIKey = key

	p, err := kaneo.ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	// #nosec G117 -- setup exists to persist the key into a 0600 file
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (0600)\n", p)
	return nil
}

func getenvOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// readSecret reads one line with terminal echo off when possible.
// Falls back to a visible prompt on platforms without stty.
func readSecret(r *bufio.Reader) (string, error) {
	if runtime.GOOS != "windows" {
		if tty, err := os.Open("/dev/tty"); err == nil {
			defer tty.Close()
			off := exec.Command("stty", "-echo") // argv form, no shell
			off.Stdin = tty
			on := exec.Command("stty", "echo")
			on.Stdin = tty
			if off.Run() == nil {
				line, err := r.ReadString('\n')
				_ = on.Run() // #nosec G104 -- best effort restore
				return strings.TrimSpace(line), err
			}
		}
	}
	fmt.Fprint(os.Stderr, "(input visible) ")
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		if err := setup(); err != nil {
			fmt.Fprintln(os.Stderr, "kaneo setup:", err)
			os.Exit(1)
		}
		return
	}
	var err error
	cfg, err = kaneo.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaneo:", err)
		os.Exit(1)
	}
	client, err = kaneo.NewClient(*cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaneo:", err)
		os.Exit(1)
	}
	srv := mcp.NewServer("kaneo", "0.1.0", tools(), nil)
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "kaneo:", err)
		os.Exit(1)
	}
}
