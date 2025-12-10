package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msto63/mDW/internal/kant/server"
	"github.com/msto63/mDW/pkg/core/config"
	"github.com/msto63/mDW/pkg/core/logging"
)

func main() {
	logger := logging.New("kant")
	logger.Info("Starting Kant API Gateway")

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

	logger.Info("Kant API Gateway started", "address", srv.Address())

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

	logger.Info("Kant API Gateway stopped")
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

		cfg.Host = appCfg.Kant.Host
		cfg.HTTPPort = appCfg.Kant.Port

		// Parse timeouts
		if appCfg.Kant.ReadTimeout.Duration > 0 {
			cfg.ReadTimeout = appCfg.Kant.ReadTimeout.Duration
		}
		if appCfg.Kant.WriteTimeout.Duration > 0 {
			cfg.WriteTimeout = appCfg.Kant.WriteTimeout.Duration
		}
	}

	// Override from environment
	if host := os.Getenv("KANT_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("KANT_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.HTTPPort)
	}

	return cfg, nil
}
