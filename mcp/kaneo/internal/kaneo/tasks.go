// SPDX-License-Identifier: 0BSD
package kaneo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// Valid values Kaneo accepts. Inputs are validated before any request.
var (
	Priorities = []string{"no-priority", "low", "medium", "high", "urgent"}
	Statuses   = []string{"backlog", "to-do", "in-progress", "in-review", "done", "cancelled"}
)

func validChoice(name, v string, allowed []string) error {
	if !slices.Contains(allowed, v) {
		return fmt.Errorf("invalid %s %q; want one of: %s", name, v, strings.Join(allowed, ", "))
	}
	return nil
}

// Task is the subset of the Kaneo task object the tools surface.
type Task struct {
	ID          string   `json:"id"`
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Position    int      `json:"position"`
	StartDate   string   `json:"startDate"`
	DueDate     string   `json:"dueDate"`
	CreatedAt   string   `json:"createdAt"`
	ProjectID   string   `json:"projectId"`
	Assignee    string   `json:"assigneeName"`
	Labels      []string `json:"labels,omitempty"`
}

type rawTask struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Position    int    `json:"position"`
	StartDate   string `json:"startDate"`
	DueDate     string `json:"dueDate"`
	CreatedAt   string `json:"createdAt"`
	ProjectID   string `json:"projectId"`
	Assignee    string `json:"assigneeName"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func toTask(r rawTask) Task {
	t := Task{
		ID: r.ID, Number: r.Number, Title: r.Title, Description: r.Description,
		Status: r.Status, Priority: r.Priority, Position: r.Position,
		StartDate: r.StartDate, DueDate: r.DueDate, CreatedAt: r.CreatedAt,
		ProjectID: r.ProjectID, Assignee: r.Assignee,
	}
	for _, l := range r.Labels {
		t.Labels = append(t.Labels, l.Name)
	}
	return t
}

// Project is one Kaneo project.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
	WorkspaceID string `json:"workspaceId"`
}

// Column is one board column with its tasks.
type Column struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Tasks []Task `json:"-"`
}

type rawColumn struct {
	ID    string    `json:"id"`
	Slug  string    `json:"slug"`
	Name  string    `json:"name"`
	Tasks []rawTask `json:"tasks"`
}

// Board is a project plus its columns and tasks.
type Board struct {
	Project
	Columns []Column `json:"columns"`
}

// ListProjects returns projects in a workspace.
func (c *Client) ListProjects(ctx context.Context, workspaceID string) ([]Project, error) {
	wid := c.WorkspaceID(workspaceID)
	if wid == "" {
		return nil, fmt.Errorf("missing workspaceId (arg or KANEO_WORKSPACE_ID)")
	}
	var out struct {
		Data []Project `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, "/project", url.Values{"workspaceId": {wid}}, nil, &out)
	if err != nil {
		// some builds return a bare array
		var arr []Project
		if err2 := c.do(ctx, http.MethodGet, "/project", url.Values{"workspaceId": {wid}}, nil, &arr); err2 == nil {
			return arr, nil
		}
		return nil, err
	}
	return out.Data, nil
}

