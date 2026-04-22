package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smartcraftsoftware/guild-cli/internal/config"
)

func TestContextJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"id":    1,
				"name":  "Test User",
				"email": "test@example.com",
			},
			"assigned_issues": []map[string]interface{}{
				{
					"id":       1,
					"title":    "Fix auth",
					"priority": "high",
					"status":   "In Progress",
					"team_name": "Engineering",
				},
			},
		})
	}))
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "context", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("context --json failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if _, ok := result["user"]; !ok {
		t.Error("JSON output missing 'user' key")
	}
	if _, ok := result["assigned_issues"]; !ok {
		t.Error("JSON output missing 'assigned_issues' key")
	}
}

func TestContextPrettyOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "Test User",
				"email": "test@example.com",
			},
			"assigned_issues": []map[string]interface{}{
				{
					"title":    "Fix auth",
					"priority": "high",
					"status":   "In Progress",
					"team_name": "Engineering",
				},
			},
		})
	}))
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "context"})

	if err := root.Execute(); err != nil {
		t.Fatalf("context failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test User") {
		t.Errorf("expected user name: %s", output)
	}
	if !strings.Contains(output, "Fix auth") {
		t.Errorf("expected issue title: %s", output)
	}
}
