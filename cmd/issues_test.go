package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smartcraftsoftware/guild-cli/internal/api"
	"github.com/smartcraftsoftware/guild-cli/internal/config"
)

func setupIssuesMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/teams/1/issues" && r.Method == "GET":
			json.NewEncoder(w).Encode(api.IssueListResponse{
				Issues: []api.Issue{
					{
						ID: 1, Title: "Fix login bug", IssueType: "bug", Priority: "high",
						WorkflowStatus: &api.WorkflowStatus{ID: 1, Name: "In Progress", Category: "in_progress"},
						Assignee:       &api.UserInfo{ID: 1, Name: "Alice"},
					},
					{
						ID: 2, Title: "Add dark mode", IssueType: "story", Priority: "medium",
						WorkflowStatus: &api.WorkflowStatus{ID: 2, Name: "Backlog", Category: "todo"},
					},
				},
			})

		case r.URL.Path == "/api/v1/teams/1/issues" && r.Method == "POST":
			var body struct {
				Issue api.CreateIssueParams `json:"issue"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Issue{
				ID:        42,
				Title:     body.Issue.Title,
				IssueType: body.Issue.IssueType,
				Priority:  body.Issue.Priority,
			})

		case strings.HasPrefix(r.URL.Path, "/api/v1/teams/1/issues/") && r.Method == "GET":
			json.NewEncoder(w).Encode(api.Issue{
				ID: 1, Title: "Fix login bug", Description: "Users can't log in", IssueType: "bug", Priority: "high",
				WorkflowStatus: &api.WorkflowStatus{ID: 1, Name: "In Progress", Category: "in_progress"},
				Assignee:       &api.UserInfo{ID: 1, Name: "Alice"},
				CreatedBy:      &api.UserInfo{ID: 2, Name: "Bob"},
				CreatedAt:      "2026-03-24T10:00:00Z",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestIssuesListOutput(t *testing.T) {
	server := setupIssuesMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token", TeamID: "1"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "issues", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issues list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Fix login bug") {
		t.Errorf("expected issue title in output: %s", output)
	}
	if !strings.Contains(output, "Add dark mode") {
		t.Errorf("expected second issue in output: %s", output)
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected assignee in output: %s", output)
	}
}

func TestIssuesListRequiresTeam(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: "http://localhost", Token: "test_token"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "issues", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no team ID")
	}
	if !strings.Contains(err.Error(), "team ID required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIssuesCreate(t *testing.T) {
	server := setupIssuesMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token", TeamID: "1"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "issues", "create", "--title", "New bug", "--type", "bug"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issues create failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Created issue #42") {
		t.Errorf("expected creation confirmation: %s", output)
	}
}

func TestIssuesCreateRequiresTitle(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: "http://localhost", Token: "test_token", TeamID: "1"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "issues", "create"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no title")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIssuesView(t *testing.T) {
	server := setupIssuesMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token", TeamID: "1"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "issues", "view", "1"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issues view failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Fix login bug") {
		t.Errorf("expected issue title in detail: %s", output)
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected assignee in detail: %s", output)
	}
	if !strings.Contains(output, "Users can't log in") {
		t.Errorf("expected description in detail: %s", output)
	}
}

func TestIssuesTeamFlagOverridesConfig(t *testing.T) {
	server := setupIssuesMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token", TeamID: "99"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	// Use --team=1 which the mock server handles; config has team 99
	root.SetArgs([]string{"--config", cfgPath, "issues", "list", "--team", "1"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issues list with --team failed: %v", err)
	}

	// Should succeed because --team=1 is handled by mock
	output := buf.String()
	if !strings.Contains(output, "Fix login bug") {
		t.Errorf("expected issue from team 1: %s", output)
	}
}
