package cmd

import (
	"fmt"

	"github.com/smartcraftsoftware/guild-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage guild CLI configuration",
		Long:  "Get and set configuration values stored in ~/.guild/config.yaml.",
	}

	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigGetCmd())

	return configCmd
}

// supportedKeys lists the config keys users can get/set.
var supportedKeys = map[string]bool{
	"server": true,
	"team":   true,
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Set a configuration value. Supported keys: server, team.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			if !supportedKeys[key] {
				return fmt.Errorf("unknown config key %q (supported: server, team)", key)
			}

			// Load current config
			cfg, err := config.Load(CfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			switch key {
			case "server":
				cfg.ServerURL = value
			case "team":
				cfg.TeamID = value
			}

			if err := cfg.Save(CfgPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  "Get a configuration value. Supported keys: server, team.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			if !supportedKeys[key] {
				return fmt.Errorf("unknown config key %q (supported: server, team)", key)
			}

			cfg, err := config.Load(CfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			var value string
			switch key {
			case "server":
				value = cfg.ServerURL
			case "team":
				value = cfg.TeamID
			}

			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		},
	}
}
