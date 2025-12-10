package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
	"github.com/msto63/mDW/internal/bayes/server"
	"github.com/msto63/mDW/internal/bayes/service"
	"github.com/msto63/mDW/pkg/core/config"
)

func main() {
	logger := logging.New("bayes")
	logger.Info("Starting Bayes Logging Service")

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

	// Start server
	if err := srv.StartAsync(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("Bayes server started", "port", cfg.Port)

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

	logger.Info("Bayes server stopped")
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

		cfg.Host = appCfg.Bayes.Host
		cfg.Port = appCfg.Bayes.Port
		cfg.Service = service.Config{
			LogDir:        appCfg.Bayes.StoragePath,
			MaxMemEntries: 10000, // Default value
			LogToFile:     true,
		}
	}

	// Override from environment
	if host := os.Getenv("BAYES_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("BAYES_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}
	if logDir := os.Getenv("BAYES_LOG_DIR"); logDir != "" {
		cfg.Service.LogDir = logDir
	}

	return cfg, nil
}
