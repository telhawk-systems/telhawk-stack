package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Alerts and cases management",
	Long:  "View and manage security alerts and investigation cases",
}

var alertsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List security alerts",
	Long:    "List all security alerts from the alerting service",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		severity, _ := cmd.Flags().GetString("severity")
		rule, _ := cmd.Flags().GetString("rule")

		filters := make(map[string]string)
		if severity != "" {
			filters["severity"] = severity
		}
		if rule != "" {
			filters["rule"] = rule
		}

		alertsResp, err := alertingClient.ListAlerts(p.AccessToken, page, limit, filters)
		if err != nil {
			return fmt.Errorf("failed to list alerts: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(alertsResp.Alerts)
		}

		if len(alertsResp.Alerts) == 0 {
			output.Info("No alerts found")
			return nil
		}

		table := output.NewTable([]string{"ID", "Detection", "Severity", "Event Count", "Triggered At"})
		for _, alert := range alertsResp.Alerts {
			count := ""
			if alert.EventCount() > 0 {
				count = fmt.Sprintf("%d", alert.EventCount())
			} else if alert.DistinctCount() > 0 {
				count = fmt.Sprintf("%d distinct", alert.DistinctCount())
			}

			table.AddRow([]string{
				alert.ID,
				alert.DetectionName(),
				alert.Severity,
				count,
				alert.TriggeredAt().Format("2006-01-02 15:04"),
			})
		}
		table.Render()

		// Show pagination info if available
		if total, ok := alertsResp.Pagination["total"].(float64); ok {
			output.Info("\nShowing %d of %d total alerts", len(alertsResp.Alerts), int(total))
		}

		return nil
	},
}

var alertsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get alert details",
	Long:  "Retrieve detailed information about a specific alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alertID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		alert, err := alertingClient.GetAlert(p.AccessToken, alertID)
		if err != nil {
			return fmt.Errorf("failed to get alert: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(alert)
		}

		output.Info("Alert ID: %s", alert.ID)
		output.Info("Detection: %s", alert.DetectionName())
		output.Info("Severity: %s", alert.Severity)
		output.Info("Triggered: %s", alert.TriggeredAt().Format("2006-01-02 15:04:05"))

		if alert.EventCount() > 0 {
			output.Info("Event Count: %d", alert.EventCount())
		}
		if alert.DistinctCount() > 0 {
			output.Info("Distinct Count: %d", alert.DistinctCount())
		}

		if alert.RawData.GroupKey != "" {
			output.Info("\nGroup Key: %s", alert.RawData.GroupKey)
		}
		if len(alert.RawData.GroupBy) > 0 {
			output.Info("Grouped By: %v", alert.RawData.GroupBy)
		}

		output.Info("\nDetection Rule:")
		output.Info("  ID: %s", alert.DetectionSchemaID)
		output.Info("  Version ID: %s", alert.DetectionSchemaVersionID)

		return nil
	},
}

var casesCmd = &cobra.Command{
	Use:   "cases",
	Short: "Investigation cases management",
	Long:  "Manage investigation cases for security alerts",
}

var casesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List investigation cases",
	Long:    "List all investigation cases",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		status, _ := cmd.Flags().GetString("status")

		casesResp, err := alertingClient.ListCases(p.AccessToken, page, limit, status)
		if err != nil {
			return fmt.Errorf("failed to list cases: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(casesResp.Cases)
		}

		if len(casesResp.Cases) == 0 {
			output.Info("No cases found")
			return nil
		}

		table := output.NewTable([]string{"ID", "Title", "Status", "Severity", "Alerts", "Created"})
		for _, c := range casesResp.Cases {
			table.AddRow([]string{
				c.ID,
				c.Title,
				c.Status,
				c.Severity,
				fmt.Sprintf("%d", c.AlertCount),
				c.CreatedAt.Format("2006-01-02"),
			})
		}
		table.Render()

		// Show pagination info if available
		if total, ok := casesResp.Pagination["total"].(float64); ok {
			output.Info("\nShowing %d of %d total cases", len(casesResp.Cases), int(total))
		}

		return nil
	},
}

var casesGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get case details",
	Long:  "Retrieve detailed information about a specific case",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		caseID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		caseData, err := alertingClient.GetCase(p.AccessToken, caseID)
		if err != nil {
			return fmt.Errorf("failed to get case: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(caseData)
		}

		output.Info("Case ID: %s", caseData.ID)
		output.Info("Title: %s", caseData.Title)
		output.Info("Status: %s", caseData.Status)
		output.Info("Severity: %s", caseData.Severity)
		output.Info("Description: %s", caseData.Description)
		output.Info("Created By: %s", caseData.CreatedBy)
		output.Info("Created: %s", caseData.CreatedAt.Format("2006-01-02 15:04:05"))
		output.Info("Updated: %s", caseData.UpdatedAt.Format("2006-01-02 15:04:05"))

		if caseData.AssignedTo != "" {
			output.Info("Assigned To: %s", caseData.AssignedTo)
		}

		if caseData.ClosedAt != nil {
			output.Info("Closed: %s by %s", caseData.ClosedAt.Format("2006-01-02 15:04:05"), caseData.ClosedBy)
		}

		if caseData.AlertCount > 0 {
			output.Info("\nAssociated Alerts: %d", caseData.AlertCount)
		}

		return nil
	},
}

var casesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new investigation case",
	Long:  "Create a new investigation case for tracking security incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		severity, _ := cmd.Flags().GetString("severity")

		if title == "" {
			return fmt.Errorf("title is required")
		}

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		caseData, err := alertingClient.CreateCase(p.AccessToken, title, description, severity)
		if err != nil {
			return fmt.Errorf("failed to create case: %w", err)
		}

		output.Success("Case created: %s", caseData.Title)
		output.Info("ID: %s", caseData.ID)
		output.Info("Status: %s", caseData.Status)

		return nil
	},
}

var casesCloseCmd = &cobra.Command{
	Use:   "close [id]",
	Short: "Close an investigation case",
	Long:  "Close a case that has been resolved",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		caseID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		if err := alertingClient.CloseCase(p.AccessToken, caseID); err != nil {
			return fmt.Errorf("failed to close case: %w", err)
		}

		output.Success("Case %s closed successfully", caseID)
		return nil
	},
}

var casesReopenCmd = &cobra.Command{
	Use:   "reopen [id]",
	Short: "Reopen a closed case",
	Long:  "Reopen a previously closed case for further investigation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		caseID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		alertingURL := cfg.GetAlertingURL(profile)
		alertingClient := client.NewAlertingClient(alertingURL)

		if err := alertingClient.ReopenCase(p.AccessToken, caseID); err != nil {
			return fmt.Errorf("failed to reopen case: %w", err)
		}

		output.Success("Case %s reopened successfully", caseID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(alertsCmd)
	alertsCmd.AddCommand(alertsListCmd)
	alertsCmd.AddCommand(alertsGetCmd)
	alertsCmd.AddCommand(casesCmd)

	casesCmd.AddCommand(casesListCmd)
	casesCmd.AddCommand(casesGetCmd)
	casesCmd.AddCommand(casesCreateCmd)
	casesCmd.AddCommand(casesCloseCmd)
	casesCmd.AddCommand(casesReopenCmd)

	// Alerts list flags
	alertsListCmd.Flags().IntP("page", "p", 1, "Page number")
	alertsListCmd.Flags().IntP("limit", "l", 20, "Results per page")
	alertsListCmd.Flags().StringP("severity", "s", "", "Filter by severity")
	alertsListCmd.Flags().StringP("rule", "r", "", "Filter by rule ID")

	// Cases list flags
	casesListCmd.Flags().IntP("page", "p", 1, "Page number")
	casesListCmd.Flags().IntP("limit", "l", 20, "Results per page")
	casesListCmd.Flags().String("status", "", "Filter by status (open, in_progress, resolved, closed)")

	// Cases create flags
	casesCreateCmd.Flags().StringP("title", "t", "", "Case title")
	casesCreateCmd.Flags().StringP("description", "d", "", "Case description")
	casesCreateCmd.Flags().StringP("severity", "s", "medium", "Case severity (low, medium, high, critical)")
	if err := casesCreateCmd.MarkFlagRequired("title"); err != nil {
		panic(fmt.Sprintf("failed to mark title as required: %v", err))
	}
}
