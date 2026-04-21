package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/smartcraftsoftware/guild-cli/internal/config"
	"github.com/spf13/cobra"
)

// claudePostToolUsePayload is the JSON Claude Code sends to PostToolUse hooks via stdin.
type claudePostToolUsePayload struct {
	SessionID string `json:"session_id"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
	ToolResponse string `json:"tool_response"`
	CWD          string `json:"cwd"`
}

// claudeStopPayload is the JSON Claude Code sends to Stop hooks via stdin.
type claudeStopPayload struct {
	SessionID    string  `json:"session_id"`
	StopReason   string  `json:"stop_reason"`
	CostUSD      float64 `json:"cost_usd"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type sessionEntry struct {
	SHA       string    `json:"sha"`
	Repo      string    `json:"repo"`
	Timestamp time.Time `json:"timestamp"`
}

type sessionRecord struct {
	SessionID string         `json:"session_id"`
	Commits   []sessionEntry `json:"commits"`
}

func newHookCmd() *cobra.Command {
	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Claude Code hook handlers (invoked automatically, not for direct use)",
		// Override root's PersistentPreRunE — hooks must never exit non-zero.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := config.DefaultPath()
			if err != nil {
				return nil
			}
			cfg, _ := config.Load(cfgPath)
			if cfg != nil {
				Cfg = cfg
			}
			return nil
		},
	}

	hookCmd.AddCommand(newHookSessionStartCmd())
	hookCmd.AddCommand(newHookPostToolUseCmd())
	hookCmd.AddCommand(newHookStopCmd())
	return hookCmd
}

func newHookSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Prompt unauthenticated users to run guild auth login",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Drain stdin — Claude Code sends a JSON payload we don't need here.
			json.NewDecoder(os.Stdin).Decode(&struct{}{}) //nolint:errcheck

			if Cfg != nil && Cfg.Token != "" {
				return nil // already logged in, nothing to do
			}

			// Stdout is injected into the conversation context by Claude Code.
			fmt.Fprintln(cmd.OutOrStdout(),
				"[Guild] You are not logged in to Guild. "+
					"Run `guild auth login` in your terminal to enable automatic AI cost tracking for your team.")
			return nil
		},
	}
}

func newHookPostToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "post-tool-use",
		Short: "Capture git commits made during a Claude Code session",
		RunE: func(cmd *cobra.Command, args []string) error {
			var payload claudePostToolUsePayload
			if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
				return nil
			}

			if payload.ToolName != "Bash" || !isGitCommitCommand(payload.ToolInput.Command) {
				return nil
			}
			if payload.SessionID == "" {
				return nil
			}

			cwd := payload.CWD
			if cwd == "" {
				cwd = "."
			}

			sha, err := gitRevParseHead(cwd)
			if err != nil || sha == "" {
				return nil
			}
			repo, err := gitDetectRepo(cwd)
			if err != nil || repo == "" {
				return nil
			}

			if err := appendSessionCommit(payload.SessionID, sha, repo); err != nil {
				fmt.Fprintf(os.Stderr, "guild hook: store commit: %v\n", err)
			}
			return nil
		},
	}
}

func newHookStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Report per-commit AI token cost to Guild at session end",
		RunE: func(cmd *cobra.Command, args []string) error {
			var payload claudeStopPayload
			if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
				return nil
			}
			if payload.SessionID == "" {
				return nil
			}

			record, err := loadSessionRecord(payload.SessionID)
			if err != nil || len(record.Commits) == 0 {
				return nil
			}
			defer deleteSessionRecord(payload.SessionID)

			if Cfg == nil || Cfg.Token == "" {
				return nil
			}

			// Prefer total_cost_usd, fall back to cost_usd
			costUSD := payload.TotalCostUSD
			if costUSD == 0 {
				costUSD = payload.CostUSD
			}
			totalTokens := payload.Usage.InputTokens + payload.Usage.OutputTokens

			// Distribute cost evenly across commits in this session
			n := len(record.Commits)
			commitTokens := totalTokens / n
			commitCost := costUSD / float64(n)

			for _, entry := range record.Commits {
				if err := reportCommitCost(Cfg.ServerURL, Cfg.Token, entry.SHA, entry.Repo, payload.SessionID, commitTokens, commitCost, cmd); err != nil {
					short := entry.SHA
					if len(short) > 8 {
						short = short[:8]
					}
					fmt.Fprintf(os.Stderr, "guild hook: report %s: %v\n", short, err)
				}
			}
			return nil
		},
	}
}

var gitCommitRe = regexp.MustCompile(`\bgit\s+commit\b`)

func isGitCommitCommand(command string) bool {
	return gitCommitRe.MatchString(command)
}

func sessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".guild", "sessions"), nil
}

var unsafeCharsRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func sessionPath(sessionID string) (string, error) {
	dir, err := sessionDir()
	if err != nil {
		return "", err
	}
	safe := unsafeCharsRe.ReplaceAllString(sessionID, "_")
	return filepath.Join(dir, safe+".json"), nil
}

func appendSessionCommit(sessionID, sha, repo string) error {
	path, err := sessionPath(sessionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	record := sessionRecord{SessionID: sessionID}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &record) //nolint:errcheck
	}

	for _, c := range record.Commits {
		if c.SHA == sha {
			return nil // already recorded
		}
	}
	record.Commits = append(record.Commits, sessionEntry{
		SHA:       sha,
		Repo:      repo,
		Timestamp: time.Now().UTC(),
	})

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadSessionRecord(sessionID string) (*sessionRecord, error) {
	path, err := sessionPath(sessionID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record sessionRecord
	return &record, json.Unmarshal(data, &record)
}

func deleteSessionRecord(sessionID string) {
	if path, err := sessionPath(sessionID); err == nil {
		os.Remove(path)
	}
}
