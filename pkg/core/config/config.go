package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds the complete application configuration
type Config struct {
	General  GeneralConfig            `toml:"general"`
	Kant     KantConfig               `toml:"kant"`
	Russell  RussellConfig            `toml:"russell"`
	Turing   TuringConfig             `toml:"turing"`
	Hypatia  HypatiaConfig            `toml:"hypatia"`
	Leibniz  LeibnizConfig            `toml:"leibniz"`
	Babbage  BabbageConfig            `toml:"babbage"`
	Bayes    BayesConfig              `toml:"bayes"`
}

// GeneralConfig holds general application settings
type GeneralConfig struct {
	Name        string `toml:"name"`
	Environment string `toml:"environment"`
	DataDir     string `toml:"data_dir"`
	LogLevel    string `toml:"log_level"`
}

// KantConfig holds API Gateway configuration
type KantConfig struct {
	Port           int           `toml:"port"`
	Host           string        `toml:"host"`
	ReadTimeout    Duration      `toml:"read_timeout"`
	WriteTimeout   Duration      `toml:"write_timeout"`
	MaxRequestSize string        `toml:"max_request_size"`
	CORS           CORSConfig    `toml:"cors"`
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	Enabled        bool     `toml:"enabled"`
	AllowedOrigins []string `toml:"allowed_origins"`
	AllowedMethods []string `toml:"allowed_methods"`
}

// RussellConfig holds Service Discovery configuration
type RussellConfig struct {
	Port                int      `toml:"port"`
	Host                string   `toml:"host"`
	HealthCheckInterval Duration `toml:"health_check_interval"`
	HeartbeatTimeout    Duration `toml:"heartbeat_timeout"`
	CleanupInterval     Duration `toml:"cleanup_interval"`
}

// TuringConfig holds LLM Management configuration
type TuringConfig struct {
	Port               int             `toml:"port"`
	Host               string          `toml:"host"`
	DefaultProvider    string          `toml:"default_provider"`
	DefaultModel       string          `toml:"default_model"`
	DefaultTemperature float32         `toml:"default_temperature"`
	DefaultMaxTokens   int             `toml:"default_max_tokens"`
	Timeout            Duration        `toml:"timeout"`
	Providers          ProvidersConfig `toml:"providers"`
}

// ProvidersConfig holds LLM provider configurations
type ProvidersConfig struct {
	Ollama    ProviderConfig `toml:"ollama"`
	OpenAI    ProviderConfig `toml:"openai"`
	Anthropic ProviderConfig `toml:"anthropic"`
}

// ProviderConfig holds a single provider's configuration
type ProviderConfig struct {
	Enabled bool   `toml:"enabled"`
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
}

// HypatiaConfig holds RAG Service configuration
type HypatiaConfig struct {
	Port              int                `toml:"port"`
	Host              string             `toml:"host"`
	DefaultCollection string             `toml:"default_collection"`
	DefaultTopK       int                `toml:"default_top_k"`
	MinRelevanceScore float32            `toml:"min_relevance_score"`
	Chunking          ChunkingConfig     `toml:"chunking"`
	Embedding         EmbeddingConfig    `toml:"embedding"`
	VectorStore       VectorStoreConfig  `toml:"vectorstore"`
}

// ChunkingConfig holds document chunking settings
type ChunkingConfig struct {
	DefaultSize    int    `toml:"default_size"`
	DefaultOverlap int    `toml:"default_overlap"`
	Strategy       string `toml:"strategy"`
}

// EmbeddingConfig holds embedding settings
type EmbeddingConfig struct {
	Model        string   `toml:"model"`
	Dimensions   int      `toml:"dimensions"`
	CacheEnabled bool     `toml:"cache_enabled"`
	CacheTTL     Duration `toml:"cache_ttl"`
}

// VectorStoreConfig holds vector store settings
type VectorStoreConfig struct {
	Type string `toml:"type"`
	Path string `toml:"path"`
	URL  string `toml:"url"`
}

// LeibnizConfig holds Agentic AI configuration
type LeibnizConfig struct {
	Port            int          `toml:"port"`
	Host            string       `toml:"host"`
	MaxIterations   int          `toml:"max_iterations"`
	DefaultTimeout  Duration     `toml:"default_timeout"`
	EnableStreaming bool         `toml:"enable_streaming"`
	Tools           ToolsConfig  `toml:"tools"`
	MCP             MCPConfig    `toml:"mcp"`
}

