package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "context",
		Short: "Get assigned issues as JSON for GSD",
		Long: `Returns your assigned issues and user info as parseable JSON.
Designed for GSD and other tools to consume programmatically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiURL := fmt.Sprintf("%s/api/v1/context", Cfg.ServerURL)

			req, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				return fmt.Errorf("creating request: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+Cfg.Token)
			req.Header.Set("Accept", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("fetching context: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("not authenticated — run 'guild auth login' first")
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("context request failed (status %d)", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decoding context: %w", err)
			}

			out := cmd.OutOrStdout()

			if jsonOutput {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			// Pretty-print summary
			if user, ok := result["user"].(map[string]interface{}); ok {
				fmt.Fprintf(out, "User: %s (%s)\n\n", user["name"], user["email"])
			}

			issues, ok := result["assigned_issues"].([]interface{})
			if !ok || len(issues) == 0 {
				fmt.Fprintln(out, "No assigned issues.")
				return nil
			}

			fmt.Fprintf(out, "Assigned Issues (%d):\n", len(issues))
			for _, raw := range issues {
				issue, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				fmt.Fprintf(out, "  [%s] %s — %s (%s)\n",
					issue["priority"],
					issue["title"],
					issue["status"],
					issue["team_name"],
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output raw JSON (for GSD consumption)")

	return cmd
}
