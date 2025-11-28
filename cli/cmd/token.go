package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "HEC token management",
	Long:  "Create and manage HEC tokens for data ingestion",
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new HEC token",
	Long:  "Create a new HEC token for Splunk-compatible ingestion",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		expiresIn, _ := cmd.Flags().GetString("expires")
		setDefault, _ := cmd.Flags().GetBool("set-default")

		if name == "" {
			return fmt.Errorf("token name is required")
		}

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		token, err := authClient.CreateHECToken(p.AccessToken, name, expiresIn)
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}

		// Optionally save token as default for this profile
		if setDefault {
			if err := cfg.SaveHECToken(profile, token.Token); err != nil {
				output.Warn("Failed to save token as default: %v", err)
			} else {
				output.Info("Token saved as default for profile '%s'", profile)
			}
		}

		output.Success("HEC token created: %s", token.Token)
		output.Info("Name: %s", token.Name)
		if !token.ExpiresAt.IsZero() {
			output.Info("Expires: %s", token.ExpiresAt.Format(time.RFC3339))
		}
		output.Info("\nUse this token with:")
		output.Info("  curl -H 'Authorization: Splunk %s' ...", token.Token)
		if !setDefault {
			output.Info("\nTo save as default token for this profile, use --set-default flag")
		}
		return nil
	},
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List HEC tokens",
	Long:  "List all HEC tokens for the current user",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		tokens, err := authClient.ListHECTokens(p.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to list tokens: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(tokens)
		}

		table := output.NewTable([]string{"Name", "Token", "Enabled", "Created", "Expires"})
		for _, token := range tokens {
			expires := "Never"
			if !token.ExpiresAt.IsZero() {
				expires = token.ExpiresAt.Format("2006-01-02")
			}
			table.AddRow([]string{
				token.Name,
				token.Token[:16] + "...",
				fmt.Sprintf("%t", token.Enabled),
				token.CreatedAt.Format("2006-01-02"),
				expires,
			})
		}
		table.Render()
		return nil
	},
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke [token]",
	Short: "Revoke a HEC token",
	Long:  "Revoke a HEC token by token string",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		if err := authClient.RevokeHECToken(p.AccessToken, token); err != nil {
			return fmt.Errorf("failed to revoke token: %w", err)
		}

		output.Success("Token revoked successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)

	tokenCreateCmd.Flags().StringP("name", "n", "", "Token name")
	tokenCreateCmd.Flags().String("expires", "", "Expiration duration (e.g., 30d, 1y)")
	tokenCreateCmd.Flags().Bool("set-default", false, "Save this token as the default for the current profile")
	if err := tokenCreateCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("failed to mark name as required: %v", err))
	}
}
