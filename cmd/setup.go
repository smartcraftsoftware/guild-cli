package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure integrations with external tools",
	}
	setupCmd.AddCommand(newSetupClaudeCmd())
	return setupCmd
}

func newSetupClaudeCmd() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Install Guild hooks into Claude Code settings",
		Long: `Installs PostToolUse and Stop hooks into Claude Code's settings.json.

Writes to the global settings (~/.claude/settings.json) by default.
Use --local to write to the project-level settings (.claude/settings.json).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			settingsPath, err := claudeSettingsPath(local)
			if err != nil {
				return err
			}
			if err := installClaudeHooks(settingsPath); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "✓ Guild hooks installed in %s\n", settingsPath)
			fmt.Fprintln(out, "  SessionStart → guild hook session-start  (prompts login if not authenticated)")
			fmt.Fprintln(out, "  PostToolUse  → guild hook post-tool-use  (captures git commits)")
			fmt.Fprintln(out, "  Stop         → guild hook stop           (reports cost to Guild)")
			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "write to .claude/settings.json in current directory instead of ~/.claude/settings.json")
	return cmd
}

func claudeSettingsPath(local bool) (string, error) {
	if local {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
		return filepath.Join(cwd, ".claude", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookEntry struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

func installClaudeHooks(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Load existing settings as a raw map to preserve unknown fields.
	raw := make(map[string]json.RawMessage)
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	hooksMap := make(map[string][]hookEntry)
	if existing, ok := raw["hooks"]; ok {
		json.Unmarshal(existing, &hooksMap) //nolint:errcheck
	}

	hooksMap["SessionStart"] = mergeHookEntry(hooksMap["SessionStart"], hookEntry{
		Hooks: []hookCommand{{Type: "command", Command: "guild hook session-start"}},
	})
	hooksMap["PostToolUse"] = mergeHookEntry(hooksMap["PostToolUse"], hookEntry{
		Matcher: "Bash",
		Hooks:   []hookCommand{{Type: "command", Command: "guild hook post-tool-use"}},
	})
	hooksMap["Stop"] = mergeHookEntry(hooksMap["Stop"], hookEntry{
		Hooks: []hookCommand{{Type: "command", Command: "guild hook stop"}},
	})

	hooksJSON, err := json.Marshal(hooksMap)
	if err != nil {
		return fmt.Errorf("serializing hooks: %w", err)
	}
	raw["hooks"] = json.RawMessage(hooksJSON)

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing settings: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// mergeHookEntry appends entry to entries only if its command isn't already present.
func mergeHookEntry(entries []hookEntry, entry hookEntry) []hookEntry {
	for _, existing := range entries {
		for _, ec := range existing.Hooks {
			for _, nc := range entry.Hooks {
				if ec.Command == nc.Command {
					return entries
				}
			}
		}
	}
	return append(entries, entry)
}