// GetBoard fetches one project with columns and tasks.
func (c *Client) GetBoard(ctx context.Context, projectID string) (*Board, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	var out struct {
		Data struct {
			Project
			Columns []rawColumn `json:"columns"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/task/tasks/"+url.PathEscape(pid), nil, nil, &out); err != nil {
		return nil, err
	}
	b := &Board{Project: out.Data.Project}
	for _, col := range out.Data.Columns {
		cc := Column{ID: col.ID, Slug: col.Slug, Name: col.Name}
		for _, rt := range col.Tasks {
			cc.Tasks = append(cc.Tasks, toTask(rt))
		}
		b.Columns = append(b.Columns, cc)
	}
	return b, nil
}

// GetTask fetches one task by id. Handles both envelope and bare
// response shapes.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/task/"+url.PathEscape(id), nil, nil, &raw); err != nil {
		return nil, err
	}
	var env struct {
		Data *rawTask `json:"data"`
	}
	var rt rawTask
	if err := json.Unmarshal(raw, &env); err == nil && env.Data != nil && env.Data.ID != "" {
		rt = *env.Data
	} else if err := json.Unmarshal(raw, &rt); err != nil || rt.ID == "" {
		return nil, fmt.Errorf("task %s: unexpected response shape", id)
	}
	t := toTask(rt)
	return &t, nil
}

// CreateTask opens a task. Empty status/priority fall back to
// to-do/no-priority.
func (c *Client) CreateTask(ctx context.Context, projectID, title, description, priority, status, dueDate string) (*Task, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("missing required argument: title")
	}
	if priority == "" {
		priority = "no-priority"
	}
	if status == "" {
		status = "to-do"
	}
	if err := validChoice("priority", priority, Priorities); err != nil {
		return nil, err
	}
	if err := validChoice("status", status, Statuses); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"title": title, "description": description,
		"priority": priority, "status": status,
	}
	if dueDate != "" {
		payload["dueDate"] = dueDate
	}
	var rt rawTask
	if err := c.do(ctx, http.MethodPost, "/task/"+url.PathEscape(pid), nil, payload, &rt); err != nil {
		return nil, err
	}
	t := toTask(rt)
	return &t, nil
}

// Patch is the set of task fields update_task can change. Empty fields
// are left untouched.
type Patch struct {
	Title       string
	Description string
	Priority    string
	Status      string
	DueDate     string
}

// UpdateTask applies the non-empty fields of p via the granular
// endpoints. Returns the updated task.
func (c *Client) UpdateTask(ctx context.Context, id string, p Patch) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	if p.Priority != "" {
		if err := validChoice("priority", p.Priority, Priorities); err != nil {
			return nil, err
		}
	}
	if p.Status != "" {
		if err := validChoice("status", p.Status, Statuses); err != nil {
			return nil, err
		}
	}
	type step struct {
		path    string
		payload map[string]any
	}
	var steps []step
	if p.Title != "" {
		steps = append(steps, step{"/task/title/" + id, map[string]any{"title": p.Title}})
	}
	if p.Description != "" {
		steps = append(steps, step{"/task/description/" + id, map[string]any{"description": p.Description}})
	}
	if p.Priority != "" {
		steps = append(steps, step{"/task/priority/" + id, map[string]any{"priority": p.Priority}})
	}
	if p.Status != "" {
		steps = append(steps, step{"/task/status/" + id, map[string]any{"status": p.Status}})
	}
	if p.DueDate != "" {
		steps = append(steps, step{"/task/due-date/" + id, map[string]any{"dueDate": p.DueDate}})
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("nothing to update; pass title, description, priority, status, or dueDate")
	}
	for _, s := range steps {
		if err := c.do(ctx, http.MethodPut, s.path, nil, s.payload, nil); err != nil {
			return nil, err
		}
	}
	return c.GetTask(ctx, id)
}

// MoveTask changes a task's column. To move across projects pass
// destinationProjectId as well.
func (c *Client) MoveTask(ctx context.Context, id, status, destProjectID string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	if destProjectID != "" {
		payload := map[string]any{"destinationProjectId": destProjectID}
		if status != "" {
			payload["destinationStatus"] = status
		}
		return c.do(ctx, http.MethodPut, "/task/move/"+url.PathEscape(id), nil, payload, nil)
	}
	if status == "" {
		return fmt.Errorf("missing required argument: status")
	}
	if err := validChoice("status", status, Statuses); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPut, "/task/status/"+url.PathEscape(id), nil,
		map[string]any{"status": status}, nil)
}

// DeleteTask removes a task. Destructive; callers should confirm first.
func (c *Client) DeleteTask(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	return c.do(ctx, http.MethodDelete, "/task/"+url.PathEscape(id), nil, nil, nil)
}

// Comment is one task comment.
type Comment struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// GetComments lists comments on a task.
func (c *Client) GetComments(ctx context.Context, taskID string) ([]Comment, error) {
	if taskID == "" {
		return nil, fmt.Errorf("missing required argument: taskId")
	}
	var out struct {
		Data []Comment `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/comment/"+url.PathEscape(taskID), nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// AddComment posts a comment on a task.
func (c *Client) AddComment(ctx context.Context, taskID, content string) error {
	if taskID == "" || strings.TrimSpace(content) == "" {
		return fmt.Errorf("missing required arguments: taskId, content")
	}
	return c.do(ctx, http.MethodPost, "/comment/"+url.PathEscape(taskID), nil,
		map[string]any{"content": content}, nil)
}

// Label is a workspace label.
type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ListLabels returns the workspace's labels.
func (c *Client) ListLabels(ctx context.Context, workspaceID string) ([]Label, error) {
	wid := c.WorkspaceID(workspaceID)
	if wid == "" {
		return nil, fmt.Errorf("missing workspaceId (arg or KANEO_WORKSPACE_ID)")
	}
	var out struct {
		Data []Label `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/label/workspace/"+url.PathEscape(wid), nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// SetLabel attaches or detaches a label to a task. The label may be an
// id or an exact name resolved against the workspace labels.
func (c *Client) SetLabel(ctx context.Context, taskID, label, workspaceID string, attach bool) error {
	if taskID == "" || label == "" {
		return fmt.Errorf("missing required arguments: taskId, label")
	}
	id := label
	if !strings.Contains(label, "-") || len(label) < 20 {
		labels, err := c.ListLabels(ctx, workspaceID)
		if err != nil {
			return err
		}
		id = ""
		for _, l := range labels {
			if strings.EqualFold(l.Name, label) {
				id = l.ID
				break
			}
		}
		if id == "" {
			return fmt.Errorf("no label named %q in workspace", label)
		}
	}
	method := http.MethodPut
	if !attach {
		method = http.MethodDelete
	}
	return c.do(ctx, method, "/label/"+url.PathEscape(id)+"/task", nil,
		map[string]any{"taskId": taskID}, nil)
}
