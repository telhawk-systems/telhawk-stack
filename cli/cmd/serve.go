package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the TelHawk service manager daemon",
	Long:  `Starts a daemon that manages all TelHawk services with auto-restart on crash`,
	RunE:  runServe,
}

var (
	startCmd = &cobra.Command{
		Use:   "start [service]",
		Short: "Start a service or all services",
		Long:  `Start a specific service or all services if no service name provided`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runStart,
	}

	stopCmd = &cobra.Command{
		Use:   "stop [service]",
		Short: "Stop a service or all services",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runStop,
	}

	restartCmd = &cobra.Command{
		Use:   "restart [service]",
		Short: "Restart a service or all services",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRestart,
	}

	statusCmd = &cobra.Command{
		Use:   "status [service]",
		Short: "Show status of services",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runStatus,
	}
)

const (
	commandFile = "/tmp/telhawk/thawk.in"
	outputFile  = "/tmp/telhawk/thawk.out"
	logDir      = "/var/log/telhawk"
)

type serviceConfig struct {
	name      string
	dir       string
	binary    string
	cmd       string
	port      int
	dependsOn []string
}

var services = []serviceConfig{
	{name: "authenticate", dir: "/app/authenticate", binary: "/app/bin/authenticate", cmd: "/app/authenticate/cmd/authenticate", port: 9080},
	{name: "ingest", dir: "/app/ingest", binary: "/app/bin/ingest", cmd: "/app/ingest/cmd/ingest", port: 9088, dependsOn: []string{"authenticate"}},
	{name: "search", dir: "/app/search", binary: "/app/bin/search", cmd: "/app/search/cmd/search", port: 9082, dependsOn: []string{"authenticate"}},
	{name: "respond", dir: "/app/respond", binary: "/app/bin/respond", cmd: "/app/respond/cmd/respond", port: 9085, dependsOn: []string{"authenticate"}},
	{name: "web", dir: "/app/web/backend", binary: "/app/bin/web", cmd: "/app/web/backend/cmd/web", port: 80, dependsOn: []string{"authenticate", "search", "respond"}},
	{name: "frontend", dir: "/app/web/frontend", binary: "npm", cmd: "", port: 5173}, // npm service (no binary build)
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	log.SetOutput(os.Stdout)
	log.SetPrefix("[thawk-serve] ")

	log.Println("TelHawk Service Manager starting...")

	// Ensure directories exist
	os.MkdirAll("/tmp/telhawk", 0755)
	os.MkdirAll(logDir, 0755)

	// Service state
	processes := make(map[string]*os.Process)

	// Watch for commands
	go watchCommands(processes)

	// Monitor services (skip auto-restart for now - causing issues)
	for {
		time.Sleep(5 * time.Second)
		// TODO: Fix auto-restart logic - currently causes infinite spawning
	}
}

