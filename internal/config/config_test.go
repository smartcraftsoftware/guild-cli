package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_DefaultsWhenNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}

	if cfg.ServerURL != DefaultServerURL {
		t.Errorf("ServerURL = %q, want %q", cfg.ServerURL, DefaultServerURL)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if cfg.TeamID != "" {
		t.Errorf("TeamID = %q, want empty", cfg.TeamID)
	}
}

func TestSave_ThenLoad_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	original := &Config{
		ServerURL: "http://guild.example.com",
		Token:     "guild_abc123",
		TeamID:    "team-42",
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ServerURL != original.ServerURL {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, original.ServerURL)
	}
	if loaded.Token != original.Token {
		t.Errorf("Token = %q, want %q", loaded.Token, original.Token)
	}
	if loaded.TeamID != original.TeamID {
		t.Errorf("TeamID = %q, want %q", loaded.TeamID, original.TeamID)
	}
}

func TestSave_CreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "nested", "config.yaml")

	cfg := &Config{ServerURL: "http://test.local"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed to create parent dirs: %v", err)
	}

	// Verify the file was actually written
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Config file not found after Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}
	if loaded.ServerURL != "http://test.local" {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, "http://test.local")
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}

	if !strings.HasSuffix(path, filepath.Join(".guild", "config.yaml")) {
		t.Errorf("DefaultPath = %q, want suffix .guild/config.yaml", path)
	}
}

func TestLoad_DefaultServerURL_WhenEmptyInFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	// Write a config with empty server_url
	if err := os.WriteFile(path, []byte("server_url: \"\"\nteam_id: myteam\n"), 0o644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.ServerURL != DefaultServerURL {
		t.Errorf("ServerURL = %q, want default %q when file has empty value", cfg.ServerURL, DefaultServerURL)
	}
	if cfg.TeamID != "myteam" {
		t.Errorf("TeamID = %q, want %q", cfg.TeamID, "myteam")
	}
}
