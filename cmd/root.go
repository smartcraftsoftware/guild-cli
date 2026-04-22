package cmd

import (
	"fmt"
	"os"

	"github.com/smartcraftsoftware/guild-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Cfg is the loaded configuration, available to all subcommands after PersistentPreRun.
	Cfg *config.Config

	// CfgPath is the resolved config file path.
	CfgPath string

	// flagServer is the --server flag override.
	flagServer string

	// flagConfig is the --config flag override.
	flagConfig string
)

// rootCmd is the base command for the guild CLI.
var rootCmd = &cobra.Command{
	Use:   "guild",
	Short: "Guild CLI — terminal interface to SmartCraft's Guild platform",
	Long: `Guild CLI provides a focused developer interface to SmartCraft's Guild
project tracking platform. Manage issues, log time, report token costs,
and pull context — all from the terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Resolve config path
		if flagConfig != "" {
			CfgPath = flagConfig
		} else {
			p, err := config.DefaultPath()
			if err != nil {
				return fmt.Errorf("resolving config path: %w", err)
			}
			CfgPath = p
		}

		// Load config
		cfg, err := config.Load(CfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Apply --server override
		if flagServer != "" {
			cfg.ServerURL = flagServer
		}

		Cfg = cfg
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "", "Guild server URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "config file path (default: ~/.guild/config.yaml)")
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newIssuesCmd())
	rootCmd.AddCommand(newTimeCmd())
	rootCmd.AddCommand(newCommitCmd())
	rootCmd.AddCommand(newHookCmd())
	rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(newContextCmd())
}

// Execute runs the root command. Called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewRootCmd returns a fresh root command for testing.
// Subcommands are added here to keep wiring in one place.
func NewRootCmd() *cobra.Command {
	// Reset globals
	Cfg = nil
	CfgPath = ""
	flagServer = ""
	flagConfig = ""

	root := &cobra.Command{
		Use:   "guild",
		Short: "Guild CLI — terminal interface to SmartCraft's Guild platform",
		Long: `Guild CLI provides a focused developer interface to SmartCraft's Guild
project tracking platform. Manage issues, log time, report token costs,
and pull context — all from the terminal.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfgFlag, _ := cmd.Flags().GetString("config")
			if cfgFlag != "" {
				CfgPath = cfgFlag
			} else {
				p, err := config.DefaultPath()
				if err != nil {
					return fmt.Errorf("resolving config path: %w", err)
				}
				CfgPath = p
			}

			cfg, err := config.Load(CfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			serverFlag, _ := cmd.Flags().GetString("server")
			if serverFlag != "" {
				cfg.ServerURL = serverFlag
			}

			Cfg = cfg
			return nil
		},
	}

	root.PersistentFlags().StringVar(&flagServer, "server", "", "Guild server URL (overrides config)")
	root.PersistentFlags().StringVar(&flagConfig, "config", "", "config file path (default: ~/.guild/config.yaml)")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newIssuesCmd())
	root.AddCommand(newTimeCmd())
	root.AddCommand(newCommitCmd())
	root.AddCommand(newHookCmd())
	root.AddCommand(newSetupCmd())
	root.AddCommand(newContextCmd())

	return root
}
