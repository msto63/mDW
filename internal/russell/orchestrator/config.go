// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     orchestrator
// Description: Service orchestration configuration and loading
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package orchestrator

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/BurntSushi/toml"
)

// ServicesConfig is the root configuration structure from services.toml
type ServicesConfig struct {
	Orchestrator OrchestratorConfig     `toml:"orchestrator"`
	Dependencies map[string]Dependency  `toml:"dependencies"`
	Services     []ServiceConfig        `toml:"services"`
}

// OrchestratorConfig holds global orchestrator settings
type OrchestratorConfig struct {
	BinaryPath          string `toml:"binary_path"`
	LogDir              string `toml:"log_dir"`
	StartupTimeout      string `toml:"startup_timeout"`
	ShutdownTimeout     string `toml:"shutdown_timeout"`
	HealthCheckInterval string `toml:"health_check_interval"`
}

// Dependency represents an external dependency like Ollama
type Dependency struct {
	Name     string `toml:"name"`
	Type     string `toml:"type"`
	URL      string `toml:"url"`
	Required bool   `toml:"required"`
}

// ServiceConfig holds configuration for a single service
type ServiceConfig struct {
	Name                 string            `toml:"name"`
	ShortName            string            `toml:"short_name"`
	Description          string            `toml:"description"`
	Version              string            `toml:"version"`
	GRPCPort             int               `toml:"grpc_port"`
	HTTPPort             int               `toml:"http_port"`
	Command              []string          `toml:"command"`
	Dependencies         []string          `toml:"dependencies"`
	ExternalDependencies []string          `toml:"external_dependencies"`
	StartOrder           int               `toml:"start_order"`
	MaxRetries           int               `toml:"max_retries"`
	Enabled              bool              `toml:"enabled"`
	HealthCheck          HealthCheckConfig `toml:"health_check"`
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	Type     string `toml:"type"`
	Endpoint string `toml:"endpoint"`
	Interval string `toml:"interval"`
	Timeout  string `toml:"timeout"`
}

// GetStartupTimeout returns the startup timeout as duration
func (c *OrchestratorConfig) GetStartupTimeout() time.Duration {
	d, err := time.ParseDuration(c.StartupTimeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// GetShutdownTimeout returns the shutdown timeout as duration
func (c *OrchestratorConfig) GetShutdownTimeout() time.Duration {
	d, err := time.ParseDuration(c.ShutdownTimeout)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// GetHealthCheckInterval returns the health check interval as duration
func (c *OrchestratorConfig) GetHealthCheckInterval() time.Duration {
	d, err := time.ParseDuration(c.HealthCheckInterval)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// GetHealthCheckInterval returns the health check interval as duration
func (c *HealthCheckConfig) GetInterval() time.Duration {
	d, err := time.ParseDuration(c.Interval)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// GetHealthCheckTimeout returns the health check timeout as duration
func (c *HealthCheckConfig) GetTimeout() time.Duration {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 3 * time.Second
	}
	return d
}

// GetPrimaryPort returns the primary port for a service (gRPC or HTTP)
func (c *ServiceConfig) GetPrimaryPort() int {
	if c.GRPCPort > 0 {
		return c.GRPCPort
	}
	return c.HTTPPort
}

// LoadServicesConfig loads the services configuration from a TOML file
func LoadServicesConfig(path string) (*ServicesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServicesConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate and set defaults
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validate checks the configuration for errors and sets defaults
func (c *ServicesConfig) validate() error {
	// Set orchestrator defaults
	if c.Orchestrator.BinaryPath == "" {
		c.Orchestrator.BinaryPath = "./bin/mdw"
	}
	if c.Orchestrator.LogDir == "" {
		c.Orchestrator.LogDir = "./logs"
	}
	if c.Orchestrator.StartupTimeout == "" {
		c.Orchestrator.StartupTimeout = "30s"
	}
	if c.Orchestrator.ShutdownTimeout == "" {
		c.Orchestrator.ShutdownTimeout = "10s"
	}
	if c.Orchestrator.HealthCheckInterval == "" {
		c.Orchestrator.HealthCheckInterval = "10s"
	}

	// Validate services
	seenNames := make(map[string]bool)
	seenPorts := make(map[int]string)

	for i := range c.Services {
		svc := &c.Services[i]

		// Check required fields
		if svc.Name == "" {
			return fmt.Errorf("service at index %d has no name", i)
		}
		if svc.ShortName == "" {
			return fmt.Errorf("service %s has no short_name", svc.Name)
		}

		// Check for duplicates
		if seenNames[svc.ShortName] {
			return fmt.Errorf("duplicate service short_name: %s", svc.ShortName)
		}
		seenNames[svc.ShortName] = true

		// Check ports
		port := svc.GetPrimaryPort()
		if port > 0 {
			if existing, ok := seenPorts[port]; ok {
				return fmt.Errorf("port %d is used by both %s and %s", port, existing, svc.Name)
			}
			seenPorts[port] = svc.Name
		}

		// Set defaults
		if svc.MaxRetries == 0 {
			svc.MaxRetries = 3
		}
		if svc.StartOrder == 0 {
			svc.StartOrder = 100 // Default to high order
		}
		if svc.HealthCheck.Interval == "" {
			svc.HealthCheck.Interval = "10s"
		}
		if svc.HealthCheck.Timeout == "" {
			svc.HealthCheck.Timeout = "3s"
		}
	}

	// Validate dependencies exist
	for _, svc := range c.Services {
		for _, dep := range svc.Dependencies {
			if !seenNames[dep] {
				return fmt.Errorf("service %s depends on unknown service: %s", svc.Name, dep)
			}
		}
		for _, dep := range svc.ExternalDependencies {
			if _, ok := c.Dependencies[dep]; !ok {
				return fmt.Errorf("service %s depends on unknown external dependency: %s", svc.Name, dep)
			}
		}
	}

	return nil
}

// GetServicesSortedByStartOrder returns services sorted by start order
func (c *ServicesConfig) GetServicesSortedByStartOrder() []ServiceConfig {
	services := make([]ServiceConfig, 0, len(c.Services))
	for _, svc := range c.Services {
		if svc.Enabled {
			services = append(services, svc)
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].StartOrder < services[j].StartOrder
	})

	return services
}

// GetServiceByShortName finds a service by its short name
func (c *ServicesConfig) GetServiceByShortName(shortName string) *ServiceConfig {
	for i := range c.Services {
		if c.Services[i].ShortName == shortName {
			return &c.Services[i]
		}
	}
	return nil
}

// GetEnabledServices returns only enabled services
func (c *ServicesConfig) GetEnabledServices() []ServiceConfig {
	var enabled []ServiceConfig
	for _, svc := range c.Services {
		if svc.Enabled {
			enabled = append(enabled, svc)
		}
	}
	return enabled
}
