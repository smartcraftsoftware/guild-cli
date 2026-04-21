package cmd

import (
	"bufio"
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
	ToolUseID    string `json:"tool_use_id"`
	CWD          string `json:"cwd"`
}

// claudeStopPayload is the JSON Claude Code sends to Stop hooks via stdin.
type claudeStopPayload struct {
	SessionID      string  `json:"session_id"`
	StopReason     string  `json:"stop_reason"`
	CostUSD        float64 `json:"cost_usd"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
	TranscriptPath string  `json:"transcript_path"`
	Usage          struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type sessionEntry struct {
	SHA       string    `json:"sha"`
	Repo      string    `json:"repo"`
	ToolUseID string    `json:"tool_use_id"`
	Timestamp time.Time `json:"timestamp"`
}

type sessionRecord struct {
	SessionID string         `json:"session_id"`
	Commits   []sessionEntry `json:"commits"`
}

// transcriptLine is a single line of a Claude Code JSONL transcript file.
type transcriptLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			ID    string `json:"id"`
			Input struct {
				Command string `json:"command"`
			} `json:"input"`
		} `json:"content"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			OutputTokens             int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
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
			json.NewDecoder(os.Stdin).Decode(&struct{}{}) //nolint:errcheck

			if Cfg != nil && Cfg.Token != "" {
				return nil
			}

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

			if err := appendSessionCommit(payload.SessionID, sha, repo, payload.ToolUseID); err != nil {
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

			costUSD := payload.TotalCostUSD
			if costUSD == 0 {
				costUSD = payload.CostUSD
			}

			// Attempt accurate per-commit attribution from the transcript.
			// Falls back to even split if the transcript is unavailable.
			transcriptPath := payload.TranscriptPath
			if transcriptPath == "" {
				transcriptPath, _ = findTranscriptPath(payload.SessionID)
			}

			tokenDeltas, totalTranscriptTokens := attributeTokensFromTranscript(transcriptPath, record.Commits)

			for i, entry := range record.Commits {
				commitTokens := tokenDeltas[i]
				commitCost := 0.0
				if totalTranscriptTokens > 0 {
					commitCost = costUSD * float64(commitTokens) / float64(totalTranscriptTokens)
				} else if len(record.Commits) > 0 {
					commitCost = costUSD / float64(len(record.Commits))
				}

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

// attributeTokensFromTranscript walks the JSONL transcript and returns the token
// delta for each commit based on where in the conversation the commit occurred.
// Returns (tokenDeltas, totalTokens). Falls back to even split on any error.
func attributeTokensFromTranscript(transcriptPath string, commits []sessionEntry) ([]int, int) {
	n := len(commits)
	evenSplit := func(total int) ([]int, int) {
		if n == 0 {
			return nil, 0
		}
		per := total / n
		deltas := make([]int, n)
		for i := range deltas {
			deltas[i] = per
		}
		return deltas, total
	}

	if transcriptPath == "" || n == 0 {
		return evenSplit(0)
	}

	f, err := os.Open(transcriptPath)
	if err != nil {
		return evenSplit(0)
	}
	defer f.Close()

	// Build a lookup from tool_use_id → commit index for O(1) matching.
	idToIdx := make(map[string]int, n)
	for i, c := range commits {
		if c.ToolUseID != "" {
			idToIdx[c.ToolUseID] = i
		}
	}

	// snapshots[i] = running token total at the moment commit i was made.
	snapshots := make([]int, n)
	var runningTokens int

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4 MB — transcript lines can be large

	for scanner.Scan() {
		var line transcriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "assistant" {
			continue
		}

		u := line.Message.Usage
		runningTokens += u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens + u.OutputTokens

		for _, block := range line.Message.Content {
			if block.Type != "tool_use" || block.Name != "Bash" {
				continue
			}
			if !isGitCommitCommand(block.Input.Command) {
				continue
			}
			if idx, ok := idToIdx[block.ID]; ok {
				snapshots[idx] = runningTokens
			}
		}
	}

	totalTokens := runningTokens
	if totalTokens == 0 {
		return evenSplit(0)
	}

	// Convert snapshots to deltas. Unknown commits (tool_use_id missing or not
	// found) receive the even-split share of whatever tokens are unaccounted for.
	deltas := make([]int, n)
	accounted := 0
	unknown := 0

	for i, snap := range snapshots {
		if snap == 0 {
			unknown++
			continue
		}
		prev := 0
		if i > 0 {
			for j := i - 1; j >= 0; j-- {
				if snapshots[j] != 0 {
					prev = snapshots[j]
					break
				}
			}
		}
		deltas[i] = snap - prev
		accounted += deltas[i]
	}

	// Distribute unaccounted tokens evenly among commits with unknown positions.
	if unknown > 0 {
		unaccounted := totalTokens - accounted
		perUnknown := unaccounted / unknown
		for i, snap := range snapshots {
			if snap == 0 {
				deltas[i] = perUnknown
			}
		}
	}

	return deltas, totalTokens
}

// findTranscriptPath searches ~/.claude/projects/ for a JSONL file matching sessionID.
func findTranscriptPath(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	matches, err := filepath.Glob(filepath.Join(home, ".claude", "projects", "*", sessionID+".jsonl"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("transcript not found for session %s", sessionID)
	}
	return matches[0], nil
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

func appendSessionCommit(sessionID, sha, repo, toolUseID string) error {
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
		ToolUseID: toolUseID,
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
