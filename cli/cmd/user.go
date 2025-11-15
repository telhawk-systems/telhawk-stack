package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  "Manage TelHawk Stack users",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long:  "Display a list of all users in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		users, err := authClient.ListUsers(p.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(users)
		}

		// Table output
		output.Info("Users (%d total):", len(users))
		fmt.Println()
		fmt.Printf("%-36s %-20s %-30s %-15s %-20s\n", "ID", "USERNAME", "EMAIL", "STATUS", "ROLES")
		fmt.Println("---------------------------------------------------------------------------------------------------------------------------")
		for _, user := range users {
			status := "enabled"
			if !user.Enabled {
				status = "disabled"
			}
			fmt.Printf("%-36s %-20s %-30s %-15s %-20v\n",
				user.ID, user.Username, user.Email, status, user.Roles)
		}
		return nil
	},
}

var userGetCmd = &cobra.Command{
	Use:   "get [user-id]",
	Short: "Get user details",
	Long:  "Display detailed information about a specific user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		user, err := authClient.GetUser(p.AccessToken, userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(user)
		}

		output.Info("User Details:")
		fmt.Printf("  ID:       %s\n", user.ID)
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  Email:    %s\n", user.Email)
		fmt.Printf("  Roles:    %v\n", user.Roles)
		fmt.Printf("  Enabled:  %v\n", user.Enabled)
		fmt.Printf("  Created:  %s\n", user.CreatedAt)
		fmt.Printf("  Updated:  %s\n", user.UpdatedAt)
		return nil
	},
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  "Create a new user in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		roles, err := cmd.Flags().GetStringSlice("roles")
		if err != nil {
			return fmt.Errorf("failed to get roles: %w", err)
		}

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		user, err := authClient.CreateUser(p.AccessToken, username, email, password, roles)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		output.Success("User created successfully")
		fmt.Printf("  ID:       %s\n", user.ID)
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  Email:    %s\n", user.Email)
		fmt.Printf("  Roles:    %v\n", user.Roles)
		return nil
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update [user-id]",
	Short: "Update a user",
	Long:  "Update user details (email, roles, enabled status)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		email, _ := cmd.Flags().GetString("email")
		roles, _ := cmd.Flags().GetStringSlice("roles")
		enabled, _ := cmd.Flags().GetBool("enabled")
		disabled, _ := cmd.Flags().GetBool("disabled")

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		var enabledPtr *bool
		if cmd.Flags().Changed("enabled") {
			enabledPtr = &enabled
		} else if cmd.Flags().Changed("disabled") {
			val := !disabled
			enabledPtr = &val
		}

		authClient := client.NewAuthClient(p.AuthURL)
		user, err := authClient.UpdateUser(p.AccessToken, userID, email, roles, enabledPtr)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		output.Success("User updated successfully")
		fmt.Printf("  ID:       %s\n", user.ID)
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  Email:    %s\n", user.Email)
		fmt.Printf("  Roles:    %v\n", user.Roles)
		fmt.Printf("  Enabled:  %v\n", user.Enabled)
		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete [user-id]",
	Short: "Delete a user",
	Long:  "Permanently delete a user from the system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			return fmt.Errorf("use --force to confirm user deletion")
		}

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		err = authClient.DeleteUser(p.AccessToken, userID)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		output.Success("User deleted successfully")
		return nil
	},
}

var userResetPasswordCmd = &cobra.Command{
	Use:   "reset-password [user-id]",
	Short: "Reset a user's password",
	Long:  "Reset a user's password to a new value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		newPassword, _ := cmd.Flags().GetString("password")

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		authClient := client.NewAuthClient(p.AuthURL)
		err = authClient.ResetPassword(p.AccessToken, userID, newPassword)
		if err != nil {
			return fmt.Errorf("failed to reset password: %w", err)
		}

		output.Success("Password reset successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userGetCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userResetPasswordCmd)

	// Create flags
	userCreateCmd.Flags().StringP("username", "u", "", "Username (required)")
	userCreateCmd.Flags().StringP("email", "e", "", "Email address (required)")
	userCreateCmd.Flags().StringP("password", "p", "", "Password (required)")
	userCreateCmd.Flags().StringSliceP("roles", "r", []string{"viewer"}, "Roles (comma-separated)")
	userCreateCmd.MarkFlagRequired("username")
	userCreateCmd.MarkFlagRequired("email")
	userCreateCmd.MarkFlagRequired("password")

	// Update flags
	userUpdateCmd.Flags().StringP("email", "e", "", "New email address")
	userUpdateCmd.Flags().StringSliceP("roles", "r", nil, "New roles (comma-separated)")
	userUpdateCmd.Flags().Bool("enabled", false, "Enable the user")
	userUpdateCmd.Flags().Bool("disabled", false, "Disable the user")

	// Delete flags
	userDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")

	// Reset password flags
	userResetPasswordCmd.Flags().StringP("password", "p", "", "New password (required)")
	userResetPasswordCmd.MarkFlagRequired("password")
}
