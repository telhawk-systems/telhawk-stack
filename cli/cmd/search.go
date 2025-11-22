package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search security events",
	Long:  "Execute SPL-compatible search queries against TelHawk Stack",
	Example: `  # SPL-style query
  thawk search "index=* | stats count by source"
  thawk search "index=security severity=high" --last 1h
  thawk search "* | head 10" --output json

  # Raw JSON query from stdin
  echo '{"filter":{"class_uid":3002}}' | thawk search --raw

  # Raw JSON query from file
  thawk search --raw < query.json
  cat query.json | thawk search --raw`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("not logged in: %w", err)
		}

		baseURL, _ := cmd.Flags().GetString("url")
		queryClient := client.NewQueryClient(baseURL)

		raw, _ := cmd.Flags().GetBool("raw")
		var results []map[string]interface{}

		if raw {
			// Read raw JSON from stdin
			jsonData, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			if len(jsonData) == 0 {
				return fmt.Errorf("no JSON provided on stdin")
			}

			results, err = queryClient.RawQuery(p.AccessToken, jsonData)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}
		} else {
			// SPL-style query
			if len(args) < 1 {
				return fmt.Errorf("query argument required (or use --raw for JSON input)")
			}
			query := args[0]
			earliest, _ := cmd.Flags().GetString("earliest")
			latest, _ := cmd.Flags().GetString("latest")
			last, _ := cmd.Flags().GetString("last")

			results, err = queryClient.Search(p.AccessToken, query, earliest, latest, last)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}
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
	searchCmd.Flags().String("url", "http://localhost:3000", "Web backend URL")
	searchCmd.Flags().Bool("raw", false, "Read raw JSON query from stdin")
}