func watchCommands(processes map[string]*os.Process) {
	for {
		if _, err := os.Stat(commandFile); err == nil {
			// Read command
			data, err := os.ReadFile(commandFile)
			if err != nil {
				log.Printf("Error reading command: %v\n", err)
				os.Remove(commandFile)
				continue
			}

			parts := strings.Fields(string(data))
			if len(parts) == 0 {
				os.Remove(commandFile)
				continue
			}

			command := parts[0]
			service := "all"
			if len(parts) > 1 {
				service = parts[1]
			}

			// Execute command
			response := executeCommand(command, service, processes)

			// Write response
			os.WriteFile(outputFile, []byte(response), 0644)

			// Delete command file
			os.Remove(commandFile)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func executeCommand(command, service string, processes map[string]*os.Process) string {
	switch command {
	case "start":
		if service == "all" {
			return startAllServices(processes)
		}
		return startService(service, processes)
	case "stop":
		if service == "all" {
			return stopAllServices(processes)
		}
		return stopService(service, processes)
	case "restart":
		if service == "all" {
			stopAllServices(processes)
			return startAllServices(processes)
		}
		stopService(service, processes)
		return startService(service, processes)
	case "status":
		if service == "all" {
			return statusAllServices(processes)
		}
		return statusService(service, processes)
	default:
		return fmt.Sprintf("Unknown command: %s\n", command)
	}
}

func startAllServices(processes map[string]*os.Process) string {
	var output strings.Builder
	for _, svc := range services {
		result := startService(svc.name, processes)
		output.WriteString(result)
	}
	return output.String()
}

func startService(name string, processes map[string]*os.Process) string {
	// Find service config
	var svc *serviceConfig
	for i := range services {
		if services[i].name == name {
			svc = &services[i]
			break
		}
	}
	if svc == nil {
		return fmt.Sprintf("Unknown service: %s\n", name)
	}

	// Kill existing process
	if proc, exists := processes[name]; exists && proc != nil {
		proc.Kill()
		proc.Wait()
		delete(processes, name)
		time.Sleep(500 * time.Millisecond)
	}

	// Open log file
	logFile := filepath.Join(logDir, name+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Failed to open log file for %s: %v\n", name, err)
	}

	var runCmd *exec.Cmd

	// Handle different service types
	if name == "frontend" {
		// Frontend: npm run dev
		fmt.Fprintf(f, "\n=== Starting frontend at %s ===\n", time.Now().Format(time.RFC3339))
		runCmd = exec.Command("npm", "run", "dev", "--", "--host", "0.0.0.0")
		runCmd.Dir = svc.dir
	} else {
		// Go services: build then run
		fmt.Fprintf(f, "\n=== Building %s at %s ===\n", name, time.Now().Format(time.RFC3339))

		buildCmd := exec.Command("go", "build", "-buildvcs=false", "-o", svc.binary, svc.cmd)
		buildCmd.Dir = svc.dir
		buildCmd.Stdout = f
		buildCmd.Stderr = f

		if err := buildCmd.Run(); err != nil {
			fmt.Fprintf(f, "Build failed: %v\n", err)
			f.Close()
			return fmt.Sprintf("Build failed for %s, check %s\n", name, logFile)
		}

		fmt.Fprintf(f, "=== Starting %s at %s ===\n", name, time.Now().Format(time.RFC3339))
		runCmd = exec.Command(svc.binary)
		runCmd.Dir = svc.dir
	}

	runCmd.Stdout = f
	runCmd.Stderr = f
	runCmd.Env = os.Environ() // Inherit all environment variables

	if err := runCmd.Start(); err != nil {
		f.Close()
		return fmt.Sprintf("Failed to start %s: %v\n", name, err)
	}

	processes[name] = runCmd.Process

	// Don't close file - keep it open for logs

	return fmt.Sprintf("Started %s (PID %d)\n", name, runCmd.Process.Pid)
}

func stopService(name string, processes map[string]*os.Process) string {
	proc, exists := processes[name]
	if !exists || proc == nil {
		return fmt.Sprintf("Service %s is not running\n", name)
	}

	proc.Kill()
	proc.Wait()
	delete(processes, name)

	return fmt.Sprintf("Stopped %s\n", name)
}

func stopAllServices(processes map[string]*os.Process) string {
	var output strings.Builder
	for name := range processes {
		result := stopService(name, processes)
		output.WriteString(result)
	}
	return output.String()
}

func restartService(name string, processes map[string]*os.Process) {
	stopService(name, processes)
	time.Sleep(500 * time.Millisecond)
	startService(name, processes)
}

func statusService(name string, processes map[string]*os.Process) string {
	proc, exists := processes[name]
	if !exists || proc == nil {
		return fmt.Sprintf("%s: stopped\n", name)
	}

	// Check if still alive
	err := proc.Signal(os.Signal(nil))
	if err != nil {
		return fmt.Sprintf("%s: dead (PID %d)\n", name, proc.Pid)
	}

	return fmt.Sprintf("%s: running (PID %d)\n", name, proc.Pid)
}

func statusAllServices(processes map[string]*os.Process) string {
	var output strings.Builder
	for _, svc := range services {
		result := statusService(svc.name, processes)
		output.WriteString(result)
	}
	return output.String()
}

// Client command functions
func sendCommand(command, service string) error {
	// Ensure directory exists
	os.MkdirAll("/tmp/telhawk", 0755)

	// Write command
	cmdStr := command
	if service != "" {
		cmdStr = command + " " + service
	}

	if err := os.WriteFile(commandFile, []byte(cmdStr), 0644); err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	// Wait for response (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for response")
		case <-ticker.C:
			if _, err := os.Stat(outputFile); err == nil {
				data, err := os.ReadFile(outputFile)
				if err != nil {
					return fmt.Errorf("failed to read response: %w", err)
				}
				os.Remove(outputFile)
				fmt.Print(string(data))
				return nil
			}
		}
	}
}

func runStart(cmd *cobra.Command, args []string) error {
	service := "all"
	if len(args) > 0 {
		service = args[0]
	}
	return sendCommand("start", service)
}

func runStop(cmd *cobra.Command, args []string) error {
	service := "all"
	if len(args) > 0 {
		service = args[0]
	}
	return sendCommand("stop", service)
}

func runRestart(cmd *cobra.Command, args []string) error {
	service := "all"
	if len(args) > 0 {
		service = args[0]
	}
	return sendCommand("restart", service)
}

func runStatus(cmd *cobra.Command, args []string) error {
	service := "all"
	if len(args) > 0 {
		service = args[0]
	}
	return sendCommand("status", service)
}
