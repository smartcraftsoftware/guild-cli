package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smartcraftsoftware/guild-cli/internal/auth"
	"github.com/smartcraftsoftware/guild-cli/internal/config"
)

func TestAuthHelpShowsSubcommands(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	root.SetArgs([]string{"--config", cfgPath, "auth", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("auth --help failed: %v", err)
	}

	output := buf.String()
	for _, sub := range []string{"login", "status", "logout"} {
		if !strings.Contains(output, sub) {
			t.Errorf("auth help missing %q subcommand", sub)
		}
	}
}

func TestAuthStatusNotLoggedIn(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	root.SetArgs([]string{"--config", cfgPath, "auth", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Not logged in") {
		t.Errorf("expected 'Not logged in', got: %s", output)
	}
}

func TestAuthLogoutWhenNotLoggedIn(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	root.SetArgs([]string{"--config", cfgPath, "auth", "logout"})

	if err := root.Execute(); err != nil {
		t.Fatalf("auth logout failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Not logged in") {
		t.Errorf("expected 'Not logged in', got: %s", output)
	}
}

func TestAuthLogoutClearsToken(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")

	// Pre-set a token in config
	cfg := &config.Config{
		ServerURL: "http://localhost:3000",
		Token:     "guild_test_token_to_clear",
	}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "auth", "logout"})

	if err := root.Execute(); err != nil {
		t.Fatalf("auth logout failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Logged out") {
		t.Errorf("expected 'Logged out', got: %s", output)
	}

	// Verify token is cleared from config file
	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if reloaded.Token != "" {
		t.Errorf("token not cleared: %q", reloaded.Token)
	}
}

func TestAuthStatusWithToken(t *testing.T) {
	// Mock /api/v1/auth/me
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/me" {
			json.NewEncoder(w).Encode(auth.MeResponse{
				User: auth.UserResponse{
					ID:    1,
					Name:  "Test User",
					Email: "test@example.com",
					Teams: []auth.TeamResponse{{ID: 1, Name: "Engineering"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{
		ServerURL: server.URL,
		Token:     "guild_test_token",
	}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "auth", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test User") {
		t.Errorf("expected 'Test User' in output, got: %s", output)
	}
	if !strings.Contains(output, "Engineering") {
		t.Errorf("expected 'Engineering' team in output, got: %s", output)
	}
}
