// SPDX-License-Identifier: 0BSD
package kaneo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateProject(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/project" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "Melovian" || body["slug"] != "MEL" || body["workspaceId"] != "w1" {
				t.Errorf("bad payload %v", body)
			}
			_, _ = w.Write([]byte(`{"id":"p9","name":"Melovian","slug":"MEL","icon":"folder"}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/project/p9" {
			_, _ = w.Write([]byte(`{"id":"p9","name":"Melovian","slug":"MEL","icon":"folder","isPublic":false}`))
			return
		}
		if r.Method == http.MethodPut && r.URL.Path == "/project/p9" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["description"] != "desc" || body["name"] != "Melovian" {
				t.Errorf("update lost fields: %v", body)
			}
			_, _ = w.Write([]byte(`{"id":"p9","name":"Melovian","slug":"MEL","description":"desc"}`))
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL, WorkspaceID: "w1"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.CreateProject(context.Background(), "Melovian", "mel", "", "desc", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "p9" || p.Description != "desc" {
		t.Fatalf("got %+v", p)
	}
	want := "POST /project,GET /project/p9,PUT /project/p9"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls %v, want %v", calls, want)
	}
}

func TestProjectLifecycle(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"p1"}`))
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.ArchiveProject(ctx, "p1", true); err != nil {
		t.Fatal(err)
	}
	if err := c.ArchiveProject(ctx, "p1", false); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	want := "PUT /project/p1/archive,PUT /project/p1/unarchive,DELETE /project/p1"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls %v, want %v", calls, want)
	}
}

func TestColumnsAndLabels(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/column/p1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"c1","slug":"to-do","name":"To Do"}]`))
		case r.URL.Path == "/label/l1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"l1","name":"bug","color":"#f00"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"x"}`))
		}
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cols, err := c.ListColumns(ctx, "p1")
	if err != nil || len(cols) != 1 {
		t.Fatalf("cols %v %v", cols, err)
	}
	fin := true
	if _, err := c.CreateColumn(ctx, "p1", "Done", "", "#0f0", &fin); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateLabel(ctx, "l1", "", "#00f"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteLabel(ctx, "l1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteColumn(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	want := "GET /column/p1,POST /column/p1,GET /label/l1,PUT /label/l1,DELETE /label/l1,DELETE /column/c1"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls %v, want %v", calls, want)
	}
}

func TestRelationsTimeSearch(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"r1"}`))
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL, WorkspaceID: "w1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := c.GetTaskRelations(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LinkTasks(ctx, "t1", "t2", "blocks"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LinkTasks(ctx, "t1", "t2", "bogus"); err == nil {
		t.Fatal("bad relation should error")
	}
	if _, err := c.ListTimeEntries(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetTaskActivity(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(ctx, "deploy", "", "", "", 10); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(calls, ",")
	for _, want := range []string{
		"GET /task-relation/t1?", "POST /task-relation?",
		"GET /time-entry/task/t1?", "GET /activity/t1?",
		"GET /search?limit=10&q=deploy&type=all&workspaceId=w1",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, calls)
		}
	}
}

func TestGitHubIntegration(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/app-info"):
			_, _ = w.Write([]byte(`{"appName":"kaneo-q4"}`))
		case strings.Contains(r.URL.Path, "/project/p1") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`null`))
		case strings.Contains(r.URL.Path, "/project/p1") && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"id":"g1","projectId":"p1","repositoryOwner":"Quad4-Software","repositoryName":"ai"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	c, err := NewClient(Config{APIURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	info, err := c.GitHubAppInfo(ctx)
	if err != nil || info.AppName != "kaneo-q4" {
		t.Fatalf("app info %v %v", info, err)
	}
	gi, err := c.GetGitHubIntegration(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if gi != nil {
		t.Fatalf("null integration should decode to nil, got %+v", gi)
	}
	gi, err = c.ConnectGitHub(ctx, "p1", "Quad4-Software", "ai")
	if err != nil || gi.RepositoryName != "ai" {
		t.Fatalf("connect %v %v", gi, err)
	}
	if err := c.DisconnectGitHub(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SplitRepo("ai"); err == nil {
		t.Fatal("SplitRepo should reject bare name")
	}
	if o, n, err := SplitRepo("Quad4-Software/ai"); err != nil || o != "Quad4-Software" || n != "ai" {
		t.Fatalf("SplitRepo %q %q %v", o, n, err)
	}
	want := "GET /github-integration/app-info," +
		"GET /github-integration/project/p1," +
		"POST /github-integration/project/p1," +
		"DELETE /github-integration/project/p1"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls %v, want %v", calls, want)
	}
}
