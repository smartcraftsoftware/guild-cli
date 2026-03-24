package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newPRCmd() *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Pull request operations",
		Long:  "Report token cost data for pull requests.",
	}

	prCmd.AddCommand(newPRCostCmd())

	return prCmd
}

func newPRCostCmd() *cobra.Command {
	var (
		repo   string
		prNum  int
		tokens int
		cost   float64
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Report token cost for a PR",
		Long:  "Push token usage and estimated cost data to a PR in Guild.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				return fmt.Errorf("--repo is required (format: org/repo)")
			}
			if prNum == 0 {
				return fmt.Errorf("--pr is required")
			}
			if tokens == 0 && cost == 0 {
				return fmt.Errorf("at least one of --tokens or --cost is required")
			}

			parts := strings.SplitN(repo, "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("--repo must be in org/repo format, got %q", repo)
			}

			apiURL := fmt.Sprintf("%s/api/v1/prs/%s/%s/%d/cost",
				Cfg.ServerURL,
				url.PathEscape(parts[0]),
				url.PathEscape(parts[1]),
				prNum,
			)

			form := url.Values{}
			if tokens > 0 {
				form.Set("tokens", strconv.Itoa(tokens))
			}
			if cost > 0 {
				form.Set("cost", fmt.Sprintf("%.4f", cost))
			}

			req, err := http.NewRequest("PATCH", apiURL, strings.NewReader(form.Encode()))
			if err != nil {
				return fmt.Errorf("creating request: %w", err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Authorization", "Bearer "+Cfg.Token)
			req.Header.Set("Accept", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("sending request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("PR not found: %s#%d", repo, prNum)
			}

			if resp.StatusCode != http.StatusOK {
				var errResp struct{ Error string `json:"error"` }
				json.NewDecoder(resp.Body).Decode(&errResp)
				return fmt.Errorf("failed (status %d): %s", resp.StatusCode, errResp.Error)
			}

			var result struct {
				Repo            string  `json:"repo"`
				PRNumber        int     `json:"pr_number"`
				Title           string  `json:"title"`
				EstimatedTokens int     `json:"estimated_tokens"`
				EstimatedCost   float64 `json:"estimated_cost_usd"`
			}
			json.NewDecoder(resp.Body).Decode(&result)

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "✓ Updated cost for %s#%d: %s\n", result.Repo, result.PRNumber, result.Title)
			if result.EstimatedTokens > 0 {
				fmt.Fprintf(out, "  Tokens: %d\n", result.EstimatedTokens)
			}
			if result.EstimatedCost > 0 {
				fmt.Fprintf(out, "  Cost: $%.4f\n", result.EstimatedCost)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "repository (org/repo format)")
	cmd.Flags().IntVar(&prNum, "pr", 0, "pull request number")
	cmd.Flags().IntVar(&tokens, "tokens", 0, "estimated token count")
	cmd.Flags().Float64Var(&cost, "cost", 0, "estimated cost in USD")

	return cmd
}
