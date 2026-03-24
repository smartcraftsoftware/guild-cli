package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "guild") {
		t.Errorf("help output missing 'guild': %s", output)
	}
	if !strings.Contains(output, "version") {
		t.Errorf("help output missing 'version' subcommand: %s", output)
	}
	if !strings.Contains(output, "config") {
		t.Errorf("help output missing 'config' subcommand: %s", output)
	}
}

func TestVersionOutput(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	// Use a temp config path so we don't touch real user config
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	root.SetArgs([]string{"--config", cfgPath, "version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "guild version") {
		t.Errorf("version output missing 'guild version': %s", output)
	}
}

func TestConfigSetAndGet(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")

	// Set server
	{
		root := NewRootCmd()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetErr(buf)
		root.SetArgs([]string{"--config", cfgPath, "config", "set", "server", "http://test.example.com"})

		if err := root.Execute(); err != nil {
			t.Fatalf("config set failed: %v", err)
		}
	}

	// Get server
	{
		root := NewRootCmd()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetErr(buf)
		root.SetArgs([]string{"--config", cfgPath, "config", "get", "server"})

		if err := root.Execute(); err != nil {
			t.Fatalf("config get failed: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if output != "http://test.example.com" {
			t.Errorf("config get server = %q, want %q", output, "http://test.example.com")
		}
	}
}

func TestConfigSetAndGetTeam(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")

	// Set team
	{
		root := NewRootCmd()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetErr(buf)
		root.SetArgs([]string{"--config", cfgPath, "config", "set", "team", "engineering"})

		if err := root.Execute(); err != nil {
			t.Fatalf("config set team failed: %v", err)
		}
	}

	// Get team
	{
		root := NewRootCmd()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetErr(buf)
		root.SetArgs([]string{"--config", cfgPath, "config", "get", "team"})

		if err := root.Execute(); err != nil {
			t.Fatalf("config get team failed: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if output != "engineering" {
			t.Errorf("config get team = %q, want %q", output, "engineering")
		}
	}
}

func TestConfigGetUnknownKey(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	root.SetArgs([]string{"--config", cfgPath, "config", "get", "nonexistent"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown config key, got nil")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error = %q, want to contain 'unknown config key'", err.Error())
	}
}

func TestServerFlagOverride(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "--server", "http://override.example.com", "version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version with --server flag failed: %v", err)
	}

	// After execution, the global Cfg should have the override
	if Cfg == nil {
		t.Fatal("Cfg is nil after execution")
	}
	if Cfg.ServerURL != "http://override.example.com" {
		t.Errorf("Cfg.ServerURL = %q, want %q", Cfg.ServerURL, "http://override.example.com")
	}
}
