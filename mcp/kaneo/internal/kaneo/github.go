// SPDX-License-Identifier: 0BSD
// GitHub App integration: the instance-level App, per-project repo
// links, verification, and issue import. These endpoints need the
// workspace:manage_settings permission on most builds.
package kaneo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GitHubAppInfo reports the GitHub App the instance is configured
// with. AppName is empty when the server has no App set up, which
// means the per-project integration cannot work yet.
type GitHubAppInfo struct {
	AppName string `json:"appName"`
}

// GitHubAppInfo returns the instance's GitHub App slug, or empty.
func (c *Client) GitHubAppInfo(ctx context.Context) (*GitHubAppInfo, error) {
	var info GitHubAppInfo
	if err := c.fetch(ctx, http.MethodGet, "/github-integration/app-info", nil, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GitHubRepo is one repository reachable through the installed App.
type GitHubRepo struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	FullName       string `json:"full_name"`
	Private        bool   `json:"private"`
	Description    string `json:"description"`
	HTMLURL        string `json:"html_url"`
	InstallationID int    `json:"installation_id"`
}

// GitHubRepositories lists repos the installed App can reach, for
// picking one to link to the project.
func (c *Client) GitHubRepositories(ctx context.Context, projectID string) ([]GitHubRepo, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	var repos []GitHubRepo
	if err := c.fetch(ctx, http.MethodGet,
		"/github-integration/repositories/"+url.PathEscape(pid), nil, nil, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// GitHubVerify checks that the App is installed on owner/repo and
// holds the permissions Kaneo needs. The endpoint always answers 200;
// problems are described in the body.
func (c *Client) GitHubVerify(ctx context.Context, projectID, owner, repo string) (json.RawMessage, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("missing required arguments: owner, repo")
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, "/github-integration/verify", nil, map[string]any{
		"projectId": pid, "repositoryOwner": owner, "repositoryName": repo,
	}, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GitHubIntegration is a project's link to a GitHub repository.
type GitHubIntegration struct {
	ID                      string `json:"id"`
	ProjectID               string `json:"projectId"`
	RepositoryOwner         string `json:"repositoryOwner"`
	RepositoryName          string `json:"repositoryName"`
	InstallationID          *int   `json:"installationId"`
	BranchPattern           string `json:"branchPattern"`
	CommentTaskLinkOnGitHub bool   `json:"commentTaskLinkOnGitHubIssue"`
	IsActive                *bool  `json:"isActive"`
	CreatedAt               string `json:"createdAt"`
	UpdatedAt               string `json:"updatedAt"`
}

// GetGitHubIntegration returns the project's repo link, or nil when
// the project is not connected.
func (c *Client) GetGitHubIntegration(ctx context.Context, projectID string) (*GitHubIntegration, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	var gi *GitHubIntegration
	if err := c.fetch(ctx, http.MethodGet,
		"/github-integration/project/"+url.PathEscape(pid), nil, nil, &gi); err != nil {
		return nil, err
	}
	return gi, nil
}

// ConnectGitHub links a project to owner/repo. The App must already be
// installed on that repository.
func (c *Client) ConnectGitHub(ctx context.Context, projectID, owner, repo string) (*GitHubIntegration, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("missing required arguments: owner, repo")
	}
	var gi GitHubIntegration
	if err := c.fetch(ctx, http.MethodPost,
		"/github-integration/project/"+url.PathEscape(pid), nil,
		map[string]any{"repositoryOwner": owner, "repositoryName": repo}, &gi); err != nil {
		return nil, err
	}
	return &gi, nil
}

// UpdateGitHubIntegration toggles the link. Nil fields keep their
// current values.
func (c *Client) UpdateGitHubIntegration(ctx context.Context, projectID string, isActive, commentLink *bool) (*GitHubIntegration, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	payload := map[string]any{}
	if isActive != nil {
		payload["isActive"] = *isActive
	}
	if commentLink != nil {
		payload["commentTaskLinkOnGitHubIssue"] = *commentLink
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update; pass isActive or commentTaskLinkOnGitHubIssue")
	}
	var gi GitHubIntegration
	if err := c.fetch(ctx, http.MethodPatch,
		"/github-integration/project/"+url.PathEscape(pid), nil, payload, &gi); err != nil {
		return nil, err
	}
	return &gi, nil
}

// DisconnectGitHub removes the project's repo link.
func (c *Client) DisconnectGitHub(ctx context.Context, projectID string) error {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	return c.do(ctx, http.MethodDelete,
		"/github-integration/project/"+url.PathEscape(pid), nil, nil, nil)
}

// ImportGitHubIssues imports the linked repository's open issues as
// tasks. Issues that already have a task are skipped. Returns the raw
// import summary.
func (c *Client) ImportGitHubIssues(ctx context.Context, projectID string) (json.RawMessage, error) {
	pid := c.ProjectID(projectID)
	if pid == "" {
		return nil, fmt.Errorf("missing projectId (arg or KANEO_PROJECT_ID)")
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, "/github-integration/import-issues", nil,
		map[string]any{"projectId": pid}, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// SplitRepo splits "owner/name" into its parts.
func SplitRepo(full string) (owner, name string, err error) {
	parts := strings.SplitN(strings.Trim(full, "/"), "/", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be owner/name, got %q", full)
	}
	return parts[0], parts[1], nil
}
