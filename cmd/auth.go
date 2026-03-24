package cmd

import (
	"fmt"
	"time"

	"github.com/smartcraftsoftware/guild-cli/internal/auth"
	"github.com/smartcraftsoftware/guild-cli/internal/config"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with a Guild server",
		Long:  "Log in, check status, or log out of your Guild server.",
	}

	authCmd.AddCommand(newAuthLoginCmd())
	authCmd.AddCommand(newAuthStatusCmd())
	authCmd.AddCommand(newAuthLogoutCmd())

	return authCmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Guild via browser",
		Long: `Starts a device authorization flow: opens your browser to log in to Guild,
then stores the API token locally for future CLI commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverURL := Cfg.ServerURL
			out := cmd.OutOrStdout()

			fmt.Fprintf(out, "Starting authentication with %s...\n\n", serverURL)

			// Step 1: Start device flow
			dr, err := auth.StartDeviceFlow(serverURL)
			if err != nil {
				return fmt.Errorf("failed to start auth flow: %w", err)
			}

			// Step 2: Open browser
			fmt.Fprintf(out, "Your confirmation code: %s\n\n", dr.UserCode)
			fmt.Fprintf(out, "Opening browser to: %s\n", dr.VerificationURI)
			fmt.Fprintf(out, "If the browser doesn't open, visit the URL above manually.\n\n")

			if err := auth.OpenBrowser(dr.VerificationURI); err != nil {
				fmt.Fprintf(out, "Could not open browser: %v\n", err)
			}

			// Step 3: Poll for token
			fmt.Fprintf(out, "Waiting for authorization...\n")

			timeout := time.Duration(dr.ExpiresIn) * time.Second
			if timeout == 0 {
				timeout = 10 * time.Minute
			}

			tr, err := auth.PollForToken(serverURL, dr.DeviceCode, timeout)
			if err != nil {
				return fmt.Errorf("authorization failed: %w", err)
			}

			// Step 4: Save token to config
			Cfg.Token = tr.AccessToken
			if err := Cfg.Save(CfgPath); err != nil {
				return fmt.Errorf("saving token: %w", err)
			}

			fmt.Fprintf(out, "\n✓ Logged in as %s (%s)\n", tr.User.Name, tr.User.Email)
			fmt.Fprintf(out, "Token saved to %s\n", CfgPath)

			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if Cfg.Token == "" {
				fmt.Fprintln(out, "Not logged in. Run 'guild auth login' to authenticate.")
				return nil
			}

			me, err := auth.GetMe(Cfg.ServerURL, Cfg.Token)
			if err != nil {
				return fmt.Errorf("checking auth status: %w", err)
			}

			fmt.Fprintf(out, "Logged in as %s (%s)\n", me.User.Name, me.User.Email)
			fmt.Fprintf(out, "Server: %s\n", Cfg.ServerURL)

			if len(me.User.Teams) > 0 {
				fmt.Fprintf(out, "Teams:\n")
				for _, t := range me.User.Teams {
					fmt.Fprintf(out, "  - %s (id: %d)\n", t.Name, t.ID)
				}
			}

			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored authentication token",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if Cfg.Token == "" {
				fmt.Fprintln(out, "Not logged in.")
				return nil
			}

			Cfg.Token = ""
			if err := Cfg.Save(CfgPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			// Reload to confirm it's cleared
			reloaded, err := config.Load(CfgPath)
			if err == nil && reloaded.Token == "" {
				fmt.Fprintln(out, "✓ Logged out. Token removed from config.")
			} else {
				fmt.Fprintln(out, "Logged out.")
			}

			return nil
		},
	}
}
