// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     controlcenter
// Description: Dependency checking for mDW Control Center
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package controlcenter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Dependency represents a system dependency
type Dependency struct {
	Name        string
	Description string
	Required    bool
	Status      DependencyStatus
	Message     string
	Version     string
	CheckFn     func() (bool, string, string) // returns: ok, message, version
}

// DependencyStatus represents the status of a dependency
type DependencyStatus int

const (
	StatusUnchecked DependencyStatus = iota
	StatusChecking
	StatusOK
	StatusFailed
	StatusWarning
)

// String returns the status as string
func (s DependencyStatus) String() string {
	switch s {
	case StatusUnchecked:
		return "unchecked"
	case StatusChecking:
		return "checking..."
	case StatusOK:
		return "OK"
	case StatusFailed:
		return "FAILED"
	case StatusWarning:
		return "WARNING"
	default:
		return "unknown"
	}
}

// DependencyChecker checks system dependencies
type DependencyChecker struct {
	Dependencies []Dependency
}

// NewDependencyChecker creates a new dependency checker with all dependencies
func NewDependencyChecker() *DependencyChecker {
	return &DependencyChecker{
		Dependencies: []Dependency{
			{
				Name:        "Go Runtime",
				Description: "Go programming language",
				Required:    true,
				Status:      StatusUnchecked,
				CheckFn:     checkGo,
			},
			{
				Name:        "Ollama",
				Description: "Local LLM inference engine",
				Required:    true,
				Status:      StatusUnchecked,
				CheckFn:     checkOllama,
			},
			{
				Name:        "LLM Models",
				Description: "Available LLM models",
				Required:    false,
				Status:      StatusUnchecked,
				CheckFn:     checkOllamaModels,
			},
			{
				Name:        "Embedding Model",
				Description: "nomic-embed-text for RAG",
				Required:    false,
				Status:      StatusUnchecked,
				CheckFn:     checkEmbeddingModel,
			},
			{
				Name:        "Russell Service",
				Description: "Service discovery (auto-start)",
				Required:    true,
				Status:      StatusUnchecked,
				CheckFn:     checkRussellWithAutoStart,
			},
			{
				Name:        "gRPC Port 9200",
				Description: "Turing service port",
				Required:    false,
				Status:      StatusUnchecked,
				CheckFn:     makePortChecker(9200),
			},
			{
				Name:        "HTTP Port 8080",
				Description: "Kant API Gateway port",
				Required:    false,
				Status:      StatusUnchecked,
				CheckFn:     makePortChecker(8080),
			},
		},
	}
}

// CheckAll checks all dependencies
func (dc *DependencyChecker) CheckAll() {
	for i := range dc.Dependencies {
		dc.Dependencies[i].Status = StatusChecking
	}

	for i := range dc.Dependencies {
		dc.Check(i)
	}
}

// Check checks a single dependency by index
func (dc *DependencyChecker) Check(index int) {
	if index < 0 || index >= len(dc.Dependencies) {
		return
	}

	dep := &dc.Dependencies[index]
	dep.Status = StatusChecking

	if dep.CheckFn != nil {
		ok, message, version := dep.CheckFn()
		dep.Message = message
		dep.Version = version
		if ok {
			dep.Status = StatusOK
		} else if dep.Required {
			dep.Status = StatusFailed
		} else {
			dep.Status = StatusWarning
		}
	}
}

// AllRequiredOK returns true if all required dependencies are OK
func (dc *DependencyChecker) AllRequiredOK() bool {
	for _, dep := range dc.Dependencies {
		if dep.Required && dep.Status != StatusOK {
			return false
		}
	}
	return true
}

// Dependency check functions

func checkGo() (bool, string, string) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return false, "Go not found", ""
	}

	version := strings.TrimSpace(string(output))
	// Extract version number
	parts := strings.Fields(version)
	if len(parts) >= 3 {
		version = parts[2]
	}

	return true, "Installed", version
}

