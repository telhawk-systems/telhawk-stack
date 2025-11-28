package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/telhawk-systems/telhawk-stack/common/config"
)

var (
	cfg *config.CLIConfig
)

var rootCmd = &cobra.Command{
	Use:   "thawk",
	Short: "TelHawk Stack CLI",
	Long: `thawk is the command-line interface for TelHawk Stack SIEM.

Manage authentication, users, search security data, configure alerts,
and interact with all TelHawk Stack services from your terminal.`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("profile", "default", "profile to use")
	rootCmd.PersistentFlags().String("output", "table", "output format: table, json, yaml")
}

func initConfig() {
	var err error
	cfg, err = config.LoadCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load config: %v\n", err)
		cfg = config.DefaultCLI()
	}
}
