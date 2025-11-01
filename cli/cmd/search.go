package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search security events",
	Long:  "Execute SPL-compatible search queries against TelHawk Stack",
	Example: `  thawk search "index=* | stats count by source"
  thawk search "index=security severity=high" --last 1h
  thawk search "* | head 10" --output json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		earliest, _ := cmd.Flags().GetString("earliest")
		latest, _ := cmd.Flags().GetString("latest")
		last, _ := cmd.Flags().GetString("last")

		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		queryURL, _ := cmd.Flags().GetString("query-url")
		queryClient := client.NewQueryClient(queryURL)

		results, err := queryClient.Search(p.AccessToken, query, earliest, latest, last)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return output.JSON(results)
		}

		output.Success("Search completed: %d results", len(results))
		for i, result := range results {
			fmt.Printf("%d: %v\n", i+1, result)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().String("earliest", "", "Earliest time (e.g., -1h, -7d)")
	searchCmd.Flags().String("latest", "", "Latest time (e.g., now, -1h)")
	searchCmd.Flags().String("last", "", "Time range shorthand (e.g., 1h, 24h, 7d)")
	searchCmd.Flags().String("query-url", "http://localhost:8081", "Query service URL")
}
