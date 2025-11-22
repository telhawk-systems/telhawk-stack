package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Detection rules management",
	Long:  "Create, list, and manage detection rules in the TelHawk Stack",
}

var rulesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List detection rules",
	Long:    "List all detection rules from the rules service",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		rulesURL := cfg.GetRulesURL(profile)
		rulesClient := client.NewRulesClient(rulesURL)

		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")

		schemas, meta, err := rulesClient.ListSchemas(p.AccessToken, page, limit)
		if err != nil {
			return fmt.Errorf("failed to list rules: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(schemas)
		}

		if len(schemas) == 0 {
			output.Info("No detection rules found")
			return nil
		}

		table := output.NewTable([]string{"ID", "Name", "Correlation", "Filters", "Severity", "Status", "Created"})
		for _, schema := range schemas {
			name := getString(schema.Attributes.View, "name", "title")
			severity := getString(schema.Attributes.View, "severity")
			correlationType := getString(schema.Attributes.Model, "correlation_type")
			filters := getFilterFields(schema.Attributes.Model)

			status := "enabled"
			if schema.Attributes.DisabledAt != nil {
				status = "disabled"
			}
			if schema.Attributes.HiddenAt != nil {
				status = "hidden"
			}

			table.AddRow([]string{
				schema.ID,
				name,
				correlationType,
				filters,
				severity,
				status,
				schema.Attributes.CreatedAt.Format("2006-01-02"),
			})
		}
		table.Render()

		// Show pagination info if available
		if total, ok := meta["total"].(float64); ok {
			output.Info("\nShowing %d of %d total rules", len(schemas), int(total))
		}

		return nil
	},
}

var rulesGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a detection rule by ID",
	Long:  "Retrieve detailed information about a specific detection rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		rulesURL := cfg.GetRulesURL(profile)
		rulesClient := client.NewRulesClient(rulesURL)

		schema, err := rulesClient.GetSchema(p.AccessToken, ruleID)
		if err != nil {
			return fmt.Errorf("failed to get rule: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(schema)
		}

		// Display in human-readable format
		name := getString(schema.Attributes.View, "name", "title")
		description := getString(schema.Attributes.View, "description")
		severity := getString(schema.Attributes.View, "severity")
		correlationType := getString(schema.Attributes.Model, "correlation_type")

		output.Info("Rule: %s", name)
		output.Info("ID: %s", schema.ID)
		output.Info("Version ID: %s", schema.Attributes.VersionID)
		output.Info("Description: %s", description)
		output.Info("Severity: %s", severity)
		output.Info("Correlation Type: %s", correlationType)
		output.Info("Created: %s", schema.Attributes.CreatedAt.Format("2006-01-02 15:04:05"))

		if schema.Attributes.DisabledAt != nil {
			output.Info("Status: Disabled at %s", schema.Attributes.DisabledAt.Format("2006-01-02 15:04:05"))
		} else {
			output.Info("Status: Enabled")
		}

		// Show MITRE ATT&CK mapping if available
		if mitreData, ok := schema.Attributes.View["mitre_attack"].(map[string]interface{}); ok {
			if tactics, ok := mitreData["tactics"].([]interface{}); ok && len(tactics) > 0 {
				output.Info("\nMITRE ATT&CK:")
				for _, tactic := range tactics {
					output.Info("  Tactic: %v", tactic)
				}
			}
			if techniques, ok := mitreData["techniques"].([]interface{}); ok && len(techniques) > 0 {
				for _, technique := range techniques {
					output.Info("  Technique: %v", technique)
				}
			}
		}

		return nil
	},
}

var rulesCreateCmd = &cobra.Command{
	Use:   "create [file]",
	Short: "Create a new detection rule from a JSON file",
	Long:  "Create a new detection rule by providing a JSON file with model, view, and controller sections",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		// Read the rule file
		fileData, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var ruleData struct {
			Model      map[string]interface{} `json:"model"`
			View       map[string]interface{} `json:"view"`
			Controller map[string]interface{} `json:"controller"`
		}
		if err := json.Unmarshal(fileData, &ruleData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		rulesURL := cfg.GetRulesURL(profile)
		rulesClient := client.NewRulesClient(rulesURL)

		schema, err := rulesClient.CreateSchema(p.AccessToken, ruleData.Model, ruleData.View, ruleData.Controller)
		if err != nil {
			return fmt.Errorf("failed to create rule: %w", err)
		}

		name := getString(schema.Attributes.View, "name", "title")
		output.Success("Detection rule created: %s", name)
		output.Info("ID: %s", schema.ID)
		output.Info("Version ID: %s", schema.Attributes.VersionID)

		return nil
	},
}

