package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables injected via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the guild CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "guild version %s (commit: %s, built: %s)\n", Version, Commit, Date)
			return nil
		},
	}
}
