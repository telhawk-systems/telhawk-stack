package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Data ingestion commands",
	Long:  "Send events to TelHawk Stack ingestion service",
}

var ingestSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an event",
	Long:  "Send a single event to the ingestion service",
	Example: `  thawk ingest send --message "Security alert" --severity high
  thawk ingest send --json '{"event":"login","user":"admin"}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		message, _ := cmd.Flags().GetString("message")
		jsonData, _ := cmd.Flags().GetString("json")
		source, _ := cmd.Flags().GetString("source")
		sourcetype, _ := cmd.Flags().GetString("sourcetype")
		hecToken, _ := cmd.Flags().GetString("token")

		if message == "" && jsonData == "" {
			return fmt.Errorf("either --message or --json is required")
		}

		if hecToken == "" {
			return fmt.Errorf("HEC token is required (use --token or create with 'thawk token create')")
		}

		ingestURL, _ := cmd.Flags().GetString("ingest-url")
		ingestClient := client.NewIngestClient(ingestURL)

		var event map[string]interface{}
		if jsonData != "" {
			// Parse JSON
			event = map[string]interface{}{"raw": jsonData}
		} else {
			event = map[string]interface{}{
				"message": message,
				"source":  source,
			}
		}

		if err := ingestClient.SendEvent(hecToken, event, source, sourcetype); err != nil {
			return fmt.Errorf("failed to send event: %w", err)
		}

		output.Success("Event sent successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.AddCommand(ingestSendCmd)

	ingestSendCmd.Flags().StringP("message", "m", "", "Event message")
	ingestSendCmd.Flags().String("json", "", "JSON event data")
	ingestSendCmd.Flags().String("source", "thawk-cli", "Event source")
	ingestSendCmd.Flags().String("sourcetype", "manual", "Event source type")
	ingestSendCmd.Flags().StringP("token", "t", "", "HEC token")
	ingestSendCmd.Flags().String("ingest-url", "http://localhost:8088", "Ingest service URL")
}
