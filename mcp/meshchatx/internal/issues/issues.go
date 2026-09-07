// SPDX-License-Identifier: 0BSD
// Package issues implements a minimal GitHub issues client and the
// MeshChatX issue templates (bug report, feature request) so agents can
// file well-formed issues without hand-writing markdown. Stdlib only.
//
// Write operations are opt-in: the client only exists when a token is
// present in the environment. The token is never returned to callers.
package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DefaultRepo is the issue tracker these tools target.
const DefaultRepo = "Quad4-Software/MeshChatX"

const apiBase = "https://api.github.com"

var repoRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Client talks to the GitHub REST API for one repository.
type Client struct {
	// Repo is "owner/name", validated against repoRe.
	Repo string
	// API is the base URL, apiBase unless overridden in tests.
	API string
	// HTTP is the client used for requests.
	HTTP *http.Client

	token string
}

// NewClientFromEnv builds a Client from the environment. Token lookup
// order: MESHCHATX_GITHUB_TOKEN, GITHUB_TOKEN, GH_TOKEN. The target repo
// can be overridden with MESHCHATX_ISSUES_REPO.
func NewClientFromEnv() (*Client, error) {
	token := ""
	for _, k := range []string{"MESHCHATX_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(getenv(k)); v != "" {
			token = v
			break
		}
	}
	if token == "" {
		return nil, fmt.Errorf("no GitHub token; set MESHCHATX_GITHUB_TOKEN, GITHUB_TOKEN, or GH_TOKEN")
	}
	repo := getenv("MESHCHATX_ISSUES_REPO")
	if repo == "" {
		repo = DefaultRepo
	}
	return New(repo, token, apiBase), nil
}

// New builds a Client for repo. An empty api defaults to apiBase.
func New(repo, token, api string) *Client {
	if api == "" {
		api = apiBase
	}
	return &Client{
		Repo:  repo,
		API:   strings.TrimRight(api, "/"),
		HTTP:  &http.Client{Timeout: 15 * time.Second},
		token: token,
	}
}

func (c *Client) validRepo() error {
	if !repoRe.MatchString(c.Repo) {
		return fmt.Errorf("invalid repo %q; want owner/name", c.Repo)
	}
	return nil
}

// do performs one API call. payload may be nil for GET. When out is
// non-nil the response body is decoded into it.
func (c *Client) do(ctx context.Context, method, path string, payload, out any) error {
	if err := c.validRepo(); err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.API+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "meshchatx")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &e)
		if e.Message == "" {
			e.Message = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("github api %s %s: %d %s", method, path, resp.StatusCode, e.Message)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// Issue is the subset of the GitHub issue object these tools surface.
type Issue struct {
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	State   string   `json:"state"`
	HTMLURL string   `json:"html_url"`
	Body    string   `json:"body"`
	Labels  []string `json:"-"`
	User    string   `json:"-"`
	Created string   `json:"created_at"`
}

type rawIssue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Created string `json:"created_at"`
	User    struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func toIssue(r rawIssue) Issue {
	it := Issue{
		Number: r.Number, Title: r.Title, State: r.State,
		HTMLURL: r.HTMLURL, Body: r.Body, User: r.User.Login,
		Created: r.Created,
	}
	for _, l := range r.Labels {
		it.Labels = append(it.Labels, l.Name)
	}
	return it
}

// Create opens a new issue and returns it.
func (c *Client) Create(ctx context.Context, title, body string, labels []string) (*Issue, error) {
	payload := map[string]any{"title": title, "body": body}
	if len(labels) > 0 {
		payload["labels"] = labels
	}
	var r rawIssue
	if err := c.do(ctx, http.MethodPost, "/repos/"+c.Repo+"/issues", payload, &r); err != nil {
		return nil, err
	}
	it := toIssue(r)
	return &it, nil
}

// Patch is the set of fields issue_update can change. Nil fields are
// left untouched.
type Patch struct {
	Title       *string
	Body        *string
	Labels      *[]string
	State       *string // open or closed
	StateReason *string // completed, not_planned, reopened
}

// Update edits an issue. Use State closed/open to close or reopen.
func (c *Client) Update(ctx context.Context, number int, p Patch) (*Issue, error) {
	payload := map[string]any{}
	if p.Title != nil {
		payload["title"] = *p.Title
	}
	if p.Body != nil {
		payload["body"] = *p.Body
	}
	if p.Labels != nil {
		payload["labels"] = *p.Labels
	}
	if p.State != nil {
		payload["state"] = *p.State
	}
	if p.StateReason != nil {
		payload["state_reason"] = *p.StateReason
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update")
	}
	var r rawIssue
	path := fmt.Sprintf("/repos/%s/issues/%d", c.Repo, number)
	if err := c.do(ctx, http.MethodPatch, path, payload, &r); err != nil {
		return nil, err
	}
	it := toIssue(r)
	return &it, nil
}

// Get fetches one issue by number.
func (c *Client) Get(ctx context.Context, number int) (*Issue, error) {
	var r rawIssue
	path := fmt.Sprintf("/repos/%s/issues/%d", c.Repo, number)
	if err := c.do(ctx, http.MethodGet, path, nil, &r); err != nil {
		return nil, err
	}
	it := toIssue(r)
	return &it, nil
}

// Comment adds a comment to an issue.
func (c *Client) Comment(ctx context.Context, number int, body string) (string, error) {
	var r struct {
		HTMLURL string `json:"html_url"`
	}
	path := fmt.Sprintf("/repos/%s/issues/%d/comments", c.Repo, number)
	if err := c.do(ctx, http.MethodPost, path, map[string]any{"body": body}, &r); err != nil {
		return "", err
	}
	return r.HTMLURL, nil
}

// SearchHit is one issue_search result.
type SearchHit struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"html_url"`
}

// Search looks for existing issues, for duplicate checks before filing.
// state may be open, closed, or empty for any.
func (c *Client) Search(ctx context.Context, query, state string, limit int) ([]SearchHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	q := "repo:" + c.Repo + " is:issue " + query
	if state == "open" || state == "closed" {
		q += " state:" + state
	}
	path := "/search/issues?q=" + url.QueryEscape(q) +
		fmt.Sprintf("&per_page=%d", limit)
	var r struct {
		Items []struct {
			Number  int    `json:"number"`
			Title   string `json:"title"`
			State   string `json:"state"`
			HTMLURL string `json:"html_url"`
		} `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &r); err != nil {
		return nil, err
	}
	hits := make([]SearchHit, 0, len(r.Items))
	for _, it := range r.Items {
		hits = append(hits, SearchHit{it.Number, it.Title, it.State, it.HTMLURL})
	}
	return hits, nil
}
