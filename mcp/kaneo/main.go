// SPDX-License-Identifier: 0BSD
// Command kaneo is a stdio MCP server for the Kaneo project management
// API (cloud or self-hosted). Read tools work with no key on public
// projects; write tools need an API key from KANEO_API_KEY or the OS
// keyring (secret-tool/pass). The key is never printed and never
// written to config.json. Run "kaneo setup" to store a key in the
// keyring and pick a default project interactively.
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
	"strconv"
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

func boolArg(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

// orderArg is the shared schema for reorder tools: a list of
// {id, position} pairs expressing relative order.
func orderArg() map[string]any {
	return map[string]any{
		"type":        "array",
		"description": "list of {id, position} pairs",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":       strArg("object id"),
				"position": map[string]any{"type": "integer", "description": "relative order"},
			},
			"required": []string{"id", "position"},
		},
	}
}

// deriveSlug builds a task prefix from a project name: uppercase
// alphanumerics, at most 5 chars (MESHCHATX -> MESHC).
func deriveSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
		if b.Len() >= 5 {
			break
		}
	}
	return b.String()
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
			Description: "Report whether a Kaneo API key is configured, where it came from, and which projects are configured. The key itself is never shown.",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				var names []string
				for _, p := range cfg.Projects {
					names = append(names, p.Name)
				}
				return marshal(map[string]any{
					"configured":     client.Configured(),
					"source":         cfg.Source,
					"apiUrl":         cfg.APIURL,
					"workspaceId":    cfg.WorkspaceID,
					"defaultProject": client.DefaultProject(),
					"projects":       names,
				})
			},
		},
		{
			Name:        "list_workspaces",
			Description: "List the workspaces (organizations) the API key can access. Use the ids with list_projects.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				ws, err := client.ListWorkspaces(ctx)
				if err != nil {
					return "", err
				}
				return marshal(ws)
			},
		},
		{
			Name:        "use_project",
			Description: "Set the session default project. Accepts a configured project name, slug, or id. Afterwards tools that omit projectId use this project.",
			InputSchema: obj(map[string]any{
				"project": strArg("project name, slug, or id"),
			}, "project"),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "project")
				if err != nil {
					return "", err
				}
				client.SetDefaultProject(a["project"])
				return marshal(map[string]any{
					"defaultProject": client.DefaultProject(),
				})
			},
		},
		{
			Name:        "list_projects",
			Description: "List projects in a workspace.",
			InputSchema: obj(map[string]any{
				"workspaceId": strArg("workspace name or id; default from config"),
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
			Name:        "get_project",
			Description: "Fetch one project by name, slug, or id.",
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				p, err := client.GetProject(ctx, a.ProjectID)
				if err != nil {
					return "", err
				}
				return marshal(p)
			},
		},
		{
			Name:        "create_project",
			Description: "Create a project in a workspace. slug is the task-id prefix (e.g. MEL); derived from name when omitted. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"name":        strArg("project name"),
				"slug":        strArg("task prefix, e.g. MEL; derived from name when omitted"),
				"icon":        strArg("icon name, default folder"),
				"description": strArg("project description"),
				"workspaceId": strArg("workspace name or id; default from config"),
			}, "name"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "name")
				if err != nil {
					return "", err
				}
				slug := a["slug"]
				if slug == "" {
					slug = deriveSlug(a["name"])
				}
				p, err := client.CreateProject(ctx, a["name"], slug,
					a["icon"], a["description"], a["workspaceId"])
				if err != nil {
					return "", err
				}
				return marshal(p)
			},
		},
		{
			Name:        "update_project",
			Description: "Update a project's name, slug, icon, description, or visibility. Omitted fields keep their current values. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":          strArg("project id"),
				"name":        strArg("new name"),
				"slug":        strArg("new task prefix"),
				"icon":        strArg("new icon"),
				"description": strArg("new description"),
				"isPublic":    map[string]any{"type": "boolean", "description": "make the project publicly readable"},
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Slug        string `json:"slug"`
					Icon        string `json:"icon"`
					Description string `json:"description"`
					Public      *bool  `json:"isPublic"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				p, err := client.UpdateProject(ctx, a.ID, kaneo.ProjectPatch{
					Name: a.Name, Slug: a.Slug, Icon: a.Icon,
					Description: a.Description, Public: a.Public,
				})
				if err != nil {
					return "", err
				}
				return marshal(p)
			},
		},
		{
			Name:        "archive_project",
			Description: "Archive a project to hide it, or restore it with archive=false. Keeps all data. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":      strArg("project id"),
				"archive": map[string]any{"type": "boolean", "description": "default true; false restores (unarchives)"},
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ID      string `json:"id"`
					Archive *bool  `json:"archive"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				archive := a.Archive == nil || *a.Archive
				if err := client.ArchiveProject(ctx, a.ID, archive); err != nil {
					return "", err
				}
				if archive {
					return "archived", nil
				}
				return "unarchived", nil
			},
		},
		{
			Name:        "reorder_projects",
			Description: "Set the sidebar order of a workspace's projects. order is a list of {id, position}; positions express relative order only. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"order": orderArg(),
			}, "order"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					Order []kaneo.IDPos `json:"order"`
				}
				if err := json.Unmarshal(args, &a); err != nil || len(a.Order) == 0 {
					return "", fmt.Errorf("missing required argument: order")
				}
				if err := client.ReorderProjects(ctx, a.Order); err != nil {
					return "", err
				}
				return "reordered", nil
			},
		},
		{
			Name:        "delete_project",
			Description: "Permanently delete a project and everything in it. Destructive; confirm with the user first. Consider archive_project instead. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{"id": strArg("project id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.DeleteProject(ctx, a["id"]); err != nil {
					return "", err
				}
				return "deleted", nil
			},
		},
		{
			Name:        "get_board",
			Description: "Fetch a project board: columns plus every task in each column (id, number, title, priority, position). Descriptions are omitted; use get_task for those.",
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
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
				"projectId": strArg("project name, slug, or id; default from config"),
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
				"workspaceId": strArg("workspace name or id; default from config"),
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
				"projectId":   strArg("project name, slug, or id; default from config"),
				"assignee":    strArg("user id to assign, optional"),
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
					a["description"], a["priority"], a["status"], a["dueDate"], a["assignee"])
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
				"assignee":    strArg("user id to assign, or \"none\" to unassign"),
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
					Assignee: a["assignee"],
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
			Name:        "list_columns",
			Description: "List a project's board columns in order (id, slug, name, isFinal, position).",
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				cols, err := client.ListColumns(ctx, a.ProjectID)
				if err != nil {
					return "", err
				}
				return marshal(cols)
			},
		},
		{
			Name:        "create_column",
			Description: "Add a board column to a project. isFinal marks the done column. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
				"name":      strArg("column name"),
				"icon":      strArg("icon name"),
				"color":     strArg("hex color like #e11d48"),
				"isFinal":   boolArg("marks the column that counts tasks as done"),
			}, "name"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ProjectID string `json:"projectId"`
					Name      string `json:"name"`
					Icon      string `json:"icon"`
					Color     string `json:"color"`
					IsFinal   *bool  `json:"isFinal"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
					return "", fmt.Errorf("missing required argument: name")
				}
				col, err := client.CreateColumn(ctx, a.ProjectID, a.Name, a.Icon, a.Color, a.IsFinal)
				if err != nil {
					return "", err
				}
				return marshal(col)
			},
		},
		{
			Name:        "update_column",
			Description: "Rename or restyle a board column. Omitted fields keep their values. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":      strArg("column id"),
				"name":    strArg("new name"),
				"icon":    strArg("new icon"),
				"color":   strArg("new hex color"),
				"isFinal": boolArg("marks the done column"),
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Icon    string `json:"icon"`
					Color   string `json:"color"`
					IsFinal *bool  `json:"isFinal"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				col, err := client.UpdateColumn(ctx, a.ID, a.Name, a.Icon, a.Color, a.IsFinal)
				if err != nil {
					return "", err
				}
				return marshal(col)
			},
		},
		{
			Name:        "reorder_columns",
			Description: "Set column order within a project. order is a list of {id, position}. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
				"order":     orderArg(),
			}, "order"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ProjectID string        `json:"projectId"`
					Order     []kaneo.IDPos `json:"order"`
				}
				if err := json.Unmarshal(args, &a); err != nil || len(a.Order) == 0 {
					return "", fmt.Errorf("missing required argument: order")
				}
				if err := client.ReorderColumns(ctx, a.ProjectID, a.Order); err != nil {
					return "", err
				}
				return "reordered", nil
			},
		},
		{
			Name:        "delete_column",
			Description: "Remove a board column. Destructive when it holds tasks; confirm with the user first. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{"id": strArg("column id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.DeleteColumn(ctx, a["id"]); err != nil {
					return "", err
				}
				return "deleted", nil
			},
		},
		{
			Name:        "update_comment",
			Description: "Edit a comment's body. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":      strArg("comment id"),
				"content": strArg("new markdown body"),
			}, "id", "content"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id", "content")
				if err != nil {
					return "", err
				}
				if err := client.UpdateComment(ctx, a["id"], a["content"]); err != nil {
					return "", err
				}
				return "comment updated", nil
			},
		},
		{
			Name:        "delete_comment",
			Description: "Delete a comment. Destructive; confirm with the user first. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{"id": strArg("comment id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.DeleteComment(ctx, a["id"]); err != nil {
					return "", err
				}
				return "deleted", nil
			},
		},
		{
			Name:        "create_label",
			Description: "Create a workspace label. color is a hex string like #e11d48. taskId optionally attaches it at creation. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"name":        strArg("label name"),
				"color":       strArg("hex color like #e11d48"),
				"workspaceId": strArg("workspace name or id; default from config"),
				"taskId":      strArg("optional task to attach"),
			}, "name", "color"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "name", "color")
				if err != nil {
					return "", err
				}
				l, err := client.CreateLabel(ctx, a["name"], a["color"], a["workspaceId"], a["taskId"])
				if err != nil {
					return "", err
				}
				return marshal(l)
			},
		},
		{
			Name:        "update_label",
			Description: "Rename or recolor a workspace label. Omitted fields keep their values. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":    strArg("label id"),
				"name":  strArg("new name"),
				"color": strArg("new hex color"),
			}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				l, err := client.UpdateLabel(ctx, a["id"], a["name"], a["color"])
				if err != nil {
					return "", err
				}
				return marshal(l)
			},
		},
		{
			Name:        "delete_label",
			Description: "Delete a workspace label. Destructive; confirm with the user first. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{"id": strArg("label id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.DeleteLabel(ctx, a["id"]); err != nil {
					return "", err
				}
				return "deleted", nil
			},
		},
		{
			Name:        "get_task_relations",
			Description: "List a task's relations (subtask, blocks, related).",
			InputSchema: obj(map[string]any{"taskId": strArg("task id")}, "taskId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "taskId")
				if err != nil {
					return "", err
				}
				rels, err := client.GetTaskRelations(ctx, a["taskId"])
				if err != nil {
					return "", err
				}
				return marshal(rels)
			},
		},
		{
			Name:        "link_tasks",
			Description: "Create a relation between two tasks: subtask, blocks, or related. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"sourceTaskId": strArg("source task id"),
				"targetTaskId": strArg("target task id"),
				"relation":     strArg("subtask|blocks|related"),
			}, "sourceTaskId", "targetTaskId", "relation"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "sourceTaskId", "targetTaskId", "relation")
				if err != nil {
					return "", err
				}
				rel, err := client.LinkTasks(ctx, a["sourceTaskId"], a["targetTaskId"], a["relation"])
				if err != nil {
					return "", err
				}
				return marshal(rel)
			},
		},
		{
			Name:        "unlink_tasks",
			Description: "Remove a task relation by its id (from get_task_relations). Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{"id": strArg("relation id")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id")
				if err != nil {
					return "", err
				}
				if err := client.UnlinkTasks(ctx, a["id"]); err != nil {
					return "", err
				}
				return "unlinked", nil
			},
		},
		{
			Name:        "list_time_entries",
			Description: "List time logged on a task. An empty endTime means the timer is still running.",
			InputSchema: obj(map[string]any{"taskId": strArg("task id")}, "taskId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "taskId")
				if err != nil {
					return "", err
				}
				entries, err := client.ListTimeEntries(ctx, a["taskId"])
				if err != nil {
					return "", err
				}
				return marshal(entries)
			},
		},
		{
			Name:        "log_time",
			Description: "Record a time entry on a task. Omit endTime to start a running timer. Timestamps are ISO 8601. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"taskId":      strArg("task id"),
				"startTime":   strArg("ISO 8601 start, e.g. 2026-01-31T09:00:00Z"),
				"endTime":     strArg("ISO 8601 end; omit for a running timer"),
				"description": strArg("what the time was spent on"),
			}, "taskId", "startTime"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "taskId", "startTime")
				if err != nil {
					return "", err
				}
				e, err := client.LogTime(ctx, a["taskId"], a["startTime"], a["endTime"], a["description"])
				if err != nil {
					return "", err
				}
				return marshal(e)
			},
		},
		{
			Name:        "update_time_entry",
			Description: "Edit a time entry; startTime is required by the API. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"id":          strArg("time entry id"),
				"startTime":   strArg("ISO 8601 start"),
				"endTime":     strArg("ISO 8601 end"),
				"description": strArg("new description"),
			}, "id", "startTime"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "id", "startTime")
				if err != nil {
					return "", err
				}
				e, err := client.UpdateTimeEntry(ctx, a["id"], a["startTime"], a["endTime"], a["description"])
				if err != nil {
					return "", err
				}
				return marshal(e)
			},
		},
		{
			Name:        "get_task_activity",
			Description: "List a task's event history (status changes, comments, edits).",
			InputSchema: obj(map[string]any{"taskId": strArg("task id")}, "taskId"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				a, err := str(args, "taskId")
				if err != nil {
					return "", err
				}
				acts, err := client.GetTaskActivity(ctx, a["taskId"])
				if err != nil {
					return "", err
				}
				return marshal(acts)
			},
		},
		{
			Name:        "search",
			Description: "Workspace-wide search over tasks, projects, comments, and activities.",
			InputSchema: obj(map[string]any{
				"q":           strArg("search text"),
				"type":        strArg("all|tasks|projects|workspaces|comments|activities"),
				"projectId":   strArg("limit to one project; name, slug, or id"),
				"workspaceId": strArg("workspace name or id; default from config"),
				"limit":       map[string]any{"type": "integer", "description": "max results, up to 50"},
			}, "q"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Q           string `json:"q"`
					Type        string `json:"type"`
					ProjectID   string `json:"projectId"`
					WorkspaceID string `json:"workspaceId"`
					Limit       int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Q == "" {
					return "", fmt.Errorf("missing required argument: q")
				}
				raw, err := client.Search(ctx, a.Q, a.Type, a.ProjectID, a.WorkspaceID, a.Limit)
				if err != nil {
					return "", err
				}
				return string(raw), nil
			},
		},
		{
			Name:        "github_app_info",
			Description: "Report the GitHub App this Kaneo instance is configured with. Empty appName means the admin has not set one up yet.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				info, err := client.GitHubAppInfo(ctx)
				if err != nil {
					return "", err
				}
				return marshal(info)
			},
		},
		{
			Name:        "github_repositories",
			Description: "List GitHub repositories reachable through the installed GitHub App, for picking one to link to a project.",
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				repos, err := client.GitHubRepositories(ctx, a.ProjectID)
				if err != nil {
					return "", err
				}
				return marshal(repos)
			},
		},
		{
			Name:        "get_github_integration",
			Description: "Show which GitHub repository a project is linked to, or null.",
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				gi, err := client.GetGitHubIntegration(ctx, a.ProjectID)
				if err != nil {
					return "", err
				}
				return marshal(gi)
			},
		},
		{
			Name:        "github_verify",
			Description: "Check that the GitHub App is installed on owner/repo with the permissions Kaneo needs. Always answers with a result object describing any problems. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
				"repo":      strArg("owner/name, e.g. Quad4-Software/ai"),
			}, "repo"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "repo")
				if err != nil {
					return "", err
				}
				owner, repo, err := kaneo.SplitRepo(a["repo"])
				if err != nil {
					return "", err
				}
				raw, err := client.GitHubVerify(ctx, a["projectId"], owner, repo)
				if err != nil {
					return "", err
				}
				return string(raw), nil
			},
		},
		{
			Name:        "connect_github",
			Description: "Link a project to a GitHub repository for two-way sync. The instance's GitHub App must already be installed on the repo. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
				"repo":      strArg("owner/name, e.g. Quad4-Software/ai"),
			}, "repo"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				a, err := str(args, "repo")
				if err != nil {
					return "", err
				}
				owner, repo, err := kaneo.SplitRepo(a["repo"])
				if err != nil {
					return "", err
				}
				gi, err := client.ConnectGitHub(ctx, a["projectId"], owner, repo)
				if err != nil {
					return "", err
				}
				return marshal(gi)
			},
		},
		{
			Name:        "update_github_integration",
			Description: "Toggle a project's GitHub link: isActive pauses sync, commentTaskLinkOnGitHubIssue controls the back-link comment. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId":                    strArg("project name, slug, or id; default from config"),
				"isActive":                     boolArg("false pauses sync"),
				"commentTaskLinkOnGitHubIssue": boolArg("comment a task link on linked GitHub issues"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ProjectID   string `json:"projectId"`
					IsActive    *bool  `json:"isActive"`
					CommentLink *bool  `json:"commentTaskLinkOnGitHubIssue"`
				}
				if err := json.Unmarshal(args, &a); err != nil {
					return "", fmt.Errorf("invalid arguments: %v", err)
				}
				gi, err := client.UpdateGitHubIntegration(ctx, a.ProjectID, a.IsActive, a.CommentLink)
				if err != nil {
					return "", err
				}
				return marshal(gi)
			},
		},
		{
			Name:        "disconnect_github",
			Description: "Unlink a project from its GitHub repository. Existing tasks stay. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				if err := client.DisconnectGitHub(ctx, a.ProjectID); err != nil {
					return "", err
				}
				return "disconnected", nil
			},
		},
		{
			Name:        "import_github_issues",
			Description: "Import the linked repository's GitHub issues as tasks. Issues that already have a task are skipped. Write tool; needs an API key.",
			Write:       true,
			InputSchema: obj(map[string]any{
				"projectId": strArg("project name, slug, or id; default from config"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				if err := needKey(); err != nil {
					return "", err
				}
				var a struct {
					ProjectID string `json:"projectId"`
				}
				_ = json.Unmarshal(args, &a)
				raw, err := client.ImportGitHubIssues(ctx, a.ProjectID)
				if err != nil {
					return "", err
				}
				return string(raw), nil
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

// setup stores the API key in the OS keyring (secret-tool or pass) and
// writes connection settings to ~/.config/kaneo/config.json with
// owner-only permissions and no key material. It then lists the
// workspaces and projects the key can see and lets the user pick a
// default project. The key is read from the terminal with echo
// disabled where the platform supports it, so it never lands in shell
// history, logs, or tool output.
func setup() error {
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

	existing, _ := kaneo.LoadConfig()
	cfg := kaneo.Config{APIURL: kaneo.DefaultAPI}
	if existing != nil {
		cfg = *existing
		cfg.APIKey = ""
	}

	cfg.APIURL = ask("API URL", getenvOr("KANEO_API_URL", firstNonEmpty(cfg.APIURL, kaneo.DefaultAPI)))

	backend := kaneo.DetectBackend()
	if backend == kaneo.BackendNone {
		return fmt.Errorf("no secret backend found; install libsecret (secret-tool) or pass")
	}
	fmt.Fprintf(os.Stderr, "storing key via %s\n", backend)

	fmt.Fprint(os.Stderr, "API key (leave empty to keep existing): ")
	key, err := readSecret(reader)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	if key != "" {
		if err := kaneo.StoreKey(backend, cfg.APIURL, key); err != nil {
			return fmt.Errorf("store key: %w", err)
		}
		cfg.KeyBackend = backend
	} else if k, _ := kaneo.LoadKey(cfg.APIURL); k != "" {
		key = k
	} else {
		return fmt.Errorf("no key given and none stored; nothing to configure")
	}

	// autodetect workspaces and projects with the fresh key
	probe, err := kaneo.NewClient(kaneo.Config{APIURL: cfg.APIURL, APIKey: key})
	if err != nil {
		return err
	}
	ctx := context.Background()
	workspaces, err := probe.ListWorkspaces(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not list workspaces (%v); enter ids manually\n", err)
		cfg.WorkspaceID = ask("Workspace ID", cfg.WorkspaceID)
		cfg.DefaultProject = ask("Default project ID", cfg.DefaultProject)
		return writeConfig(&cfg)
	}

	var refs []kaneo.ProjectRef
	for _, w := range workspaces {
		projects, err := probe.ListProjects(ctx, w.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: workspace %s: %v\n", w.Name, err)
			continue
		}
		for _, pr := range projects {
			refs = append(refs, kaneo.ProjectRef{
				Name: pr.Name, Slug: pr.Slug,
				Workspace: w.Name, WorkspaceID: w.ID, ProjectID: pr.ID,
			})
		}
	}
	if len(refs) == 0 {
		fmt.Fprintln(os.Stderr, "no projects found for this key")
		return writeConfig(&cfg)
	}
	cfg.Projects = refs
	if cfg.WorkspaceID == "" && len(workspaces) == 1 {
		cfg.WorkspaceID = workspaces[0].ID
	}

	fmt.Fprintln(os.Stderr, "\nprojects:")
	for i, r := range refs {
		fmt.Fprintf(os.Stderr, "  %d) %s / %s\n", i+1, r.Workspace, r.Name)
	}
	cur := cfg.DefaultProject
	for i, r := range refs {
		if r.ProjectID == cur {
			cur = fmt.Sprintf("%d", i+1)
		}
	}
	choice := ask("default project number", cur)
	if idx, err := strconv.Atoi(choice); err == nil && idx >= 1 && idx <= len(refs) {
		cfg.DefaultProject = refs[idx-1].ProjectID
	} else if choice != "" {
		cfg.DefaultProject = cfg.ResolveProject(choice)
	}
	if cfg.DefaultProject == "" {
		cfg.DefaultProject = refs[0].ProjectID
	}

	return writeConfig(&cfg)
}

func writeConfig(cfg *kaneo.Config) error {
	p, err := kaneo.ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if err := kaneo.SaveConfig(p, cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (0600, no key material)\n", p)
	return nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
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