var rulesDisableCmd = &cobra.Command{
	Use:   "disable [id]",
	Short: "Disable a detection rule",
	Long:  "Disable a detection rule to stop it from generating alerts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		rulesURL := cfg.GetRulesURL(profile)
		rulesClient := client.NewRulesClient(rulesURL)

		if err := rulesClient.DisableSchema(p.AccessToken, ruleID); err != nil {
			return fmt.Errorf("failed to disable rule: %w", err)
		}

		output.Success("Rule %s disabled successfully", ruleID)
		return nil
	},
}

var rulesEnableCmd = &cobra.Command{
	Use:   "enable [id]",
	Short: "Enable a detection rule",
	Long:  "Enable a previously disabled detection rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		rulesURL := cfg.GetRulesURL(profile)
		rulesClient := client.NewRulesClient(rulesURL)

		if err := rulesClient.EnableSchema(p.AccessToken, ruleID); err != nil {
			return fmt.Errorf("failed to enable rule: %w", err)
		}

		output.Success("Rule %s enabled successfully", ruleID)
		return nil
	},
}

var rulesVersionsCmd = &cobra.Command{
	Use:   "versions [id]",
	Short: "List version history for a detection rule",
	Long:  "Show all versions of a detection rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleID := args[0]

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		rulesURL := cfg.GetRulesURL(profile)
		rulesClient := client.NewRulesClient(rulesURL)

		versions, err := rulesClient.GetVersionHistory(p.AccessToken, ruleID)
		if err != nil {
			return fmt.Errorf("failed to get version history: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(versions)
		}

		if len(versions) == 0 {
			output.Info("No versions found")
			return nil
		}

		table := output.NewTable([]string{"Version #", "Version ID", "Created", "Status"})
		for i, version := range versions {
			status := "enabled"
			if version.Attributes.DisabledAt != nil {
				status = "disabled"
			}

			table.AddRow([]string{
				fmt.Sprintf("%d", len(versions)-i),
				version.Attributes.VersionID,
				version.Attributes.CreatedAt.Format("2006-01-02 15:04:05"),
				status,
			})
		}
		table.Render()

		return nil
	},
}

// Helper function to get string value from nested map
func getString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

// getFilterFields extracts filter field names from the model
func getFilterFields(model map[string]interface{}) string {
	params, ok := model["parameters"].(map[string]interface{})
	if !ok {
		return ""
	}
	query, ok := params["query"].(map[string]interface{})
	if !ok {
		return ""
	}
	filter, ok := query["filter"].(map[string]interface{})
	if !ok {
		return ""
	}

	var fields []string

	// Check for compound filter with conditions array
	if conditions, ok := filter["conditions"].([]interface{}); ok {
		for _, cond := range conditions {
			if c, ok := cond.(map[string]interface{}); ok {
				if field, ok := c["field"].(string); ok {
					fields = append(fields, field)
				}
			}
		}
	} else if field, ok := filter["field"].(string); ok {
		// Simple single filter
		fields = append(fields, field)
	}

	return strings.Join(fields, ", ")
}

func init() {
	rootCmd.AddCommand(rulesCmd)
	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesGetCmd)
	rulesCmd.AddCommand(rulesCreateCmd)
	rulesCmd.AddCommand(rulesDisableCmd)
	rulesCmd.AddCommand(rulesEnableCmd)
	rulesCmd.AddCommand(rulesVersionsCmd)

	// List command flags
	rulesListCmd.Flags().IntP("page", "p", 1, "Page number")
	rulesListCmd.Flags().IntP("limit", "l", 50, "Results per page")
}