// ToolsConfig holds built-in tools configuration
type ToolsConfig struct {
	WebSearch       bool `toml:"web_search"`
	Calculator      bool `toml:"calculator"`
	CodeInterpreter bool `toml:"code_interpreter"`
	FileReader      bool `toml:"file_reader"`
	ShellCommand    bool `toml:"shell_command"`
}

// MCPConfig holds MCP (Model Context Protocol) configuration
type MCPConfig struct {
	Enabled bool        `toml:"enabled"`
	Servers []MCPServer `toml:"servers"`
}

// MCPServer holds a single MCP server configuration
type MCPServer struct {
	Name    string            `toml:"name"`
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
}

// BabbageConfig holds NLP Service configuration
type BabbageConfig struct {
	Port            int    `toml:"port"`
	Host            string `toml:"host"`
	DefaultLanguage string `toml:"default_language"`
}

// BayesConfig holds Logging Service configuration
type BayesConfig struct {
	Port          int            `toml:"port"`
	Host          string         `toml:"host"`
	StoragePath   string         `toml:"storage_path"`
	RetentionDays int            `toml:"retention_days"`
	MaxLogSize    string         `toml:"max_log_size"`
	Rotation      RotationConfig `toml:"rotation"`
}

// RotationConfig holds log rotation settings
type RotationConfig struct {
	Enabled  bool `toml:"enabled"`
	MaxFiles int  `toml:"max_files"`
	Compress bool `toml:"compress"`
}

// Duration wraps time.Duration for TOML parsing
type Duration struct {
	time.Duration
}

// UnmarshalText parses a duration string
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// MarshalText formats the duration as a string
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// Load loads configuration from a TOML file
func Load(path string) (*Config, error) {
	// Expand environment variables in path
	path = os.ExpandEnv(path)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	// Expand environment variables in sensitive fields
	cfg.expandEnvVars()

	return &cfg, nil
}

