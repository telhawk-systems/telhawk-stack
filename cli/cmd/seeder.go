package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/seeder"
)

var (
	seederCfgFile    string
	seederHECURL     string
	seederToken      string
	seederCount      int
	seederTimeSpread string
	seederBatchSize  int
	seederEventTypes string
	seederAttacks    string
	seederFromRules  string
)

var seederCmd = &cobra.Command{
	Use:   "seeder",
	Short: "Event seeder commands",
	Long:  "Generate and inject realistic OCSF events for testing and development",
}

var seederRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the event seeder",
	Long: `Generate and send events to HEC endpoint.

Configuration cascade (priority order):
  1. Command-line flags
  2. ./seeder.yaml (project directory)
  3. ~/.thawk/seeder.yaml (user directory)
  4. Built-in defaults

Examples:
  # Use project config
  thawk seeder run

  # Override specific values
  thawk seeder run --token YOUR_TOKEN --count 1000

  # Run specific attacks
  thawk seeder run --attack brute_force_admin,credential_stuffing

  # Use custom config file
  thawk seeder run --config ./custom-seeder.yaml`,
	RunE: runSeeder,
}

var seederListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available attack patterns",
	Long:  "Display all available MITRE ATT&CK patterns that can be used with the seeder",
	Run: func(cmd *cobra.Command, args []string) {
		seeder.ListAttacks()
	},
}

var seederListRulesCmd = &cobra.Command{
	Use:   "list-rules [directory]",
	Short: "List detection rules that can be used for event generation",
	Long:  "Scan a directory for detection rule JSON files and display which ones are supported for event generation",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rulesDir := "./alerting/rules"
		if len(args) > 0 {
			rulesDir = args[0]
		}

		fmt.Printf("Loading rules from %s\n\n", rulesDir)

		rules, err := seeder.LoadRulesFromDirectory(rulesDir)
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}

		if len(rules) == 0 {
			fmt.Println("No rule files found")
			return nil
		}

		supportedCount := 0
		unsupportedCount := 0

		for _, rule := range rules {
			supported, reason := rule.IsSupported()
			status := "✓"
			details := fmt.Sprintf("(%s)", rule.Model.CorrelationType)

			if !supported {
				status = "⚠"
				details = fmt.Sprintf("(%s)", reason)
				unsupportedCount++
			} else {
				supportedCount++
			}

			fmt.Printf("  %s %s - %s %s\n", status, rule.Name, rule.View.Title, details)

			if supported && len(rule.View.MITREAttack.Techniques) > 0 {
				fmt.Printf("      MITRE: %s\n", strings.Join(rule.View.MITREAttack.Techniques, ", "))
			}
		}

		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Supported: %d\n", supportedCount)
		fmt.Printf("  Not yet supported: %d\n", unsupportedCount)

		return nil
	},
}

var seederValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate seeder configuration",
	Long:  "Check if the seeder configuration file is valid without running the seeder",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := seeder.LoadConfig(seederCfgFile)
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Println("Configuration is valid:")
		fmt.Printf("  Version: %s\n", config.Version)
		fmt.Printf("  HEC URL: %s\n", config.Defaults.HECURL)
		fmt.Printf("  Event count: %d\n", config.Defaults.Count)
		fmt.Printf("  Time spread: %v\n", config.Defaults.TimeSpread)
		fmt.Printf("  Batch size: %d\n", config.Defaults.BatchSize)
		fmt.Printf("  Event types: %v\n", config.Defaults.EventTypes)

		if len(config.Attacks) > 0 {
			fmt.Printf("\nConfigured attacks:\n")
			for name, attack := range config.Attacks {
				status := "disabled"
				if attack.Enabled {
					status = "enabled"
				}
				fmt.Printf("  - %s (%s): %s\n", name, attack.Pattern, status)
			}
		}

		if len(config.Rules) > 0 {
			fmt.Printf("\nConfigured rules:\n")
			for name, rule := range config.Rules {
				status := "disabled"
				if rule.Enabled {
					status = "enabled"
				}
				fmt.Printf("  - %s (%s): multiplier=%.1fx, %s\n", name, rule.RuleFile, rule.Multiplier, status)
			}
		}

		return nil
	},
}

