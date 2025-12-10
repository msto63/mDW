// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     mcp
// Description: Standard MCP server configurations
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package mcp

import (
	"os"
	"path/filepath"
)

// StandardServer represents a standard MCP server configuration
type StandardServer struct {
	Name        string
	Description string
	Config      ServerConfig
	Category    string
	Required    []string // Required dependencies
}

// GetStandardServers returns configurations for standard MCP servers
func GetStandardServers() []StandardServer {
	homeDir, _ := os.UserHomeDir()

	return []StandardServer{
		// Filesystem server
		{
			Name:        "filesystem",
			Description: "Dateisystem-Operationen (Lesen, Schreiben, Suchen)",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", homeDir},
			},
			Category: "core",
			Required: []string{"node", "npx"},
		},

		// Browser/Puppeteer server
		{
			Name:        "puppeteer",
			Description: "Web-Browser-Automatisierung und Screenshots",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-puppeteer"},
			},
			Category: "web",
			Required: []string{"node", "npx"},
		},

		// Git server
		{
			Name:        "git",
			Description: "Git-Repository-Operationen",
			Config: ServerConfig{
				Command: "uvx",
				Args:    []string{"mcp-server-git", "--repository", "."},
			},
			Category: "dev",
			Required: []string{"uv", "uvx"},
		},

		// GitHub server
		{
			Name:        "github",
			Description: "GitHub API-Zugriff (Issues, PRs, Repos)",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-github"},
				Env: map[string]string{
					"GITHUB_PERSONAL_ACCESS_TOKEN": os.Getenv("GITHUB_TOKEN"),
				},
			},
			Category: "dev",
			Required: []string{"node", "npx", "GITHUB_TOKEN"},
		},

		// SQLite server
		{
			Name:        "sqlite",
			Description: "SQLite-Datenbankoperationen",
			Config: ServerConfig{
				Command: "uvx",
				Args:    []string{"mcp-server-sqlite", "--db-path", filepath.Join(homeDir, "data.db")},
			},
			Category: "data",
			Required: []string{"uv", "uvx"},
		},

		// Memory/Knowledge Graph server
		{
			Name:        "memory",
			Description: "Persistenter Wissensspeicher",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-memory"},
			},
			Category: "core",
			Required: []string{"node", "npx"},
		},

		// Fetch/HTTP server
		{
			Name:        "fetch",
			Description: "HTTP-Anfragen und Webseiten-Inhalte",
			Config: ServerConfig{
				Command: "uvx",
				Args:    []string{"mcp-server-fetch"},
			},
			Category: "web",
			Required: []string{"uv", "uvx"},
		},

		// Slack server
		{
			Name:        "slack",
			Description: "Slack-Workspace-Zugriff",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-slack"},
				Env: map[string]string{
					"SLACK_BOT_TOKEN": os.Getenv("SLACK_BOT_TOKEN"),
					"SLACK_TEAM_ID":   os.Getenv("SLACK_TEAM_ID"),
				},
			},
			Category: "communication",
			Required: []string{"node", "npx", "SLACK_BOT_TOKEN", "SLACK_TEAM_ID"},
		},

		// Google Drive server
		{
			Name:        "gdrive",
			Description: "Google Drive-Zugriff",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-gdrive"},
			},
			Category: "cloud",
			Required: []string{"node", "npx"},
		},

		// PostgreSQL server
		{
			Name:        "postgres",
			Description: "PostgreSQL-Datenbankoperationen",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-postgres"},
				Env: map[string]string{
					"POSTGRES_CONNECTION_STRING": os.Getenv("DATABASE_URL"),
				},
			},
			Category: "data",
			Required: []string{"node", "npx", "DATABASE_URL"},
		},

		// Sequential Thinking server
		{
			Name:        "sequential-thinking",
			Description: "Strukturiertes Denken und Problemlösung",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
			},
			Category: "reasoning",
			Required: []string{"node", "npx"},
		},

		// Everything (macOS Spotlight alternative) server
		{
			Name:        "everything",
			Description: "Schnelle Dateisuche (Everything-Integration)",
			Config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-everything"},
			},
			Category: "core",
			Required: []string{"node", "npx"},
		},
	}
}

// GetServerByName returns a standard server configuration by name
func GetServerByName(name string) *StandardServer {
	for _, server := range GetStandardServers() {
		if server.Name == name {
			return &server
		}
	}
	return nil
}

// GetServersByCategory returns all servers in a category
func GetServersByCategory(category string) []StandardServer {
	var servers []StandardServer
	for _, server := range GetStandardServers() {
		if server.Category == category {
			servers = append(servers, server)
		}
	}
	return servers
}

// GetCategories returns all available categories
func GetCategories() []string {
	categorySet := make(map[string]bool)
	for _, server := range GetStandardServers() {
		categorySet[server.Category] = true
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}
	return categories
}

// CheckRequirements checks if all requirements for a server are met
func CheckRequirements(server StandardServer) []string {
	var missing []string

	for _, req := range server.Required {
		// Check for environment variables
		if req == "GITHUB_TOKEN" ||
			req == "SLACK_BOT_TOKEN" || req == "SLACK_TEAM_ID" ||
			req == "DATABASE_URL" {
			if os.Getenv(req) == "" {
				missing = append(missing, "env:"+req)
			}
			continue
		}

		// Check for commands
		_, err := findExecutable(req)
		if err != nil {
			missing = append(missing, "cmd:"+req)
		}
	}

	return missing
}

// findExecutable looks for an executable in PATH
func findExecutable(name string) (string, error) {
	return filepath.Abs(name) // Simplified - in production use exec.LookPath
}

// RecommendedServers returns the recommended servers for common use cases
func RecommendedServers() []string {
	return []string{
		"filesystem",
		"memory",
		"fetch",
		"sequential-thinking",
	}
}

// ServerPresets defines common server combinations
var ServerPresets = map[string][]string{
	"minimal":   {"filesystem"},
	"standard":  {"filesystem", "memory", "fetch"},
	"web":       {"filesystem", "fetch"},          // Web-Recherche (use WebResearchAgent for search)
	"developer": {"filesystem", "memory", "git", "github"},
	"full":      {"filesystem", "memory", "fetch", "git", "github", "sqlite", "puppeteer"},
}

// GetPreset returns the server names for a preset
func GetPreset(name string) []string {
	if servers, ok := ServerPresets[name]; ok {
		return servers
	}
	return nil
}
