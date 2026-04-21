package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func newCommitCmd() *cobra.Command {
	commitCmd := &cobra.Command{
		Use:   "commit",
		Short: "Commit operations",
		Long:  "Report AI token cost data for commits.",
	}
	commitCmd.AddCommand(newCommitCostCmd())
	return commitCmd
}

func newCommitCostCmd() *cobra.Command {
	var (
		sha       string
		repoFlag  string
		sessionID string
		tokens    int
		cost      float64
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Report token cost for a commit",
		Long:  "Push token usage and estimated cost data for a commit to Guild.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sha == "" {
				detected, err := gitRevParseHead(".")
				if err != nil {
					return fmt.Errorf("--sha required (HEAD detection failed: %w)", err)
				}
				sha = detected
			}
			if repoFlag == "" {
				detected, err := gitDetectRepo(".")
				if err != nil {
					return fmt.Errorf("--repo required (remote detection failed: %w)", err)
				}
				repoFlag = detected
			}
			if tokens == 0 && cost == 0 {
				return fmt.Errorf("at least one of --tokens or --cost is required")
			}
			return reportCommitCost(Cfg.ServerURL, Cfg.Token, sha, repoFlag, sessionID, tokens, cost, cmd)
		},
	}

	cmd.Flags().StringVar(&sha, "sha", "", "commit SHA (default: HEAD)")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "repository in org/repo format (default: auto-detected from git remote)")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "Claude Code session ID for deduplication")
	cmd.Flags().IntVar(&tokens, "tokens", 0, "estimated token count")
	cmd.Flags().Float64Var(&cost, "cost", 0, "estimated cost in USD")

	return cmd
}

func reportCommitCost(serverURL, token, sha, repo, sessionID string, tokens int, cost float64, cmd *cobra.Command) error {
	apiURL := fmt.Sprintf("%s/api/v1/commits/%s/cost", serverURL, url.PathEscape(sha))

	form := url.Values{}
	form.Set("repo", repo)
	if sessionID != "" {
		form.Set("session_id", sessionID)
	}
	if tokens > 0 {
		form.Set("tokens", fmt.Sprintf("%d", tokens))
	}
	if cost > 0 {
		form.Set("cost", fmt.Sprintf("%.4f", cost))
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed (status %d): %s", resp.StatusCode, errResp.Error)
	}

	var result struct {
		SHA             string  `json:"sha"`
		Repo            string  `json:"repo"`
		EstimatedTokens int     `json:"estimated_tokens"`
		EstimatedCost   float64 `json:"estimated_cost_usd"`
		LinkedToPR      bool    `json:"linked_to_pr"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	out := cmd.OutOrStdout()
	short := sha
	if len(short) > 8 {
		short = short[:8]
	}
	fmt.Fprintf(out, "✓ Reported cost for %s (%s)\n", short, repo)
	if result.EstimatedTokens > 0 {
		fmt.Fprintf(out, "  Tokens: %d\n", result.EstimatedTokens)
	}
	if result.EstimatedCost > 0 {
		fmt.Fprintf(out, "  Cost:   $%.4f\n", result.EstimatedCost)
	}
	if result.LinkedToPR {
		fmt.Fprintf(out, "  Linked to PR: yes\n")
	} else {
		fmt.Fprintf(out, "  Linked to PR: pending (commit not yet on GitHub)\n")
	}
	return nil
}

// gitRevParseHead returns the full SHA of HEAD in the given directory.
func gitRevParseHead(dir string) (string, error) {
	c := exec.Command("git", "rev-parse", "HEAD")
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gitDetectRepo parses the origin remote URL and returns "org/repo".
func gitDetectRepo(dir string) (string, error) {
	c := exec.Command("git", "remote", "get-url", "origin")
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return parseGitRemote(strings.TrimSpace(string(out)))
}

var (
	httpsRemoteRe = regexp.MustCompile(`https?://[^/]+/([^/]+/[^/]+?)(?:\.git)?$`)
	sshRemoteRe   = regexp.MustCompile(`[^:]+:([^/]+/[^/]+?)(?:\.git)?$`)
)

func parseGitRemote(remote string) (string, error) {
	if m := httpsRemoteRe.FindStringSubmatch(remote); m != nil {
		return m[1], nil
	}
	if m := sshRemoteRe.FindStringSubmatch(remote); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("unrecognized remote format: %q", remote)
}