func init() {
	// Add seeder command to root
	rootCmd.AddCommand(seederCmd)

	// Add subcommands
	seederCmd.AddCommand(seederRunCmd)
	seederCmd.AddCommand(seederListCmd)
	seederCmd.AddCommand(seederListRulesCmd)
	seederCmd.AddCommand(seederValidateCmd)

	// Common flags
	seederCmd.PersistentFlags().StringVar(&seederCfgFile, "config", "", "config file (default: ./seeder.yaml or ~/.thawk/seeder.yaml)")

	// Run command flags
	seederRunCmd.Flags().StringVar(&seederHECURL, "hec-url", "", "HEC endpoint URL")
	seederRunCmd.Flags().StringVarP(&seederToken, "token", "t", "", "HEC authentication token")
	seederRunCmd.Flags().IntVarP(&seederCount, "count", "c", 0, "Number of events to generate")
	seederRunCmd.Flags().StringVarP(&seederTimeSpread, "time-spread", "s", "", "Time period to spread events (e.g., 24h, 7d, 90d)")
	seederRunCmd.Flags().IntVarP(&seederBatchSize, "batch-size", "b", 0, "Number of events per batch")
	seederRunCmd.Flags().StringVar(&seederEventTypes, "types", "", "Comma-separated event types")
	seederRunCmd.Flags().StringVarP(&seederAttacks, "attack", "a", "", "Comma-separated attack names from config")
	seederRunCmd.Flags().StringVar(&seederFromRules, "from-rules", "", "Directory containing detection rule JSON files")

	// Validate command flags
	seederValidateCmd.Flags().StringVar(&seederCfgFile, "config", "", "config file to validate")
}

func runSeeder(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := seeder.LoadConfig(seederCfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with flags if provided
	if cmd.Flags().Changed("hec-url") {
		config.Defaults.HECURL = seederHECURL
	}
	if cmd.Flags().Changed("token") {
		config.Defaults.Token = seederToken
	}
	if cmd.Flags().Changed("count") {
		config.Defaults.Count = seederCount
	}
	if cmd.Flags().Changed("time-spread") {
		duration, err := parseDuration(seederTimeSpread)
		if err != nil {
			return fmt.Errorf("invalid time-spread: %w", err)
		}
		config.Defaults.TimeSpread = duration
	}
	if cmd.Flags().Changed("batch-size") {
		config.Defaults.BatchSize = seederBatchSize
	}
	if cmd.Flags().Changed("types") {
		config.Defaults.EventTypes = strings.Split(seederEventTypes, ",")
	}

	// Validate token is provided
	if config.Defaults.Token == "" {
		return fmt.Errorf("HEC token is required (use --token flag or set in config)")
	}

	// Parse attack selection
	var selectedAttacks []string
	if seederAttacks != "" {
		selectedAttacks = strings.Split(seederAttacks, ",")
		for i := range selectedAttacks {
			selectedAttacks[i] = strings.TrimSpace(selectedAttacks[i])
		}
	}

	// Create and run seeder
	runner := seeder.NewRunner(config)

	// If --from-rules is specified, load and process rules
	if seederFromRules != "" {
		if err := runner.RunFromRulesDirectory(seederFromRules); err != nil {
			return fmt.Errorf("failed to run from rules: %w", err)
		}
		return nil
	}

	if err := runner.Run(selectedAttacks); err != nil {
		return fmt.Errorf("seeder failed: %w", err)
	}

	return nil
}

// parseDuration parses duration strings like "24h", "7d", "90d"
func parseDuration(s string) (time.Duration, error) {
	// Handle day suffix
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, err
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}

	// Use standard time.ParseDuration for other formats
	return time.ParseDuration(s)
}
