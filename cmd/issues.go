package cmd

import (
	"fmt"
	"strconv"

	"github.com/smartcraftsoftware/guild-cli/internal/api"
	"github.com/smartcraftsoftware/guild-cli/internal/output"
	"github.com/spf13/cobra"
)

func newIssuesCmd() *cobra.Command {
	issuesCmd := &cobra.Command{
		Use:   "issues",
		Short: "Manage team issues",
		Long:  "List, create, and view issues in your team.",
	}

	issuesCmd.AddCommand(newIssuesListCmd())
	issuesCmd.AddCommand(newIssuesCreateCmd())
	issuesCmd.AddCommand(newIssuesViewCmd())

	return issuesCmd
}

func newIssuesListCmd() *cobra.Command {
	var (
		teamID   string
		status   string
		issType  string
		assignee string
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List team issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			tid := resolveTeamID(teamID)
			if tid == "" {
				return fmt.Errorf("team ID required: use --team flag or 'guild config set team <id>'")
			}

			client := api.NewClient(Cfg.ServerURL, Cfg.Token)
			issues, err := client.ListIssues(tid, api.IssueListParams{
				Status:     status,
				IssueType:  issType,
				AssigneeID: assignee,
				Limit:      limit,
			})
			if err != nil {
				return fmt.Errorf("listing issues: %w", err)
			}

			if len(issues) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No issues found.")
				return nil
			}

			headers := []string{"ID", "TYPE", "PRIORITY", "STATUS", "ASSIGNEE", "TITLE"}
			var rows [][]string
			for _, i := range issues {
				assigneeName := ""
				if i.Assignee != nil {
					assigneeName = i.Assignee.Name
				}
				statusName := ""
				if i.WorkflowStatus != nil {
					statusName = i.WorkflowStatus.Name
				}
				rows = append(rows, []string{
					strconv.Itoa(i.ID),
					i.IssueType,
					i.Priority,
					statusName,
					assigneeName,
					i.Title,
				})
			}

			output.PrintTable(cmd.OutOrStdout(), headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", "team ID (overrides config)")
	cmd.Flags().StringVar(&status, "status", "", "filter by status category (todo, in_progress, done)")
	cmd.Flags().StringVar(&issType, "type", "", "filter by issue type (task, bug, story, epic)")
	cmd.Flags().StringVar(&assignee, "assignee", "", "filter by assignee ID")
	cmd.Flags().IntVar(&limit, "limit", 50, "max issues to return")

	return cmd
}

func newIssuesCreateCmd() *cobra.Command {
	var (
		teamID      string
		title       string
		description string
		issType     string
		priority    string
		assigneeID  int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}

			tid := resolveTeamID(teamID)
			if tid == "" {
				return fmt.Errorf("team ID required: use --team flag or 'guild config set team <id>'")
			}

			client := api.NewClient(Cfg.ServerURL, Cfg.Token)
			issue, err := client.CreateIssue(tid, api.CreateIssueParams{
				Title:       title,
				Description: description,
				IssueType:   issType,
				Priority:    priority,
				AssigneeID:  assigneeID,
			})
			if err != nil {
				return fmt.Errorf("creating issue: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Created issue #%d: %s\n", issue.ID, issue.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", "team ID (overrides config)")
	cmd.Flags().StringVar(&title, "title", "", "issue title (required)")
	cmd.Flags().StringVar(&description, "description", "", "issue description")
	cmd.Flags().StringVar(&issType, "type", "task", "issue type (task, bug, story, epic)")
	cmd.Flags().StringVar(&priority, "priority", "medium", "priority (low, medium, high, urgent)")
	cmd.Flags().IntVar(&assigneeID, "assignee", 0, "assignee user ID")

	return cmd
}

func newIssuesViewCmd() *cobra.Command {
	var teamID string

	cmd := &cobra.Command{
		Use:   "view <issue-id>",
		Short: "View issue details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tid := resolveTeamID(teamID)
			if tid == "" {
				return fmt.Errorf("team ID required: use --team flag or 'guild config set team <id>'")
			}

			client := api.NewClient(Cfg.ServerURL, Cfg.Token)
			issue, err := client.GetIssue(tid, args[0])
			if err != nil {
				return fmt.Errorf("viewing issue: %w", err)
			}

			assignee := "Unassigned"
			if issue.Assignee != nil {
				assignee = issue.Assignee.Name
			}
			status := ""
			if issue.WorkflowStatus != nil {
				status = issue.WorkflowStatus.Name
			}
			createdBy := ""
			if issue.CreatedBy != nil {
				createdBy = issue.CreatedBy.Name
			}

			fields := [][]string{
				{"ID", strconv.Itoa(issue.ID)},
				{"Title", issue.Title},
				{"Type", issue.IssueType},
				{"Priority", issue.Priority},
				{"Status", status},
				{"Assignee", assignee},
				{"Created by", createdBy},
				{"Created", issue.CreatedAt},
			}
			if issue.Description != "" {
				fields = append(fields, []string{"Description", issue.Description})
			}

			output.PrintDetail(cmd.OutOrStdout(), fields)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", "team ID (overrides config)")

	return cmd
}

// resolveTeamID returns the team ID from the flag or config.
func resolveTeamID(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if Cfg != nil {
		return Cfg.TeamID
	}
	return ""
}
