package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  "Manage authentication with TelHawk Stack services",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to TelHawk Stack",
	Long:  "Authenticate with TelHawk Stack and save credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		authURL, _ := cmd.Flags().GetString("auth-url")
		
		// Use config default if not provided via flag
		if authURL == "" || !cmd.Flags().Changed("auth-url") {
			profile, _ := cmd.Flags().GetString("profile")
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
		profile, _ := cmd.Flags().GetString("profile")
		if err := cfg.SaveProfile(profile, authURL, resp.AccessToken, resp.RefreshToken); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		output.Success("Successfully logged in as %s", username)
		output.Info("Profile '%s' saved to ~/.thawk/config.yaml", profile)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from TelHawk Stack",
	Long:  "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		
		if err := cfg.RemoveProfile(profile); err != nil {
			return err
		}

		output.Success("Successfully logged out from profile '%s'", profile)
		return nil
	},
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current user information",
	Long:  "Show information about the currently authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
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
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authWhoamiCmd)

	authLoginCmd.Flags().StringP("username", "u", "", "Username")
	authLoginCmd.Flags().StringP("password", "p", "", "Password")
	authLoginCmd.Flags().String("auth-url", "", "Auth service URL (default from config/env)")
	authLoginCmd.MarkFlagRequired("username")
	authLoginCmd.MarkFlagRequired("password")
}
