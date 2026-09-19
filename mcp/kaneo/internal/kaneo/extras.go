// SPDX-License-Identifier: 0BSD
// Comments, labels, task relations, time entries, activity, and
// workspace-wide search.
package kaneo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// UpdateComment edits a comment's body.
func (c *Client) UpdateComment(ctx context.Context, id, content string) error {
	if id == "" || strings.TrimSpace(content) == "" {
		return fmt.Errorf("missing required arguments: id, content")
	}
	return c.do(ctx, http.MethodPut, "/comment/"+url.PathEscape(id), nil,
		map[string]any{"content": content}, nil)
}

// DeleteComment removes a comment. Destructive; confirm first.
func (c *Client) DeleteComment(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	return c.do(ctx, http.MethodDelete, "/comment/"+url.PathEscape(id), nil, nil, nil)
}

// CreateLabel makes a workspace label. color is a hex string such as
// "#e11d48". taskId optionally attaches it on creation.
func (c *Client) CreateLabel(ctx context.Context, name, color, workspaceID, taskID string) (*Label, error) {
	wid := c.WorkspaceID(workspaceID)
	if wid == "" {
		return nil, fmt.Errorf("missing workspaceId (arg or KANEO_WORKSPACE_ID)")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(color) == "" {
		return nil, fmt.Errorf("missing required arguments: name, color")
	}
	payload := map[string]any{"name": name, "color": color, "workspaceId": wid}
	if taskID != "" {
		payload["taskId"] = taskID
	}
	var l Label
	if err := c.fetch(ctx, http.MethodPost, "/label", nil, payload, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// GetLabel fetches one label by id.
func (c *Client) GetLabel(ctx context.Context, id string) (*Label, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	var l Label
	if err := c.fetch(ctx, http.MethodGet, "/label/"+url.PathEscape(id), nil, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateLabel renames or recolors a label. The API replaces both
// fields, so an empty name or color keeps the current value.
func (c *Client) UpdateLabel(ctx context.Context, id, name, color string) (*Label, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	if name == "" && color == "" {
		return nil, fmt.Errorf("nothing to update; pass name or color")
	}
	cur, err := c.GetLabel(ctx, id)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = cur.Name
	}
	if color == "" {
		color = cur.Color
	}
	var l Label
	if err := c.fetch(ctx, http.MethodPut, "/label/"+url.PathEscape(id), nil,
		map[string]any{"name": name, "color": color}, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// DeleteLabel removes a workspace label. Destructive; confirm first.
func (c *Client) DeleteLabel(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	return c.do(ctx, http.MethodDelete, "/label/"+url.PathEscape(id), nil, nil, nil)
}

// TaskRelation links two tasks: subtask, blocks, or related.
type TaskRelation struct {
	ID           string `json:"id"`
	SourceTaskID string `json:"sourceTaskId"`
	TargetTaskID string `json:"targetTaskId"`
	RelationType string `json:"relationType"`
}

// RelationTypes are the link kinds Kaneo accepts.
var RelationTypes = []string{"subtask", "blocks", "related"}

// GetTaskRelations lists a task's relations.
func (c *Client) GetTaskRelations(ctx context.Context, taskID string) ([]TaskRelation, error) {
	if taskID == "" {
		return nil, fmt.Errorf("missing required argument: taskId")
	}
	var rels []TaskRelation
	if err := c.fetch(ctx, http.MethodGet, "/task-relation/"+url.PathEscape(taskID), nil, nil, &rels); err != nil {
		return nil, err
	}
	return rels, nil
}

// LinkTasks creates a relation between two tasks.
func (c *Client) LinkTasks(ctx context.Context, sourceID, targetID, relation string) (*TaskRelation, error) {
	if sourceID == "" || targetID == "" {
		return nil, fmt.Errorf("missing required arguments: sourceTaskId, targetTaskId")
	}
	if err := validChoice("relation", relation, RelationTypes); err != nil {
		return nil, err
	}
	var rel TaskRelation
	if err := c.fetch(ctx, http.MethodPost, "/task-relation", nil, map[string]any{
		"sourceTaskId": sourceID, "targetTaskId": targetID, "relationType": relation,
	}, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// UnlinkTasks removes a relation by its id.
func (c *Client) UnlinkTasks(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	return c.do(ctx, http.MethodDelete, "/task-relation/"+url.PathEscape(id), nil, nil, nil)
}

// TimeEntry is one logged work interval on a task. Empty EndTime
// means the timer is still running.
type TimeEntry struct {
	ID          string `json:"id"`
	TaskID      string `json:"taskId"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Description string `json:"description"`
}

// ListTimeEntries returns the time logged on a task.
func (c *Client) ListTimeEntries(ctx context.Context, taskID string) ([]TimeEntry, error) {
	if taskID == "" {
		return nil, fmt.Errorf("missing required argument: taskId")
	}
	var entries []TimeEntry
	if err := c.fetch(ctx, http.MethodGet, "/time-entry/task/"+url.PathEscape(taskID), nil, nil, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// LogTime records a time entry. endTime empty starts a running timer;
// pass ISO 8601 timestamps.
func (c *Client) LogTime(ctx context.Context, taskID, startTime, endTime, description string) (*TimeEntry, error) {
	if taskID == "" || strings.TrimSpace(startTime) == "" {
		return nil, fmt.Errorf("missing required arguments: taskId, startTime")
	}
	payload := map[string]any{"taskId": taskID, "startTime": startTime}
	if endTime != "" {
		payload["endTime"] = endTime
	}
	if description != "" {
		payload["description"] = description
	}
	var e TimeEntry
	if err := c.fetch(ctx, http.MethodPost, "/time-entry", nil, payload, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateTimeEntry edits a time entry. The API wants startTime always;
// endTime and description are optional.
func (c *Client) UpdateTimeEntry(ctx context.Context, id, startTime, endTime, description string) (*TimeEntry, error) {
	if id == "" || strings.TrimSpace(startTime) == "" {
		return nil, fmt.Errorf("missing required arguments: id, startTime")
	}
	payload := map[string]any{"startTime": startTime}
	if endTime != "" {
		payload["endTime"] = endTime
	}
	if description != "" {
		payload["description"] = description
	}
	var e TimeEntry
	if err := c.fetch(ctx, http.MethodPut, "/time-entry/"+url.PathEscape(id), nil, payload, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// Activity is one event in a task's history (status changes,
// comments, edits).
type Activity struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"taskId"`
	Type      string         `json:"type"`
	Message   string         `json:"message"`
	EventData map[string]any `json:"eventData"`
	CreatedAt string         `json:"createdAt"`
}

// GetTaskActivity returns a task's event history.
func (c *Client) GetTaskActivity(ctx context.Context, taskID string) ([]Activity, error) {
	if taskID == "" {
		return nil, fmt.Errorf("missing required argument: taskId")
	}
	var acts []Activity
	if err := c.fetch(ctx, http.MethodGet, "/activity/"+url.PathEscape(taskID), nil, nil, &acts); err != nil {
		return nil, err
	}
	return acts, nil
}

// SearchTypes narrows a workspace search.
var SearchTypes = []string{"all", "tasks", "projects", "workspaces", "comments", "activities"}

// Search runs the workspace-wide search and returns the raw result
// JSON: the shape varies by result type. typ is one of SearchTypes;
// limit caps results at 50.
func (c *Client) Search(ctx context.Context, query, typ, projectID, workspaceID string, limit int) (json.RawMessage, error) {
	wid := c.WorkspaceID(workspaceID)
	if wid == "" {
		return nil, fmt.Errorf("missing workspaceId (arg or KANEO_WORKSPACE_ID)")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("missing required argument: q")
	}
	if typ == "" {
		typ = "all"
	}
	if err := validChoice("type", typ, SearchTypes); err != nil {
		return nil, err
	}
	q := url.Values{"q": {query}, "type": {typ}, "workspaceId": {wid}}
	if pid := c.ProjectID(projectID); pid != "" {
		q.Set("projectId", pid)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/search", q, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
