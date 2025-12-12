package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	// Service definitions with ports
	serviceDefinitions = []struct {
		Name string
		Port int
		Desc string
	}{
		{"bayes", 9120, "Logging Service"},
		{"russell", 9100, "Service Discovery"},
		{"turing", 9200, "LLM Management"},
		{"hypatia", 9220, "RAG Service"},
		{"babbage", 9150, "NLP Service"},
		{"platon", 9130, "Pipeline Processing"},
		{"leibniz", 9140, "Agentic AI"},
		{"aristoteles", 9160, "Agentic Pipeline"},
		{"kant", 8080, "API Gateway"},
	}
)

var startCmd = &cobra.Command{
	Use:   "start [service...]",
	Short: "Startet Services im Hintergrund",
	Long: `Startet einen oder mehrere mDW Services als Hintergrundprozesse.

Ohne Argument werden alle Services gestartet.
Mit Argumenten werden nur die angegebenen Services gestartet.

Services:
  kant        - API Gateway (HTTP :8080)
  russell     - Service Discovery (gRPC :9100)
  turing      - LLM Management (gRPC :9200)
  hypatia     - RAG Service (gRPC :9220)
  babbage     - NLP Service (gRPC :9150)
  leibniz     - Agentic AI (gRPC :9140)
  platon      - Pipeline Processing (gRPC :9130)
  aristoteles - Agentic Pipeline (gRPC :9160)
  bayes       - Logging Service (gRPC :9120)

Beispiele:
  mdw start              # Alle Services starten
  mdw start turing       # Nur Turing starten
  mdw start turing kant  # Turing und Kant starten`,
	ValidArgs: []string{"kant", "russell", "turing", "hypatia", "leibniz", "babbage", "bayes", "platon", "aristoteles"},
	RunE:      runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop [service...]",
	Short: "Stoppt laufende Services",
	Long: `Stoppt einen oder mehrere laufende mDW Services.

Ohne Argument werden alle Services gestoppt.
Mit Argumenten werden nur die angegebenen Services gestoppt.

Beispiele:
  mdw stop              # Alle Services stoppen
  mdw stop turing       # Nur Turing stoppen
  mdw stop turing kant  # Turing und Kant stoppen`,
	ValidArgs: []string{"kant", "russell", "turing", "hypatia", "leibniz", "babbage", "bayes", "platon", "aristoteles"},
	RunE:      runStop,
}

var restartCmd = &cobra.Command{
	Use:   "restart [service...]",
	Short: "Startet Services neu",
	Long: `Startet einen oder mehrere mDW Services neu.

Ohne Argument werden alle Services neu gestartet.
Mit Argumenten werden nur die angegebenen Services neu gestartet.

Beispiele:
  mdw restart              # Alle Services neu starten
  mdw restart turing       # Nur Turing neu starten`,
	ValidArgs: []string{"kant", "russell", "turing", "hypatia", "leibniz", "babbage", "bayes", "platon", "aristoteles"},
	RunE:      runRestart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
}

// getPidDir returns the directory for PID files
func getPidDir() string {
	// Use XDG_RUNTIME_DIR if available, otherwise /tmp
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "mdw")
	}
	return filepath.Join(os.TempDir(), "mdw")
}

// getPidFile returns the PID file path for a service
func getPidFile(service string) string {
	return filepath.Join(getPidDir(), fmt.Sprintf("%s.pid", service))
}

// getLogFile returns the log file path for a service
func getLogFile(service string) string {
	logDir := filepath.Join(getPidDir(), "logs")
	return filepath.Join(logDir, fmt.Sprintf("%s.log", service))
}

// ensureDirs ensures PID and log directories exist
func ensureDirs() error {
	pidDir := getPidDir()
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return err
	}
	logDir := filepath.Join(pidDir, "logs")
	return os.MkdirAll(logDir, 0755)
}

// isServiceRunning checks if a service is running by reading its PID file
func isServiceRunning(service string) (bool, int) {
	pidFile := getPidFile(service)
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, 0
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist, clean up PID file
		os.Remove(pidFile)
		return false, 0
	}

	return true, pid
}

