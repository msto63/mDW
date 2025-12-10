package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msto63/mDW/internal/turing/server"
	"github.com/msto63/mDW/pkg/core/config"
	"github.com/msto63/mDW/pkg/core/logging"
)

func main() {
	logger := logging.New("turing")
	logger.Info("Starting Turing LLM Service")

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

	logger.Info("Turing server started", "port", cfg.Port)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutdown signal received, stopping server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	srv.Stop(ctx)

	logger.Info("Turing server stopped")
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

		cfg.Host = appCfg.Turing.Host
		cfg.Port = appCfg.Turing.Port
		if appCfg.Turing.Providers.Ollama.BaseURL != "" {
			cfg.OllamaURL = appCfg.Turing.Providers.Ollama.BaseURL
		}
		if appCfg.Turing.Timeout.Duration > 0 {
			cfg.OllamaTimeout = appCfg.Turing.Timeout.Duration
		}
		cfg.DefaultModel = appCfg.Turing.DefaultModel
		// EmbeddingModel uses default if not in config
	}

	// Override from environment
	if host := os.Getenv("TURING_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("TURING_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}
	if url := os.Getenv("OLLAMA_URL"); url != "" {
		cfg.OllamaURL = url
	}
	if model := os.Getenv("TURING_DEFAULT_MODEL"); model != "" {
		cfg.DefaultModel = model
	}

	return cfg, nil
}