// LoadFromEnv loads configuration from the MDW_CONFIG environment variable
func LoadFromEnv() (*Config, error) {
	path := os.Getenv("MDW_CONFIG")
	if path == "" {
		// Try default locations
		defaultPaths := []string{
			"./configs/config.toml",
			"./config.toml",
			filepath.Join(os.Getenv("HOME"), ".config/meindenkwerk/config.toml"),
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path == "" {
		return nil, fmt.Errorf("no config file found, set MDW_CONFIG or create configs/config.toml")
	}

	return Load(path)
}

// applyDefaults sets default values for missing configuration
func (c *Config) applyDefaults() {
	// General
	if c.General.Name == "" {
		c.General.Name = "meinDENKWERK"
	}
	if c.General.Environment == "" {
		c.General.Environment = "development"
	}
	if c.General.DataDir == "" {
		c.General.DataDir = "./data"
	}
	if c.General.LogLevel == "" {
		c.General.LogLevel = "info"
	}

	// Kant
	if c.Kant.Port == 0 {
		c.Kant.Port = 8080
	}
	if c.Kant.Host == "" {
		c.Kant.Host = "0.0.0.0"
	}
	if c.Kant.ReadTimeout.Duration == 0 {
		c.Kant.ReadTimeout.Duration = 30 * time.Second
	}
	if c.Kant.WriteTimeout.Duration == 0 {
		c.Kant.WriteTimeout.Duration = 120 * time.Second
	}

	// Russell
	if c.Russell.Port == 0 {
		c.Russell.Port = 9100
	}
	if c.Russell.Host == "" {
		c.Russell.Host = "0.0.0.0"
	}
	if c.Russell.HealthCheckInterval.Duration == 0 {
		c.Russell.HealthCheckInterval.Duration = 10 * time.Second
	}
	if c.Russell.HeartbeatTimeout.Duration == 0 {
		c.Russell.HeartbeatTimeout.Duration = 30 * time.Second
	}

	// Turing
	if c.Turing.Port == 0 {
		c.Turing.Port = 9200
	}
	if c.Turing.Host == "" {
		c.Turing.Host = "0.0.0.0"
	}
	if c.Turing.DefaultProvider == "" {
		c.Turing.DefaultProvider = "ollama"
	}
	if c.Turing.DefaultModel == "" {
		c.Turing.DefaultModel = "mistral:7b"
	}
	if c.Turing.DefaultTemperature == 0 {
		c.Turing.DefaultTemperature = 0.7
	}
	if c.Turing.DefaultMaxTokens == 0 {
		c.Turing.DefaultMaxTokens = 2048
	}
	if c.Turing.Timeout.Duration == 0 {
		c.Turing.Timeout.Duration = 120 * time.Second
	}

	// Hypatia
	if c.Hypatia.Port == 0 {
		c.Hypatia.Port = 9220
	}
	if c.Hypatia.Host == "" {
		c.Hypatia.Host = "0.0.0.0"
	}
	if c.Hypatia.DefaultCollection == "" {
		c.Hypatia.DefaultCollection = "default"
	}
	if c.Hypatia.DefaultTopK == 0 {
		c.Hypatia.DefaultTopK = 5
	}
	if c.Hypatia.MinRelevanceScore == 0 {
		c.Hypatia.MinRelevanceScore = 0.7
	}
	if c.Hypatia.Chunking.DefaultSize == 0 {
		c.Hypatia.Chunking.DefaultSize = 512
	}
	if c.Hypatia.Chunking.DefaultOverlap == 0 {
		c.Hypatia.Chunking.DefaultOverlap = 128
	}
	if c.Hypatia.Chunking.Strategy == "" {
		c.Hypatia.Chunking.Strategy = "sentence"
	}
	if c.Hypatia.VectorStore.Type == "" {
		c.Hypatia.VectorStore.Type = "sqlite"
	}

	// Leibniz
	if c.Leibniz.Port == 0 {
		c.Leibniz.Port = 9140
	}
	if c.Leibniz.Host == "" {
		c.Leibniz.Host = "0.0.0.0"
	}
	if c.Leibniz.MaxIterations == 0 {
		c.Leibniz.MaxIterations = 10
	}
	if c.Leibniz.DefaultTimeout.Duration == 0 {
		c.Leibniz.DefaultTimeout.Duration = 60 * time.Second
	}

	// Babbage
	if c.Babbage.Port == 0 {
		c.Babbage.Port = 9150
	}
	if c.Babbage.Host == "" {
		c.Babbage.Host = "0.0.0.0"
	}
	if c.Babbage.DefaultLanguage == "" {
		c.Babbage.DefaultLanguage = "de"
	}

	// Bayes
	if c.Bayes.Port == 0 {
		c.Bayes.Port = 9120
	}
	if c.Bayes.Host == "" {
		c.Bayes.Host = "0.0.0.0"
	}
	if c.Bayes.StoragePath == "" {
		c.Bayes.StoragePath = "./data/logs"
	}
	if c.Bayes.RetentionDays == 0 {
		c.Bayes.RetentionDays = 30
	}
}

// expandEnvVars expands environment variables in configuration values
func (c *Config) expandEnvVars() {
	c.Turing.Providers.OpenAI.APIKey = os.ExpandEnv(c.Turing.Providers.OpenAI.APIKey)
	c.Turing.Providers.Anthropic.APIKey = os.ExpandEnv(c.Turing.Providers.Anthropic.APIKey)
	c.General.DataDir = os.ExpandEnv(c.General.DataDir)
	c.Bayes.StoragePath = os.ExpandEnv(c.Bayes.StoragePath)
	c.Hypatia.VectorStore.Path = os.ExpandEnv(c.Hypatia.VectorStore.Path)
}

// GetServiceAddress returns the address string for a service
func (c *Config) GetServiceAddress(service string) string {
	switch service {
	case "kant":
		return fmt.Sprintf("%s:%d", c.Kant.Host, c.Kant.Port)
	case "russell":
		return fmt.Sprintf("%s:%d", c.Russell.Host, c.Russell.Port)
	case "turing":
		return fmt.Sprintf("%s:%d", c.Turing.Host, c.Turing.Port)
	case "hypatia":
		return fmt.Sprintf("%s:%d", c.Hypatia.Host, c.Hypatia.Port)
	case "leibniz":
		return fmt.Sprintf("%s:%d", c.Leibniz.Host, c.Leibniz.Port)
	case "babbage":
		return fmt.Sprintf("%s:%d", c.Babbage.Host, c.Babbage.Port)
	case "bayes":
		return fmt.Sprintf("%s:%d", c.Bayes.Host, c.Bayes.Port)
	default:
		return ""
	}
}
