package cmd

import (
	"testing"

	"github.com/telhawk-systems/telhawk-stack/common/config"
)

// Test command initialization and registration
func TestCommandsRegistered(t *testing.T) {
	// Setup config
	cfg = config.DefaultCLI()

	// Verify root command exists
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	// Check that all main commands are registered
	commands := rootCmd.Commands()
	expectedCommands := map[string]bool{
		"login":  false,
		"logout": false,
		"whoami": false,
		"alerts": false,
		"rules":  false,
		"search": false,
		"ingest": false,
		"token":  false,
		"user":   false,
		"seeder": false,
	}

	for _, cmd := range commands {
		// Extract command name (handles "search [query]" -> "search")
		cmdName := cmd.Use
		for key := range expectedCommands {
			if len(cmdName) >= len(key) && cmdName[:len(key)] == key {
				expectedCommands[key] = true
				break
			}
		}
	}

	for cmdName, found := range expectedCommands {
		if !found {
			t.Errorf("expected command '%s' to be registered with root command", cmdName)
		}
	}
}

func TestLoginCommandExists(t *testing.T) {
	if loginCmd == nil {
		t.Fatal("loginCmd should not be nil")
	}

	// Verify login is a root-level command
	commands := rootCmd.Commands()
	found := false
	for _, cmd := range commands {
		if cmd.Use == "login" {
			found = true
			break
		}
	}
	if !found {
		t.Error("login command should be registered as a root-level command")
	}
}

func TestAlertsCommandHasSubcommands(t *testing.T) {
	if alertsCmd == nil {
		t.Fatal("alertsCmd should not be nil")
	}

	// Alerts should have subcommands including cases
	subcommands := alertsCmd.Commands()
	hasGet := false
	hasList := false
	hasCases := false

	for _, cmd := range subcommands {
		switch {
		case cmd.Use == "list" || cmd.Use == "ls":
			hasList = true
		case len(cmd.Use) >= 3 && cmd.Use[:3] == "get":
			hasGet = true
		case cmd.Use == "cases":
			hasCases = true
		}
	}

	if !hasList {
		t.Error("alerts command should have 'list' subcommand")
	}
	if !hasGet {
		t.Error("alerts command should have 'get' subcommand")
	}
	if !hasCases {
		t.Error("alerts command should have 'cases' subcommand")
	}
}

func TestRulesCommandHasSubcommands(t *testing.T) {
	if rulesCmd == nil {
		t.Fatal("rulesCmd should not be nil")
	}

	subcommands := rulesCmd.Commands()
	expectedCommands := map[string]bool{
		"list":     false,
		"get":      false,
		"create":   false,
		"disable":  false,
		"enable":   false,
		"versions": false,
	}

	for _, cmd := range subcommands {
		// Extract command name (handles "get [id]" -> "get")
		cmdName := cmd.Use
		for key := range expectedCommands {
			if len(cmdName) >= len(key) && cmdName[:len(key)] == key {
				expectedCommands[key] = true
			}
		}
	}

	for cmdName, found := range expectedCommands {
		if !found {
			t.Errorf("rules command should have '%s' subcommand", cmdName)
		}
	}
}

func TestGlobalFlags(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	// Check that global flags exist
	flags := []string{"output", "profile"}
	for _, flagName := range flags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected global flag '%s' to be defined", flagName)
		}
	}
}

func TestLoginCommandFlags(t *testing.T) {
	if loginCmd == nil {
		t.Fatal("loginCmd should not be nil")
	}

	// Check required flags
	requiredFlags := []string{"username", "password"}
	for _, flagName := range requiredFlags {
		flag := loginCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag '%s' to be defined on login command", flagName)
		}
	}

	// Check optional flags
	optionalFlags := []string{"auth-url"}
	for _, flagName := range optionalFlags {
		flag := loginCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag '%s' to be defined on login command", flagName)
		}
	}
}

func TestAlertsListCommandFlags(t *testing.T) {
	if alertsListCmd == nil {
		t.Fatal("alertsListCmd should not be nil")
	}

	// Check pagination and filter flags
	flags := []string{"page", "limit", "severity", "rule"}
	for _, flagName := range flags {
		flag := alertsListCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag '%s' to be defined on alerts list command", flagName)
		}
	}
}

func TestRulesListCommandFlags(t *testing.T) {
	if rulesListCmd == nil {
		t.Fatal("rulesListCmd should not be nil")
	}

	// Check pagination flags
	flags := []string{"page", "limit"}
	for _, flagName := range flags {
		flag := rulesListCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag '%s' to be defined on rules list command", flagName)
		}
	}
}

func TestCasesCommandHasSubcommands(t *testing.T) {
	if casesCmd == nil {
		t.Fatal("casesCmd should not be nil")
	}

	subcommands := casesCmd.Commands()
	expectedCommands := map[string]bool{
		"list":   false,
		"get":    false,
		"create": false,
		"close":  false,
		"reopen": false,
	}

	for _, cmd := range subcommands {
		cmdName := cmd.Use
		for key := range expectedCommands {
			if len(cmdName) >= len(key) && cmdName[:len(key)] == key {
				expectedCommands[key] = true
			}
		}
	}

	for cmdName, found := range expectedCommands {
		if !found {
			t.Errorf("cases command should have '%s' subcommand", cmdName)
		}
	}
}

func TestCasesCreateCommandFlags(t *testing.T) {
	if casesCreateCmd == nil {
		t.Fatal("casesCreateCmd should not be nil")
	}

	// Check required and optional flags
	flags := []string{"title", "description", "severity"}
	for _, flagName := range flags {
		flag := casesCreateCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag '%s' to be defined on cases create command", flagName)
		}
	}
}
