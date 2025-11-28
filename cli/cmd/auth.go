package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to TelHawk Stack",
	Long:  "Authenticate with TelHawk Stack and save credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, err := cmd.Flags().GetString("username")
		if err != nil {
			return fmt.Errorf("failed to get username: %w", err)
		}
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return fmt.Errorf("failed to get password: %w", err)
		}
		authURL, err := cmd.Flags().GetString("auth-url")
		if err != nil {
			return fmt.Errorf("failed to get auth-url: %w", err)
		}

		// Use config default if not provided via flag
		if authURL == "" || !cmd.Flags().Changed("auth-url") {
			profile, err := cmd.Flags().GetString("profile")
			if err != nil {
				return fmt.Errorf("failed to get profile: %w", err)
			}
			authURL = cfg.GetAuthURL(profile)
		}

		if username == "" {
			return fmt.Errorf("username is required")
		}
		if password == "" {
			return fmt.Errorf("password is required")
		}

		authClient := client.NewAuthClient(authURL)
		resp, err := authClient.Login(username, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Save credentials to config
		profile, err := cmd.Flags().GetString("profile")
		if err != nil {
			return fmt.Errorf("failed to get profile: %w", err)
		}
		if err := cfg.SaveProfile(profile, authURL, resp.AccessToken, resp.RefreshToken); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		output.Success("Successfully logged in as %s", username)
		output.Info("Profile '%s' saved to ~/.thawk/config.yaml", profile)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from TelHawk Stack",
	Long:  "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := cmd.Flags().GetString("profile")
		if err != nil {
			return fmt.Errorf("failed to get profile: %w", err)
		}

		if err := cfg.RemoveProfile(profile); err != nil {
			return err
		}

		output.Success("Successfully logged out from profile '%s'", profile)
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current user information",
	Long:  "Show information about the currently authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := cmd.Flags().GetString("profile")
		if err != nil {
			return fmt.Errorf("failed to get profile: %w", err)
		}
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		valid, err := authClient.ValidateToken(p.AccessToken)
		if err != nil || !valid.Valid {
			return fmt.Errorf("token invalid or expired, please run 'thawk auth login'")
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(valid)
		}

		output.Info("Profile: %s", profile)
		output.Info("User ID: %s", valid.UserID)
		output.Info("Roles: %v", valid.Roles)
		output.Info("Auth URL: %s", p.AuthURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)

	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
	loginCmd.Flags().String("auth-url", "", "Auth service URL (default from config/env)")
	if err := loginCmd.MarkFlagRequired("username"); err != nil {
		panic(fmt.Sprintf("failed to mark username as required: %v", err))
	}
	if err := loginCmd.MarkFlagRequired("password"); err != nil {
		panic(fmt.Sprintf("failed to mark password as required: %v", err))
	}
}
