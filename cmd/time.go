package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/smartcraftsoftware/guild-cli/internal/api"
	"github.com/smartcraftsoftware/guild-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTimeCmd() *cobra.Command {
	timeCmd := &cobra.Command{
		Use:   "time",
		Short: "Track time entries",
		Long:  "Log, list, and delete time entries.",
	}

	timeCmd.AddCommand(newTimeLogCmd())
	timeCmd.AddCommand(newTimeListCmd())
	timeCmd.AddCommand(newTimeDeleteCmd())

	return timeCmd
}

func newTimeLogCmd() *cobra.Command {
	var (
		projectID   int
		duration    string
		description string
		date        string
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Log a time entry",
		Long:  "Log a time entry. Duration accepts formats: 2h, 1h30m, 90m, 1.5h",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectID == 0 {
				return fmt.Errorf("--project is required")
			}
			if duration == "" {
				return fmt.Errorf("--duration is required")
			}

			minutes, err := parseDuration(duration)
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", duration, err)
			}

			entryDate := date
			if entryDate == "" {
				entryDate = time.Now().Format("2006-01-02")
			}

			client := api.NewClient(Cfg.ServerURL, Cfg.Token)
			entry, err := client.CreateTimeEntry(api.CreateTimeEntryParams{
				ProjectID:       projectID,
				Date:            entryDate,
				DurationMinutes: minutes,
				Description:     description,
				EntryType:       "work",
			})
			if err != nil {
				return fmt.Errorf("logging time: %w", err)
			}

			projectName := ""
			if entry.Project != nil {
				projectName = entry.Project.Name
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Logged %s on %s for %s\n", formatMinutes(entry.DurationMinutes), entry.Date, projectName)
			return nil
		},
	}

	cmd.Flags().IntVar(&projectID, "project", 0, "project ID (required)")
	cmd.Flags().StringVar(&duration, "duration", "", "duration (e.g. 2h, 1h30m, 90m)")
	cmd.Flags().StringVar(&description, "description", "", "description of work")
	cmd.Flags().StringVar(&date, "date", "", "date (YYYY-MM-DD, default: today)")

	return cmd
}

func newTimeListCmd() *cobra.Command {
	var (
		from      string
		to        string
		projectID string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent time entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(Cfg.ServerURL, Cfg.Token)
			entries, err := client.ListTimeEntries(api.TimeEntryListParams{
				From:      from,
				To:        to,
				ProjectID: projectID,
				Limit:     limit,
			})
			if err != nil {
				return fmt.Errorf("listing time entries: %w", err)
			}

			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No time entries found.")
				return nil
			}

			headers := []string{"ID", "DATE", "DURATION", "PROJECT", "DESCRIPTION"}
			var rows [][]string
			for _, e := range entries {
				projectName := ""
				if e.Project != nil {
					projectName = e.Project.Name
				}
				desc := e.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				rows = append(rows, []string{
					strconv.Itoa(e.ID),
					e.Date,
					formatMinutes(e.DurationMinutes),
					projectName,
					desc,
				})
			}

			output.PrintTable(cmd.OutOrStdout(), headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&projectID, "project", "", "filter by project ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "max entries to return")

	return cmd
}

func newTimeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a time entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(Cfg.ServerURL, Cfg.Token)
			if err := client.DeleteTimeEntry(args[0]); err != nil {
				return fmt.Errorf("deleting time entry: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✓ Time entry deleted.")
			return nil
		},
	}
}

// parseDuration parses human-friendly duration strings: 2h, 1h30m, 90m, 1.5h
func parseDuration(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Try Go duration parsing first for formats like "1h30m"
	if d, err := time.ParseDuration(s); err == nil {
		minutes := int(d.Minutes())
		if minutes <= 0 {
			return 0, fmt.Errorf("duration must be positive")
		}
		return minutes, nil
	}

	// Try decimal hours: "1.5h"
	if strings.HasSuffix(s, "h") {
		numStr := strings.TrimSuffix(s, "h")
		if hours, err := strconv.ParseFloat(numStr, 64); err == nil {
			minutes := int(hours * 60)
			if minutes <= 0 {
				return 0, fmt.Errorf("duration must be positive")
			}
			return minutes, nil
		}
	}

	// Try plain minutes: "90m"
	if strings.HasSuffix(s, "m") {
		numStr := strings.TrimSuffix(s, "m")
		if mins, err := strconv.Atoi(numStr); err == nil {
			if mins <= 0 {
				return 0, fmt.Errorf("duration must be positive")
			}
			return mins, nil
		}
	}

	return 0, fmt.Errorf("unrecognized format (try: 2h, 1h30m, 90m, 1.5h)")
}

// formatMinutes converts minutes to a human-readable string like "2h 30m"
func formatMinutes(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