func checkOllama() (bool, string, string) {
	// First check if ollama binary exists
	cmd := exec.Command("ollama", "--version")
	output, err := cmd.Output()

	var version string
	if err == nil {
		version = strings.TrimSpace(string(output))
	}

	// Then check if the API is responsive
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/version", nil)
	if err != nil {
		return false, "Cannot create request", version
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "API not responding", version
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if v, ok := result["version"].(string); ok {
				version = v
			}
		}
		return true, "Running", version
	}

	return false, "API error", version
}

func checkOllamaModels() (bool, string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	if err != nil {
		return false, "Cannot check models", ""
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "Ollama not running", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, "Cannot fetch models", ""
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "Parse error", ""
	}

	count := len(result.Models)
	if count == 0 {
		return false, "No models installed", "0 models"
	}

	// Get first few model names
	var modelNames []string
	for i, m := range result.Models {
		if i >= 3 {
			break
		}
		modelNames = append(modelNames, m.Name)
	}

	modelList := strings.Join(modelNames, ", ")
	if count > 3 {
		modelList += "..."
	}

	return true, fmt.Sprintf("%d models", count), modelList
}

func makePortChecker(port int) func() (bool, string, string) {
	return func() (bool, string, string) {
		// Check if port is available (not in use means services not running)
		cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
		output, err := cmd.Output()

		if err != nil {
			// Port is available (not in use)
			return true, "Available", ""
		}

		// Port is in use - check what's using it
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) > 0 {
				return true, "In use by " + fields[0], ""
			}
		}

		return true, "In use", ""
	}
}

func checkRussellWithAutoStart() (bool, string, string) {
	const russellPort = 9100
	const startupTimeout = 10 * time.Second

	// First, check if Russell is already running
	if isPortListening(russellPort) {
		return true, "Running", ""
	}

	// Russell is not running, try to auto-start it
	binaryPath := findMdwBinary()
	if binaryPath == "" {
		return false, "Binary not found", ""
	}

	// Start Russell as background process
	cmd := exec.Command(binaryPath, "serve", "russell")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return false, "Failed to start: " + err.Error(), ""
	}

	// Wait for Russell to become available
	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		if isPortListening(russellPort) {
			return true, "Started", ""
		}
		time.Sleep(500 * time.Millisecond)
	}

	return false, "Timeout waiting for startup", ""
}

// isPortListening checks if a port is listening
func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findMdwBinary locates the mDW binary
func findMdwBinary() string {
	// Try common locations
	candidates := []string{
		"./bin/mdw",
		"bin/mdw",
		"mdw",
	}

	// Get working directory for absolute paths
	wd, _ := os.Getwd()

	for _, candidate := range candidates {
		var path string
		if filepath.IsAbs(candidate) {
			path = candidate
		} else {
			path = filepath.Join(wd, candidate)
		}

		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try finding in PATH
	if path, err := exec.LookPath("mdw"); err == nil {
		return path
	}

	return ""
}

func checkEmbeddingModel() (bool, string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	if err != nil {
		return false, "Cannot check models", ""
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "Ollama not running", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, "Cannot fetch models", ""
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "Parse error", ""
	}

	// Check for embedding models (nomic-embed-text, all-minilm, etc.)
	embeddingModels := []string{"nomic-embed-text", "all-minilm", "mxbai-embed", "snowflake-arctic-embed"}

	for _, m := range result.Models {
		for _, embed := range embeddingModels {
			if strings.Contains(strings.ToLower(m.Name), embed) {
				return true, "Available", m.Name
			}
		}
	}

	return false, "Not installed", "ollama pull nomic-embed-text"
}

// GetDependencyIcon returns the appropriate icon for a dependency status
func GetDependencyIcon(status DependencyStatus) string {
	switch status {
	case StatusOK:
		return IconOK
	case StatusFailed:
		return IconError
	case StatusWarning:
		return IconWarning
	case StatusChecking:
		return IconSpinner
	default:
		return IconBullet
	}
}
