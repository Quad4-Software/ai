// SPDX-License-Identifier: 0BSD
// Project and column management. Most endpoints answer with bare
// objects on current builds; fetch tolerates {"data": ...} envelopes.
package kaneo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// decJSON unmarshals data into out, transparently unwrapping a
// {"data": ...} envelope when present.
func decJSON(data json.RawMessage, out any) error {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err == nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return json.Unmarshal(data, out)
}

// fetch performs one call and decodes the body, envelope-aware.
// out may be nil when the body is not needed.
func (c *Client) fetch(ctx context.Context, method, path string, q url.Values, payload, out any) error {
	var raw json.RawMessage
	if err := c.do(ctx, method, path, q, payload, &raw); err != nil {
		return err
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return decJSON(raw, out)
}

// IDPos pairs an object id with its new position for reorder calls.
type IDPos struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// GetProject fetches one project by id (or configured name/slug).
func (c *Client) GetProject(ctx context.Context, ref string) (*Project, error) {
	pid := c.ProjectID(ref)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	var p Project
	if err := c.fetch(ctx, http.MethodGet, "/project/"+url.PathEscape(pid), nil, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateProject creates a project in the workspace. slug is the task
// prefix; icon is a UI hint the web app understands. description is
// applied with a follow-up PUT because the create body has no field
// for it.
func (c *Client) CreateProject(ctx context.Context, name, slug, icon, description, workspaceID string) (*Project, error) {
	wid := c.WorkspaceID(workspaceID)
	if wid == "" {
		return nil, fmt.Errorf("missing workspaceId (arg or KANEO_WORKSPACE_ID)")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("missing required arguments: name, slug")
	}
	if icon == "" {
		icon = "folder"
	}
	var p Project
	payload := map[string]any{
		"name": name, "slug": strings.ToUpper(slug), "icon": icon, "workspaceId": wid,
	}
	if err := c.fetch(ctx, http.MethodPost, "/project", nil, payload, &p); err != nil {
		return nil, err
	}
	if description != "" {
		if _, err := c.UpdateProject(ctx, p.ID, ProjectPatch{Description: description}); err != nil {
			return &p, fmt.Errorf("project created but setting description failed: %w", err)
		}
		p.Description = description
	}
	return &p, nil
}

// ProjectPatch holds the updatable project fields. Empty strings mean
// "keep current"; the API replaces the full record so UpdateProject
// merges against the live object.
type ProjectPatch struct {
	Name        string
	Slug        string
	Icon        string
	Description string
	Public      *bool
}

// UpdateProject merges p over the live project and PUTs the result.
func (c *Client) UpdateProject(ctx context.Context, id string, p ProjectPatch) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	cur, err := c.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"name": cur.Name, "icon": cur.Icon, "slug": cur.Slug,
		"description": cur.Description, "isPublic": cur.IsPublic,
	}
	if p.Name != "" {
		body["name"] = p.Name
	}
	if p.Slug != "" {
		body["slug"] = strings.ToUpper(p.Slug)
	}
	if p.Icon != "" {
		body["icon"] = p.Icon
	}
	if p.Description != "" {
		body["description"] = p.Description
	}
	if p.Public != nil {
		body["isPublic"] = *p.Public
	}
	var out Project
	if err := c.fetch(ctx, http.MethodPut, "/project/"+url.PathEscape(id), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteProject permanently removes a project and everything in it.
// Destructive; callers should confirm first.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	return c.do(ctx, http.MethodDelete, "/project/"+url.PathEscape(id), nil, nil, nil)
}

// ArchiveProject hides (archive=true) or restores (archive=false) a
// project. Archiving keeps the data; deleting does not.
func (c *Client) ArchiveProject(ctx context.Context, id string, archive bool) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	verb := "archive"
	if !archive {
		verb = "unarchive"
	}
	return c.do(ctx, http.MethodPut, "/project/"+url.PathEscape(id)+"/"+verb, nil, nil, nil)
}

// ReorderProjects sets sidebar order. Positions express relative
// order; the server renumbers 0..n-1.
func (c *Client) ReorderProjects(ctx context.Context, order []IDPos) error {
	if len(order) == 0 {
		return fmt.Errorf("missing required argument: order")
	}
	return c.do(ctx, http.MethodPut, "/project/reorder", nil,
		map[string]any{"projects": order}, nil)
}

// ListColumns returns a project's board columns in order.
func (c *Client) ListColumns(ctx context.Context, projectID string) ([]Column, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	var cols []Column
	if err := c.fetch(ctx, http.MethodGet, "/column/"+url.PathEscape(pid), nil, nil, &cols); err != nil {
		return nil, err
	}
	return cols, nil
}

// CreateColumn adds a board column. isFinal marks the "done" column.
func (c *Client) CreateColumn(ctx context.Context, projectID, name, icon, color string, isFinal *bool) (*Column, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("missing required argument: name")
	}
	payload := map[string]any{"name": name}
	if icon != "" {
		payload["icon"] = icon
	}
	if color != "" {
		payload["color"] = color
	}
	if isFinal != nil {
		payload["isFinal"] = *isFinal
	}
	var col Column
	if err := c.fetch(ctx, http.MethodPost, "/column/"+url.PathEscape(pid), nil, payload, &col); err != nil {
		return nil, err
	}
	return &col, nil
}

// UpdateColumn patches a column's name, icon, color, or isFinal flag.
func (c *Client) UpdateColumn(ctx context.Context, id, name, icon, color string, isFinal *bool) (*Column, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}
	payload := map[string]any{}
	if name != "" {
		payload["name"] = name
	}
	if icon != "" {
		payload["icon"] = icon
	}
	if color != "" {
		payload["color"] = color
	}
	if isFinal != nil {
		payload["isFinal"] = *isFinal
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass name, icon, color, or isFinal")
	}
	var col Column
	if err := c.fetch(ctx, http.MethodPut, "/column/"+url.PathEscape(id), nil, payload, &col); err != nil {
		return nil, err
	}
	return &col, nil
}

// DeleteColumn removes a board column. Destructive when tasks live in
// it; callers should confirm first.
func (c *Client) DeleteColumn(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing required argument: id")
	}
	return c.do(ctx, http.MethodDelete, "/column/"+url.PathEscape(id), nil, nil, nil)
}

// ReorderColumns sets column order within a project.
func (c *Client) ReorderColumns(ctx context.Context, projectID string, order []IDPos) error {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	if len(order) == 0 {
		return fmt.Errorf("missing required argument: order")
	}
	return c.do(ctx, http.MethodPut, "/column/reorder/"+url.PathEscape(pid), nil,
		map[string]any{"columns": order}, nil)
}
