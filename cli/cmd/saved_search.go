package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/cli/internal/client"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/output"
)

func init() {
	// attach as subcommands to existing `search` command to support `thawk search save` style
	searchCmd.AddCommand(savedSaveCmd)
	searchCmd.AddCommand(savedListCmd)
	searchCmd.AddCommand(savedShowCmd)
	searchCmd.AddCommand(savedUpdateCmd)
	searchCmd.AddCommand(savedDeleteCmd)
	searchCmd.AddCommand(savedRunCmd)
	searchCmd.AddCommand(savedDisableCmd)
	searchCmd.AddCommand(savedEnableCmd)
	searchCmd.AddCommand(savedHideCmd)
	searchCmd.AddCommand(savedUnhideCmd)

	// shared flags
	for _, c := range []*cobra.Command{savedSaveCmd, savedListCmd, savedShowCmd, savedUpdateCmd, savedDeleteCmd, savedRunCmd, savedDisableCmd, savedEnableCmd, savedHideCmd, savedUnhideCmd} {
		c.Flags().String("query-url", "http://localhost:8082", "Query service URL")
		c.Flags().String("profile", "default", "profile to use")
		c.Flags().String("output", "table", "output format: table, json, yaml")
	}

	savedSaveCmd.Flags().String("name", "", "Saved search name")
	savedSaveCmd.Flags().String("file", "", "Path to JSON DSL file")
	savedSaveCmd.Flags().String("query", "", "Inline JSON DSL string")
	savedSaveCmd.Flags().Bool("global", false, "Create as global (admin only)")

	savedListCmd.Flags().Bool("show-all", false, "Include hidden searches")

	savedUpdateCmd.Flags().String("name", "", "New name")
	savedUpdateCmd.Flags().String("file", "", "Path to JSON DSL file")
	savedUpdateCmd.Flags().String("query", "", "Inline JSON DSL string")
}

var savedSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save a query (JSON DSL)",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		qstr, err := cmd.Flags().GetString("query")
		if err != nil {
			return fmt.Errorf("failed to get query: %w", err)
		}
		file, err := cmd.Flags().GetString("file")
		if err != nil {
			return fmt.Errorf("failed to get file: %w", err)
		}
		isGlobal, err := cmd.Flags().GetBool("global")
		if err != nil {
			return fmt.Errorf("failed to get global flag: %w", err)
		}
		var q map[string]interface{}
		if file != "" {
			b, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &q); err != nil {
				return fmt.Errorf("invalid JSON in --file: %w", err)
			}
		} else if qstr != "" {
			if err := json.Unmarshal([]byte(qstr), &q); err != nil {
				return fmt.Errorf("invalid JSON in --query: %w", err)
			}
		} else {
			return fmt.Errorf("provide --file or --query with JSON DSL")
		}
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		saved, err := qc.SavedSearchCreate(p.AccessToken, queryURL, name, q, map[string]interface{}{}, isGlobal)
		if err != nil {
			return err
		}
		output.Success("Saved search created: %s (%s)", saved.Name, saved.ID)
		return nil
	},
}

var savedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved searches",
	RunE: func(cmd *cobra.Command, args []string) error {
		showAll, err := cmd.Flags().GetBool("show-all")
		if err != nil {
			return fmt.Errorf("failed to get show-all flag: %w", err)
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		items, err := qc.SavedSearchList(queryURL, showAll)
		if err != nil {
			return err
		}
		out, _ := cmd.Flags().GetString("output")
		if out == "json" {
			return output.JSON(items)
		}
		tbl := output.NewTable([]string{"ID", "Name", "State", "Created"})
		for _, s := range items {
			state := "active"
			if s.HiddenAt != nil {
				state = "hidden"
			} else if s.DisabledAt != nil {
				state = "disabled"
			}
			tbl.AddRow([]string{s.ID, s.Name, state, s.CreatedAt})
		}
		tbl.Render()
		return nil
	},
}

var savedShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a saved search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		s, err := qc.SavedSearchGet(queryURL, id)
		if err != nil {
			return err
		}
		return output.JSON(s)
	},
}

var savedUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a saved search (new version)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		attrs := map[string]interface{}{}
		if name, _ := cmd.Flags().GetString("name"); name != "" {
			attrs["name"] = name
		}
		if file, _ := cmd.Flags().GetString("file"); file != "" {
			b, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			var q map[string]interface{}
			if err := json.Unmarshal(b, &q); err != nil {
				return err
			}
			attrs["query"] = q
		}
		if qstr, _ := cmd.Flags().GetString("query"); qstr != "" {
			var q map[string]interface{}
			if err := json.Unmarshal([]byte(qstr), &q); err != nil {
				return err
			}
			attrs["query"] = q
		}
		if len(attrs) == 0 {
			return fmt.Errorf("no changes provided")
		}
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		s, err := qc.SavedSearchUpdate(p.AccessToken, queryURL, id, attrs)
		if err != nil {
			return err
		}
		output.Success("Updated: %s (%s)", s.Name, s.VersionID)
		return nil
	},
}

var savedDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Hide a saved search (immutable)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		_, err = qc.SavedSearchAction(p.AccessToken, queryURL, id, "hide")
		if err != nil {
			return err
		}
		output.Success("Hidden: %s", id)
		return nil
	},
}

var savedRunCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "Run a saved search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		// Use generic action to trigger run and print raw response
		// For now, call endpoint directly via HTTP for simplicity
		qcli := client.NewQueryClient(queryURL)
		req, _ := http.NewRequest("POST", queryURL+"/api/v1/saved-searches/"+id+"/run", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+p.AccessToken)
		req.Header.Set("Accept", "application/json")
		resp, err := qcli.Client().Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 409 {
			return fmt.Errorf("search disabled")
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("run failed: %d", resp.StatusCode)
		}
		var v map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
			return err
		}
		return output.JSON(v)
	},
}

var savedDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disable a saved search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		_, err = qc.SavedSearchAction(p.AccessToken, queryURL, id, "disable")
		if err != nil {
			return err
		}
		output.Success("Disabled: %s", id)
		return nil
	},
}

var savedEnableCmd = &cobra.Command{
	Use:   "enable <id>",
	Short: "Enable a saved search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		_, err = qc.SavedSearchAction(p.AccessToken, queryURL, id, "enable")
		if err != nil {
			return err
		}
		output.Success("Enabled: %s", id)
		return nil
	},
}

var savedHideCmd = &cobra.Command{
	Use:   "hide <id>",
	Short: "Hide a saved search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		_, err = qc.SavedSearchAction(p.AccessToken, queryURL, id, "hide")
		if err != nil {
			return err
		}
		output.Success("Hidden: %s", id)
		return nil
	},
}

var savedUnhideCmd = &cobra.Command{
	Use:   "unhide <id>",
	Short: "Unhide a saved search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		profile, _ := cmd.Flags().GetString("profile")
		p, err := cfg.GetProfile(profile)
		if err != nil {
			return err
		}
		queryURL, _ := cmd.Flags().GetString("query-url")
		qc := client.NewQueryClient(queryURL)
		_, err = qc.SavedSearchAction(p.AccessToken, queryURL, id, "unhide")
		if err != nil {
			return err
		}
		output.Success("Unhidden: %s", id)
		return nil
	},
}
