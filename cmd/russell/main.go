package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msto63/mDW/internal/russell/server"
	"github.com/msto63/mDW/pkg/core/config"
	"github.com/msto63/mDW/pkg/core/logging"
)

func main() {
	logger := logging.New("russell")
	logger.Info("Starting Russell Orchestration Service")

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

	logger.Info("Russell server started", "port", cfg.Port)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutdown signal received, stopping server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	srv.Stop(ctx)

	logger.Info("Russell server stopped")
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

		cfg.Host = appCfg.Russell.Host
		cfg.Port = appCfg.Russell.Port
	}

	// Override from environment
	if host := os.Getenv("RUSSELL_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("RUSSELL_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}

	return cfg, nil
}
