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

func setupTimeMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/time_entries" && r.Method == "GET":
			json.NewEncoder(w).Encode(api.TimeEntryListResponse{
				TimeEntries: []api.TimeEntry{
					{
						ID: 1, Date: "2026-03-24", DurationMinutes: 120, Description: "Feature work",
						Project: &api.ProjectInfo{ID: 1, Name: "Project Alpha"},
					},
					{
						ID: 2, Date: "2026-03-23", DurationMinutes: 60, Description: "Bug fix",
						Project: &api.ProjectInfo{ID: 1, Name: "Project Alpha"},
					},
				},
			})

		case r.URL.Path == "/api/v1/time_entries" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.TimeEntry{
				ID: 42, Date: "2026-03-24", DurationMinutes: 120, Description: "New entry",
				Project: &api.ProjectInfo{ID: 1, Name: "Project Alpha"},
			})

		case strings.HasPrefix(r.URL.Path, "/api/v1/time_entries/") && r.Method == "DELETE":
			json.NewEncoder(w).Encode(map[string]string{"message": "Time entry deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestTimeListOutput(t *testing.T) {
	server := setupTimeMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "time", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("time list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Feature work") {
		t.Errorf("expected time entry in output: %s", output)
	}
	if !strings.Contains(output, "Project Alpha") {
		t.Errorf("expected project name in output: %s", output)
	}
	if !strings.Contains(output, "2h") {
		t.Errorf("expected formatted duration in output: %s", output)
	}
}

func TestTimeLog(t *testing.T) {
	server := setupTimeMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "time", "log", "--project", "1", "--duration", "2h", "--description", "Test work"})

	if err := root.Execute(); err != nil {
		t.Fatalf("time log failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Logged") {
		t.Errorf("expected confirmation: %s", output)
	}
}

func TestTimeLogRequiresProject(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: "http://localhost", Token: "test"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "time", "log", "--duration", "1h"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error without --project")
	}
	if !strings.Contains(err.Error(), "--project is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTimeDelete(t *testing.T) {
	server := setupTimeMockServer(t)
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{ServerURL: server.URL, Token: "test_token"}
	cfg.Save(cfgPath)

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "time", "delete", "1"})

	if err := root.Execute(); err != nil {
		t.Fatalf("time delete failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "deleted") {
		t.Errorf("expected deletion confirmation: %s", output)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"2h", 120, false},
		{"1h30m", 90, false},
		{"90m", 90, false},
		{"1.5h", 90, false},
		{"30m", 30, false},
		{"0h", 0, true},
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		got, err := parseDuration(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseDuration(%q) = %d, want error", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDuration(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDuration(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFormatMinutes(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{120, "2h"},
		{90, "1h 30m"},
		{30, "30m"},
		{60, "1h"},
		{150, "2h 30m"},
	}

	for _, tt := range tests {
		got := formatMinutes(tt.input)
		if got != tt.want {
			t.Errorf("formatMinutes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