// startService starts a single service in the background
func startService(service string) error {
	// Check if already running
	if running, pid := isServiceRunning(service); running {
		return fmt.Errorf("bereits gestartet (PID %d)", pid)
	}

	// Ensure directories exist
	if err := ensureDirs(); err != nil {
		return fmt.Errorf("Verzeichnis-Fehler: %v", err)
	}

	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Executable nicht gefunden: %v", err)
	}

	// Create log file
	logFile := getLogFile(service)
	logFd, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("Log-Datei Fehler: %v", err)
	}

	// Start the service using 'mdw serve <service>'
	cmd := exec.Command(executable, "serve", service)
	cmd.Stdout = logFd
	cmd.Stderr = logFd
	cmd.Stdin = nil

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		logFd.Close()
		return fmt.Errorf("Start fehlgeschlagen: %v", err)
	}

	// Write PID file
	pidFile := getPidFile(service)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		logFd.Close()
		return fmt.Errorf("PID-Datei Fehler: %v", err)
	}

	// Detach - don't wait for child
	go func() {
		cmd.Wait()
		logFd.Close()
	}()

	return nil
}

// stopService stops a single service
func stopService(service string) error {
	running, pid := isServiceRunning(service)
	if !running {
		return fmt.Errorf("nicht gestartet")
	}

	// Find and kill the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("Prozess nicht gefunden: %v", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("Stop fehlgeschlagen: %v", err)
	}

	// Remove PID file
	os.Remove(getPidFile(service))

	return nil
}

// getServicesToManage returns the list of services to start/stop
func getServicesToManage(args []string) []string {
	if len(args) == 0 {
		// Return all services in order
		services := make([]string, len(serviceDefinitions))
		for i, svc := range serviceDefinitions {
			services[i] = svc.Name
		}
		return services
	}
	return args
}

func runStart(cmd *cobra.Command, args []string) error {
	services := getServicesToManage(args)

	fmt.Println("meinDENKWERK - Services starten")
	fmt.Println(strings.Repeat("=", 40))

	successCount := 0
	for _, service := range services {
		fmt.Printf("  %s: ", service)
		if err := startService(service); err != nil {
			fmt.Printf("FEHLER - %v\n", err)
		} else {
			fmt.Printf("gestartet\n")
			successCount++
		}
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Gestartet: %d/%d Services\n", successCount, len(services))

	if successCount > 0 {
		fmt.Println()
		fmt.Println("Log-Dateien:", filepath.Join(getPidDir(), "logs"))
		fmt.Println("Status prüfen: mdw status")
	}

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	services := getServicesToManage(args)

	// Stop in reverse order
	if len(args) == 0 {
		for i, j := 0, len(services)-1; i < j; i, j = i+1, j-1 {
			services[i], services[j] = services[j], services[i]
		}
	}

	fmt.Println("meinDENKWERK - Services stoppen")
	fmt.Println(strings.Repeat("=", 40))

	successCount := 0
	for _, service := range services {
		fmt.Printf("  %s: ", service)
		if err := stopService(service); err != nil {
			fmt.Printf("FEHLER - %v\n", err)
		} else {
			fmt.Printf("gestoppt\n")
			successCount++
		}
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Gestoppt: %d/%d Services\n", successCount, len(services))

	return nil
}

func runRestart(cmd *cobra.Command, args []string) error {
	services := getServicesToManage(args)

	fmt.Println("meinDENKWERK - Services neu starten")
	fmt.Println(strings.Repeat("=", 40))

	for _, service := range services {
		fmt.Printf("  %s: ", service)

		// Stop if running
		if running, _ := isServiceRunning(service); running {
			if err := stopService(service); err != nil {
				fmt.Printf("Stop-Fehler - %v\n", err)
				continue
			}
			fmt.Print("gestoppt -> ")
		}

		// Start
		if err := startService(service); err != nil {
			fmt.Printf("Start-Fehler - %v\n", err)
		} else {
			fmt.Printf("gestartet\n")
		}
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Status prüfen: mdw status")

	return nil
}

// ShowServiceLogs shows the last lines of a service log
func ShowServiceLogs(service string, lines int) error {
	logFile := getLogFile(service)

	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("Log-Datei nicht gefunden: %v", err)
	}
	defer f.Close()

	// Simple tail implementation
	content, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	allLines := strings.Split(string(content), "\n")
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	for _, line := range allLines[start:] {
		fmt.Println(line)
	}

	return nil
}
