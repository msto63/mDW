package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msto63/mDW/internal/leibniz/server"
	"github.com/msto63/mDW/pkg/core/config"
	"github.com/msto63/mDW/pkg/core/logging"
)

func main() {
	logger := logging.New("leibniz")
	logger.Info("Starting Leibniz Agentic AI Service")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Note: LLMFunc should be set by connecting to Turing service
	// For standalone operation, a placeholder would be needed

	// Start server
	if err := srv.StartAsync(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("Leibniz server started", "port", cfg.Port)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutdown signal received, stopping server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}

	logger.Info("Leibniz server stopped")
}

func loadConfig() (server.Config, error) {
	cfg := server.DefaultConfig()

	// Try to load from config file
	configPath := os.Getenv("MDW_CONFIG")
	if configPath == "" {
		configPath = "configs/config.toml"
	}

	if _, err := os.Stat(configPath); err == nil {
		appCfg, err := config.Load(configPath)
		if err != nil {
			return cfg, fmt.Errorf("failed to load config: %w", err)
		}

		cfg.Host = appCfg.Leibniz.Host
		cfg.Port = appCfg.Leibniz.Port
		cfg.MaxSteps = appCfg.Leibniz.MaxIterations
		if appCfg.Leibniz.DefaultTimeout.Duration > 0 {
			cfg.Timeout = appCfg.Leibniz.DefaultTimeout.Duration
		}

		// Load MCP server configurations
		for _, mcpCfg := range appCfg.Leibniz.MCP.Servers {
			cfg.MCPServers = append(cfg.MCPServers, server.MCPServerConfig{
				Name:    mcpCfg.Name,
				Command: mcpCfg.Command,
				Args:    mcpCfg.Args,
				Env:     mcpCfg.Env,
			})
		}
	}

	// Override from environment
	if host := os.Getenv("LEIBNIZ_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("LEIBNIZ_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}

	return cfg, nil
}
