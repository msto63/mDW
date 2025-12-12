package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/internal/leibniz/agentloader"
	"github.com/msto63/mDW/internal/leibniz/evaluator"
	"github.com/msto63/mDW/internal/leibniz/mcp"
	"github.com/msto63/mDW/internal/leibniz/platon"
	"github.com/msto63/mDW/internal/leibniz/store"
	"github.com/msto63/mDW/internal/leibniz/tools"
	"github.com/msto63/mDW/internal/leibniz/websearch"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/msto63/mDW/pkg/core/logging"
)

// ExecuteRequest represents an agent execution request
type ExecuteRequest struct {
	Task       string
	Tools      []string // Specific tools to enable (empty = all)
	MaxSteps   int
	Timeout    time.Duration
	Context    map[string]string
}

// EvaluationOptions controls self-evaluation behavior
type EvaluationOptions struct {
	SkipEvaluation    bool // Skip evaluation even if agent has it enabled
	MaxIterations     int  // Override max iterations (0 = use agent default)
}

// EvaluationInfo contains results of self-evaluation
type EvaluationInfo struct {
	Performed      bool
	IterationsUsed int
	FinalScore     float32
	Passed         bool
	Feedback       string
	Criteria       []CriterionResultInfo
}

// CriterionResultInfo contains result of a single criterion
type CriterionResultInfo struct {
	Name     string
	Passed   bool
	Required bool
	Feedback string
}

// ExecuteResponse represents an agent execution response
type ExecuteResponse struct {
	ID         string
	Status     string
	Result     string
	Steps      []StepInfo
	ToolsUsed  []string
	Duration   time.Duration
	Error      string
	Evaluation *EvaluationInfo // Self-evaluation results (nil if not performed)
}

// StepInfo represents information about an execution step
type StepInfo struct {
	Index     int
	Thought   string
	Action    string
	ToolName  string
	ToolInput string
	ToolOutput string
	Timestamp time.Time
}

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string
	Description string
	Source      string // "builtin" or "mcp"
}

// AgentDefinition represents a stored agent definition
type AgentDefinition struct {
	ID           string
	Name         string
	Description  string
	SystemPrompt string
	Tools        []string
	Model        string
	MaxSteps     int
	Timeout      time.Duration
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ExecutionRecord represents a running or completed execution
type ExecutionRecord struct {
	ID            string
	AgentID       string
	Message       string
	Status        string
	Result        string
	Error         string
	Steps         []StepInfo
	ToolsUsed     []string
	StartedAt     time.Time
	CompletedAt   time.Time
	Duration      time.Duration
	Cancel        context.CancelFunc
}

// CustomTool represents a user-registered tool
type CustomTool struct {
	Name                 string
	Description          string
	ParameterSchema      string
	RequiresConfirmation bool
}

// Service is the Leibniz agentic AI service
type Service struct {
	agent            *agent.Agent
	mcpClients       map[string]*mcp.Client
	builtinTools     *tools.BuiltinTools
	webResearchAgent *websearch.WebResearchAgent
	platonClient     *platon.Client
	logger           *logging.Logger
	maxSteps         int
	llmFunc          agent.LLMFunc
	store            store.AgentStore

	// YAML-based agent loader with hot-reload
	agentLoader *agentloader.Loader

	// Self-evaluation support
	evaluator   *evaluator.Evaluator
	ollamaClient *ollama.Client

	// In-memory storage (fallback when store is nil)
	mu           sync.RWMutex
	agents       map[string]*AgentDefinition
	executions   map[string]*ExecutionRecord
	customTools  map[string]*CustomTool
	nextAgentID  int
	nextExecID   int
}

// Config holds service configuration
type Config struct {
	MaxSteps           int
	MCPServers         []MCPServerConfig
	MCPPreset          string   // "minimal", "standard", "developer", "full"
	StorePath          string
	EnablePersistence  bool
	EnableBuiltinTools bool
	AllowedPaths       []string // Paths allowed for file operations
	EnableNetwork      bool     // Enable network tools
	EnableWebSearch    bool     // Enable web search tools (deprecated, use EnableWebResearchAgent)
	EnableShell        bool     // Enable shell commands

	// Web Research Agent configuration
	EnableWebResearchAgent bool     // Enable the specialized web research agent
	SearXNGInstances       []string // Custom SearXNG instance URLs

	// Platon integration for pipeline processing
	EnablePlaton  bool          // Enable Platon integration
	PlatonHost    string        // Platon service host
	PlatonPort    int           // Platon service port
	PlatonTimeout time.Duration // Timeout for Platon calls

	// YAML-based agent configuration
	AgentsDir       string // Directory for YAML agent definitions
	EnableHotReload bool   // Enable hot-reload of agent definitions
}

// MCPServerConfig holds MCP server configuration
type MCPServerConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		MaxSteps:               10,
		MCPServers:             []MCPServerConfig{},
		MCPPreset:              "",
		StorePath:              "./data/agents.db",
		EnablePersistence:      true,
		EnableBuiltinTools:     true,
		AllowedPaths:           []string{},
		EnableNetwork:          true,
		EnableWebSearch:        false,      // Deprecated, use EnableWebResearchAgent
		EnableShell:            false,
		EnableWebResearchAgent: true,       // Web research agent enabled by default
		SearXNGInstances:       []string{}, // Use default public instances
		EnablePlaton:           true,       // Platon integration enabled by default
		PlatonHost:             "localhost",
		PlatonPort:             9130,
		PlatonTimeout:          30 * time.Second,
		AgentsDir:              "./configs/agents", // YAML agent definitions
		EnableHotReload:        true,               // Hot-reload enabled by default
	}
}

// NewService creates a new Leibniz service
func NewService(cfg Config) (*Service, error) {
	logger := logging.New("leibniz")

	// Create agent
	agentCfg := agent.DefaultConfig()
	agentCfg.MaxSteps = cfg.MaxSteps
	ag := agent.NewAgent(agentCfg)

	svc := &Service{
		agent:       ag,
		mcpClients:  make(map[string]*mcp.Client),
		logger:      logger,
		maxSteps:    cfg.MaxSteps,
		agents:      make(map[string]*AgentDefinition),
		executions:  make(map[string]*ExecutionRecord),
		customTools: make(map[string]*CustomTool),
	}

	// Initialize persistent store if enabled
	if cfg.EnablePersistence {
		agentStore, err := store.NewSQLiteAgentStore(store.SQLiteAgentConfig{
			Path: cfg.StorePath,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create agent store: %w", err)
		}
		svc.store = agentStore
		logger.Info("Agent persistence enabled", "path", cfg.StorePath)

		// Load agents from store
		if err := svc.loadAgentsFromStore(); err != nil {
			logger.Warn("Failed to load agents from store", "error", err)
		}
	}

	// Initialize YAML-based agent loader
	if cfg.AgentsDir != "" {
		svc.agentLoader = agentloader.NewLoader(cfg.AgentsDir)

		// Set callbacks for hot-reload
		svc.agentLoader.SetOnChange(func(agentID string, yamlAgent *agentloader.AgentYAML) {
			svc.onAgentReloaded(agentID, yamlAgent)
		})
		svc.agentLoader.SetOnDelete(func(agentID string) {
			svc.onAgentDeleted(agentID)
		})

		// Load all YAML agents
		if err := svc.agentLoader.LoadAll(); err != nil {
			logger.Warn("Failed to load YAML agents", "error", err)
		} else {
			// Register loaded agents
			for _, yamlAgent := range svc.agentLoader.GetAll() {
				svc.registerYAMLAgent(yamlAgent)
			}
		}

		// Start hot-reload watcher if enabled
		if cfg.EnableHotReload {
			if err := svc.agentLoader.StartWatching(context.Background()); err != nil {
				logger.Warn("Failed to start agent hot-reload watcher", "error", err)
			} else {
				logger.Info("Agent hot-reload enabled", "dir", cfg.AgentsDir)
			}
		}
	}

	// Create default agent if not exists
	if _, exists := svc.agents["default"]; !exists {
		defaultAgent := &AgentDefinition{
			ID:          "default",
			Name:        "Default Agent",
			Description: "Standard-Agent für allgemeine Aufgaben",
			SystemPrompt: `Du bist ein hilfreicher Assistent mit Zugriff auf verschiedene Tools.

VERFÜGBARE TOOLS:
- calculator: Mathematische Berechnungen
- current_time: Aktuelle Uhrzeit und Datum
- read_file, write_file, list_directory: Dateisystem-Operationen
- web_search: Internet-Suche (SearXNG/DuckDuckGo)
- fetch_webpage: Webseiten-Inhalte laden
- search_news: Aktuelle Nachrichten suchen

HINWEIS: Für umfangreiche Web-Recherchen steht der spezialisierte
Agent "web-researcher" zur Verfügung.

Arbeitsweise:
1. Analysiere die Anfrage des Benutzers
2. Wähle die passenden Tools aus
3. Führe die notwendigen Aktionen durch
4. Fasse die Ergebnisse verständlich zusammen`,
			MaxSteps:  cfg.MaxSteps,
			Timeout:   120 * time.Second,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		svc.agents["default"] = defaultAgent

		// Persist default agent
		if svc.store != nil {
			svc.store.CreateAgent(context.Background(), toStoreAgent(defaultAgent))
		}
	}

	// Register built-in tools (basic ones like calculator and time)
	svc.registerBuiltinTools()

	// Initialize built-in file/network tools if enabled
	if cfg.EnableBuiltinTools {
		toolsCfg := tools.DefaultConfig()
		if len(cfg.AllowedPaths) > 0 {
			toolsCfg.AllowedPaths = cfg.AllowedPaths
		}
		toolsCfg.EnableNetwork = cfg.EnableNetwork
		toolsCfg.EnableShell = cfg.EnableShell
		toolsCfg.EnableWebSearch = cfg.EnableWebSearch

		svc.builtinTools = tools.NewBuiltinTools(toolsCfg)
		svc.builtinTools.RegisterAll(ag, toolsCfg)
		logger.Info("Built-in tools enabled",
			"network", cfg.EnableNetwork,
			"websearch", cfg.EnableWebSearch,
			"shell", cfg.EnableShell,
		)
	}

	// Initialize Platon client if enabled (before WebResearchAgent so it can use it)
	if cfg.EnablePlaton {
		platonCfg := platon.Config{
			Host:    cfg.PlatonHost,
			Port:    cfg.PlatonPort,
			Timeout: cfg.PlatonTimeout,
		}
		platonClient, err := platon.NewClient(platonCfg)
		if err != nil {
			// Log warning but don't fail - Platon may not be running yet
			logger.Warn("Failed to connect to Platon service", "error", err,
				"host", cfg.PlatonHost, "port", cfg.PlatonPort)
		} else {
			svc.platonClient = platonClient
			logger.Info("Platon integration enabled",
				"host", cfg.PlatonHost, "port", cfg.PlatonPort)
		}
	}

	// Initialize Web Research Agent if enabled
	if cfg.EnableWebResearchAgent {
		webAgentCfg := websearch.DefaultAgentConfig()
		if len(cfg.SearXNGInstances) > 0 {
			webAgentCfg.SearXNGInstances = cfg.SearXNGInstances
		}

		svc.webResearchAgent = websearch.NewWebResearchAgent(webAgentCfg)

		// Connect Platon client to Web Research Agent for content filtering
		if svc.platonClient != nil {
			svc.webResearchAgent.SetPlatonClient(svc.platonClient, "web-research")
		}

		svc.webResearchAgent.RegisterTools(ag)

		// Register web-researcher agent definition
		webAgentDef := svc.webResearchAgent.GetAgentDefinition()
		svc.agents[webAgentDef.ID] = &AgentDefinition{
			ID:           webAgentDef.ID,
			Name:         webAgentDef.Name,
			Description:  webAgentDef.Description,
			SystemPrompt: webAgentDef.SystemPrompt,
			Tools:        webAgentDef.Tools,
			Model:        webAgentDef.Model,
			MaxSteps:     webAgentDef.MaxSteps,
			Timeout:      webAgentDef.Timeout,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Persist web-researcher agent
		if svc.store != nil {
			svc.store.CreateAgent(context.Background(), toStoreAgent(svc.agents[webAgentDef.ID]))
		}

		logger.Info("Web Research Agent enabled",
			"searxng_instances", len(webAgentCfg.SearXNGInstances),
			"platon_enabled", svc.platonClient != nil,
		)
	}

	// Auto-connect MCP servers from preset
	if cfg.MCPPreset != "" {
		serverNames := mcp.GetPreset(cfg.MCPPreset)
		for _, name := range serverNames {
			if stdServer := mcp.GetServerByName(name); stdServer != nil {
				missing := mcp.CheckRequirements(*stdServer)
				if len(missing) == 0 {
					go func(server *mcp.StandardServer) {
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()
						if err := svc.ConnectMCPServer(ctx, server.Name, server.Config); err != nil {
							logger.Warn("Failed to connect MCP server", "name", server.Name, "error", err)
						}
					}(stdServer)
				} else {
					logger.Debug("Skipping MCP server due to missing requirements",
						"name", name, "missing", missing)
				}
			}
		}
	}

	return svc, nil
}

// loadAgentsFromStore loads agents from persistent storage
func (s *Service) loadAgentsFromStore() error {
	if s.store == nil {
		return nil
	}

	agents, err := s.store.ListAgents(context.Background())
	if err != nil {
		return err
	}

	for _, a := range agents {
		s.agents[a.ID] = fromStoreAgent(a)
	}

	s.logger.Info("Agents loaded from store", "count", len(agents))
	return nil
}

// toStoreAgent converts service AgentDefinition to store AgentDefinition
func toStoreAgent(a *AgentDefinition) *store.AgentDefinition {
	return &store.AgentDefinition{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		SystemPrompt: a.SystemPrompt,
		Tools:        a.Tools,
		Model:        a.Model,
		MaxSteps:     a.MaxSteps,
		Timeout:      a.Timeout,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// fromStoreAgent converts store AgentDefinition to service AgentDefinition
func fromStoreAgent(a *store.AgentDefinition) *AgentDefinition {
	return &AgentDefinition{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		SystemPrompt: a.SystemPrompt,
		Tools:        a.Tools,
		Model:        a.Model,
		MaxSteps:     a.MaxSteps,
		Timeout:      a.Timeout,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// SetLLMFunc sets the LLM function for the agent
func (s *Service) SetLLMFunc(fn agent.LLMFunc) {
	s.llmFunc = fn
	s.agent.SetLLMFunc(fn)
}

// SetModelAwareLLMFunc sets the model-aware LLM function for the agent
func (s *Service) SetModelAwareLLMFunc(fn agent.ModelAwareLLMFunc) {
	s.agent.SetModelAwareLLMFunc(fn)
}

// SetOllamaClient sets the Ollama client for self-evaluation
func (s *Service) SetOllamaClient(client *ollama.Client) {
	s.ollamaClient = client
	if client != nil {
		s.evaluator = evaluator.New(client)
		s.logger.Info("Self-evaluation support enabled")
	}
}

// GetEvaluator returns the evaluator instance (may be nil)
func (s *Service) GetEvaluator() *evaluator.Evaluator {
	return s.evaluator
}

// registerBuiltinTools registers built-in tools
func (s *Service) registerBuiltinTools() {
	// Calculator tool
	s.agent.RegisterTool(&agent.Tool{
		Name:        "calculator",
		Description: "Führt mathematische Berechnungen durch",
		Parameters: map[string]agent.ParameterDef{
			"expression": {Type: "string", Description: "Mathematischer Ausdruck", Required: true},
		},
		Handler: s.calculatorHandler,
	})

	// Current time tool
	s.agent.RegisterTool(&agent.Tool{
		Name:        "current_time",
		Description: "Gibt die aktuelle Uhrzeit und das Datum zurück",
		Parameters:  map[string]agent.ParameterDef{},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return time.Now().Format("2006-01-02 15:04:05"), nil
		},
	})

	s.logger.Info("Built-in tools registered")
}

// calculatorHandler handles calculator tool calls
func (s *Service) calculatorHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	expr, ok := params["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("expression parameter required")
	}

	// Simple calculator (only handles basic operations)
	// In a real implementation, use a proper expression parser
	return fmt.Sprintf("Calculation result for '%s': [needs implementation]", expr), nil
}

// ConnectMCPServer connects to an MCP server
func (s *Service) ConnectMCPServer(ctx context.Context, name string, cfg mcp.ServerConfig) error {
	client, err := mcp.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	s.mcpClients[name] = client

	// Register MCP tools as agent tools
	for _, tool := range client.ListTools() {
		s.registerMCPTool(name, tool)
	}

	s.logger.Info("MCP server connected", "name", name, "tools", len(client.ListTools()))
	return nil
}

// registerMCPTool registers an MCP tool as an agent tool
func (s *Service) registerMCPTool(serverName string, tool mcp.Tool) {
	// Convert MCP tool to agent tool
	params := make(map[string]agent.ParameterDef)
	if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
		for name, def := range props {
			if defMap, ok := def.(map[string]interface{}); ok {
				params[name] = agent.ParameterDef{
					Type:        getString(defMap, "type"),
					Description: getString(defMap, "description"),
					Required:    false, // MCP doesn't always specify required
				}
			}
		}
	}

	agentTool := &agent.Tool{
		Name:        fmt.Sprintf("%s_%s", serverName, tool.Name),
		Description: tool.Description,
		Parameters:  params,
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			client, ok := s.mcpClients[serverName]
			if !ok {
				return nil, fmt.Errorf("MCP server not connected: %s", serverName)
			}

			result, err := client.CallTool(ctx, mcp.ToolCall{
				Name:      tool.Name,
				Arguments: args,
			})
			if err != nil {
				return nil, err
			}
			if result.IsError {
				return nil, fmt.Errorf("tool error: %s", result.Content)
			}
			return result.Content, nil
		},
	}

	s.agent.RegisterTool(agentTool)
}

// DisconnectMCPServer disconnects from an MCP server
func (s *Service) DisconnectMCPServer(name string) error {
	client, ok := s.mcpClients[name]
	if !ok {
		return nil
	}

	if err := client.Close(); err != nil {
		return err
	}

	delete(s.mcpClients, name)
	s.logger.Info("MCP server disconnected", "name", name)
	return nil
}

// Execute runs an agent task
func (s *Service) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	if req.Task == "" {
		return nil, fmt.Errorf("task is required")
	}

	s.logger.Info("Executing agent task", "task", req.Task)

	// Set timeout if specified
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// Execute agent
	start := time.Now()
	execution, err := s.agent.Execute(ctx, req.Task)

	response := &ExecuteResponse{
		ID:       execution.ID,
		Status:   string(execution.Status),
		Result:   execution.Result,
		Error:    execution.Error,
		Duration: time.Since(start),
		ToolsUsed: execution.ToolsUsed,
	}

	// Convert steps
	for _, step := range execution.Steps {
		stepInfo := StepInfo{
			Index:     step.Index,
			Thought:   step.Thought,
			Action:    step.Action,
			Timestamp: step.Timestamp,
		}
		if step.ToolCall != nil {
			stepInfo.ToolName = step.ToolCall.Name
			stepInfo.ToolInput = fmt.Sprintf("%v", step.ToolCall.Params)
		}
		if step.ToolResult != nil {
			if step.ToolResult.Error != "" {
				stepInfo.ToolOutput = "Error: " + step.ToolResult.Error
			} else {
				stepInfo.ToolOutput = fmt.Sprintf("%v", step.ToolResult.Result)
			}
		}
		response.Steps = append(response.Steps, stepInfo)
	}

	return response, err
}

// ListTools returns all available tools
func (s *Service) ListTools() []ToolInfo {
	var tools []ToolInfo

	// Built-in tools
	for _, t := range s.agent.ListTools() {
		source := "builtin"
		for serverName := range s.mcpClients {
			if len(t.Name) > len(serverName)+1 && t.Name[:len(serverName)+1] == serverName+"_" {
				source = "mcp:" + serverName
				break
			}
		}
		tools = append(tools, ToolInfo{
			Name:        t.Name,
			Description: t.Description,
			Source:      source,
		})
	}

	return tools
}

// HealthCheck checks if the service is healthy
func (s *Service) HealthCheck(ctx context.Context) error {
	return nil
}

// Agent returns the underlying agent for tool registration
func (s *Service) Agent() *agent.Agent {
	return s.agent
}

// Close closes the service and all MCP connections
func (s *Service) Close() error {
	// Stop agent hot-reload watcher
	if s.agentLoader != nil {
		s.agentLoader.Stop()
	}

	for name := range s.mcpClients {
		s.DisconnectMCPServer(name)
	}
	if s.platonClient != nil {
		s.platonClient.Close()
	}
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// PlatonClient returns the Platon client (may be nil if not enabled)
func (s *Service) PlatonClient() *platon.Client {
	return s.platonClient
}

// ProcessWithPlaton processes input through Platon pipeline
func (s *Service) ProcessWithPlaton(ctx context.Context, pipelineID, prompt string, metadata map[string]string) (*platon.ProcessResponse, error) {
	if s.platonClient == nil {
		return nil, fmt.Errorf("platon client not initialized")
	}

	req := &platon.ProcessRequest{
		RequestID:  fmt.Sprintf("leibniz-%d", time.Now().UnixNano()),
		PipelineID: pipelineID,
		Prompt:     prompt,
		Metadata:   metadata,
	}

	return s.platonClient.ProcessPre(ctx, req)
}

// ProcessResponseWithPlaton processes response through Platon pipeline
func (s *Service) ProcessResponseWithPlaton(ctx context.Context, pipelineID, prompt, response string, metadata map[string]string) (*platon.ProcessResponse, error) {
	if s.platonClient == nil {
		return nil, fmt.Errorf("platon client not initialized")
	}

	req := &platon.ProcessRequest{
		RequestID:  fmt.Sprintf("leibniz-%d", time.Now().UnixNano()),
		PipelineID: pipelineID,
		Prompt:     prompt,
		Response:   response,
		Metadata:   metadata,
	}

	return s.platonClient.ProcessPost(ctx, req)
}

// MCP Server Management Methods

// ListMCPServers returns connected MCP servers
func (s *Service) ListMCPServers() []string {
	servers := make([]string, 0, len(s.mcpClients))
	for name := range s.mcpClients {
		servers = append(servers, name)
	}
	return servers
}

// GetAvailableMCPServers returns all available standard MCP servers
func (s *Service) GetAvailableMCPServers() []mcp.StandardServer {
	return mcp.GetStandardServers()
}

// GetMCPServerCategories returns all MCP server categories
func (s *Service) GetMCPServerCategories() []string {
	return mcp.GetCategories()
}

// GetMCPServersByCategory returns servers in a category
func (s *Service) GetMCPServersByCategory(category string) []mcp.StandardServer {
	return mcp.GetServersByCategory(category)
}

// ConnectStandardMCPServer connects a standard MCP server by name
func (s *Service) ConnectStandardMCPServer(ctx context.Context, name string) error {
	server := mcp.GetServerByName(name)
	if server == nil {
		return fmt.Errorf("unknown MCP server: %s", name)
	}

	// Check requirements
	missing := mcp.CheckRequirements(*server)
	if len(missing) > 0 {
		return fmt.Errorf("missing requirements: %v", missing)
	}

	return s.ConnectMCPServer(ctx, name, server.Config)
}

// GetMCPServerStatus returns the status of connected MCP servers
func (s *Service) GetMCPServerStatus() map[string]bool {
	status := make(map[string]bool)
	for name, client := range s.mcpClients {
		status[name] = client.IsConnected()
	}
	return status
}

// Agent Management Methods

// CreateAgent creates a new agent definition
func (s *Service) CreateAgent(def *AgentDefinition) (*AgentDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextAgentID++
	def.ID = fmt.Sprintf("agent_%d", s.nextAgentID)
	def.CreatedAt = time.Now()
	def.UpdatedAt = time.Now()

	if def.MaxSteps <= 0 {
		def.MaxSteps = s.maxSteps
	}
	if def.Timeout <= 0 {
		def.Timeout = 60 * time.Second
	}

	// Persist to store
	if s.store != nil {
		if err := s.store.CreateAgent(context.Background(), toStoreAgent(def)); err != nil {
			s.logger.Warn("Failed to persist agent", "error", err)
		}
	}

	s.agents[def.ID] = def
	s.logger.Info("Agent created", "id", def.ID, "name", def.Name)

	return def, nil
}

// UpdateAgent updates an existing agent definition
func (s *Service) UpdateAgent(id string, updates *AgentDefinition) (*AgentDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	// Apply updates
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.SystemPrompt != "" {
		existing.SystemPrompt = updates.SystemPrompt
	}
	if len(updates.Tools) > 0 {
		existing.Tools = updates.Tools
	}
	if updates.Model != "" {
		existing.Model = updates.Model
	}
	if updates.MaxSteps > 0 {
		existing.MaxSteps = updates.MaxSteps
	}
	if updates.Timeout > 0 {
		existing.Timeout = updates.Timeout
	}

	existing.UpdatedAt = time.Now()

	// Persist to store
	if s.store != nil {
		if err := s.store.UpdateAgent(context.Background(), toStoreAgent(existing)); err != nil {
			s.logger.Warn("Failed to persist agent update", "error", err)
		}
	}

	s.logger.Info("Agent updated", "id", id)

	return existing, nil
}

// DeleteAgent deletes an agent definition
func (s *Service) DeleteAgent(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "default" {
		return fmt.Errorf("cannot delete default agent")
	}

	if _, ok := s.agents[id]; !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	// Persist deletion to store
	if s.store != nil {
		if err := s.store.DeleteAgent(context.Background(), id); err != nil {
			s.logger.Warn("Failed to persist agent deletion", "error", err)
		}
	}

	delete(s.agents, id)
	s.logger.Info("Agent deleted", "id", id)

	return nil
}

// GetAgent retrieves an agent definition
func (s *Service) GetAgent(id string) (*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return agent, nil
}

// ListAgents lists all agent definitions
func (s *Service) ListAgents() []*AgentDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*AgentDefinition, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}

	return agents
}

// Execution Management Methods

// ExecuteWithAgent runs a task with a specific agent
func (s *Service) ExecuteWithAgent(ctx context.Context, agentID string, message string) (*ExecuteResponse, error) {
	s.mu.RLock()
	agentDef, ok := s.agents[agentID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// Create execution record
	s.mu.Lock()
	s.nextExecID++
	execID := fmt.Sprintf("exec_%d", s.nextExecID)

	execCtx, cancel := context.WithTimeout(ctx, agentDef.Timeout)

	record := &ExecutionRecord{
		ID:        execID,
		AgentID:   agentID,
		Message:   message,
		Status:    "running",
		StartedAt: time.Now(),
		Cancel:    cancel,
	}
	s.executions[execID] = record

	// Persist initial execution record
	if s.store != nil {
		s.store.CreateExecution(context.Background(), toStoreExecution(record))
	}
	s.mu.Unlock()

	// Store original settings to restore after execution
	originalModel := s.agent.GetModel()
	originalPrompt := s.agent.GetSystemPrompt()

	// Set agent-specific model if defined
	if agentDef.Model != "" {
		s.agent.SetModel(agentDef.Model)
		s.logger.Info("Using agent-specific model", "agent", agentID, "model", agentDef.Model)
	}

	// Set agent-specific system prompt if defined
	if agentDef.SystemPrompt != "" {
		s.agent.SetSystemPrompt(agentDef.SystemPrompt)
		s.logger.Info("Using agent-specific system prompt", "agent", agentID, "prompt_length", len(agentDef.SystemPrompt))
	}

	// Defer restoration of original settings
	defer func() {
		s.agent.SetModel(originalModel)
		s.agent.SetSystemPrompt(originalPrompt)
	}()

	// Execute the task
	req := &ExecuteRequest{
		Task:     message,
		MaxSteps: agentDef.MaxSteps,
		Timeout:  agentDef.Timeout,
	}

	resp, err := s.Execute(execCtx, req)

	// Update execution record
	s.mu.Lock()
	if record, ok := s.executions[execID]; ok {
		record.Status = resp.Status
		record.Result = resp.Result
		record.Error = resp.Error
		record.Steps = resp.Steps
		record.ToolsUsed = resp.ToolsUsed
		record.CompletedAt = time.Now()
		record.Duration = resp.Duration
		if err != nil {
			record.Status = "error"
			record.Error = err.Error()
		}

		// Persist updated execution record
		if s.store != nil {
			s.store.UpdateExecution(context.Background(), toStoreExecution(record))
		}
	}
	s.mu.Unlock()

	if resp != nil {
		resp.ID = execID
	}

	return resp, err
}

// ExecuteWithAgentAndEvaluation runs a task with self-evaluation and iterative improvement
// opts can be nil to use agent defaults
func (s *Service) ExecuteWithAgentAndEvaluation(ctx context.Context, agentID string, message string, opts *EvaluationOptions) (*ExecuteResponse, error) {
	// Check if evaluation should be skipped
	if opts != nil && opts.SkipEvaluation {
		s.logger.Info("Evaluation skipped by request", "agent", agentID)
		return s.ExecuteWithAgent(ctx, agentID, message)
	}

	// Get YAML agent definition for evaluation config
	var yamlAgent *agentloader.AgentYAML
	if s.agentLoader != nil {
		yamlAgent, _ = s.agentLoader.Get(agentID)
	}

	// If no YAML agent or no evaluation config, fall back to standard execution
	if yamlAgent == nil || yamlAgent.Evaluation == nil || !yamlAgent.Evaluation.Enabled {
		return s.ExecuteWithAgent(ctx, agentID, message)
	}

	// Check if evaluator is available
	if s.evaluator == nil {
		s.logger.Warn("Evaluation requested but evaluator not initialized, falling back to standard execution",
			"agent", agentID)
		return s.ExecuteWithAgent(ctx, agentID, message)
	}

	// Apply max iterations override if specified
	if opts != nil && opts.MaxIterations > 0 {
		// Create a copy of the evaluation config with overridden max iterations
		evalCopy := *yamlAgent.Evaluation
		evalCopy.MaxIterations = opts.MaxIterations
		yamlAgent.Evaluation = &evalCopy
		s.logger.Info("Max iterations overridden", "agent", agentID, "max_iterations", opts.MaxIterations)
	}

	s.mu.RLock()
	agentDef, ok := s.agents[agentID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// Create execution record
	s.mu.Lock()
	s.nextExecID++
	execID := fmt.Sprintf("exec_%d", s.nextExecID)

	execCtx, cancel := context.WithTimeout(ctx, agentDef.Timeout*time.Duration(yamlAgent.Evaluation.MaxIterations+1))

	record := &ExecutionRecord{
		ID:        execID,
		AgentID:   agentID,
		Message:   message,
		Status:    "running",
		StartedAt: time.Now(),
		Cancel:    cancel,
	}
	s.executions[execID] = record

	if s.store != nil {
		s.store.CreateExecution(context.Background(), toStoreExecution(record))
	}
	s.mu.Unlock()

	// Store original settings to restore after execution
	originalModel := s.agent.GetModel()
	originalPrompt := s.agent.GetSystemPrompt()

	// Set agent-specific model if defined
	if agentDef.Model != "" {
		s.agent.SetModel(agentDef.Model)
	}

	// Set agent-specific system prompt if defined
	if agentDef.SystemPrompt != "" {
		s.agent.SetSystemPrompt(agentDef.SystemPrompt)
	}

	// Defer restoration of original settings
	defer func() {
		s.agent.SetModel(originalModel)
		s.agent.SetSystemPrompt(originalPrompt)
	}()

	s.logger.Info("Starting execution with self-evaluation",
		"agent", agentID,
		"max_iterations", yamlAgent.Evaluation.MaxIterations,
		"criteria_count", len(yamlAgent.Evaluation.Criteria),
	)

	// Execute with evaluation loop
	start := time.Now()
	execution, err := s.agent.ExecuteWithEvaluation(execCtx, message, yamlAgent, s.evaluator)

	response := &ExecuteResponse{
		ID:        execID,
		Status:    string(execution.Status),
		Result:    execution.Result,
		Error:     execution.Error,
		Duration:  time.Since(start),
		ToolsUsed: execution.ToolsUsed,
	}

	// Convert steps
	for _, step := range execution.Steps {
		stepInfo := StepInfo{
			Index:     step.Index,
			Thought:   step.Thought,
			Action:    step.Action,
			Timestamp: step.Timestamp,
		}
		if step.ToolCall != nil {
			stepInfo.ToolName = step.ToolCall.Name
			stepInfo.ToolInput = fmt.Sprintf("%v", step.ToolCall.Params)
		}
		if step.ToolResult != nil {
			if step.ToolResult.Error != "" {
				stepInfo.ToolOutput = "Error: " + step.ToolResult.Error
			} else {
				stepInfo.ToolOutput = fmt.Sprintf("%v", step.ToolResult.Result)
			}
		}
		response.Steps = append(response.Steps, stepInfo)
	}

	// Add evaluation metadata to response
	if execution.Iterations > 0 && len(execution.EvaluationResults) > 0 {
		lastEval := execution.EvaluationResults[len(execution.EvaluationResults)-1]
		evalInfo := &EvaluationInfo{
			Performed:      true,
			IterationsUsed: execution.Iterations,
			FinalScore:     execution.FinalQualityScore,
			Passed:         lastEval.Passed,
			Feedback:       lastEval.Feedback,
		}
		// Convert criteria results
		for _, cr := range lastEval.CriteriaResults {
			evalInfo.Criteria = append(evalInfo.Criteria, CriterionResultInfo{
				Name:     cr.Name,
				Passed:   cr.Passed,
				Required: cr.Required,
				Feedback: cr.Feedback,
			})
		}
		response.Evaluation = evalInfo

		s.logger.Info("Execution completed with evaluation",
			"agent", agentID,
			"iterations", execution.Iterations,
			"final_score", execution.FinalQualityScore,
			"passed", lastEval.Passed,
		)
	}

	// Update execution record
	s.mu.Lock()
	if record, ok := s.executions[execID]; ok {
		record.Status = response.Status
		record.Result = response.Result
		record.Error = response.Error
		record.Steps = response.Steps
		record.ToolsUsed = response.ToolsUsed
		record.CompletedAt = time.Now()
		record.Duration = response.Duration
		if err != nil {
			record.Status = "error"
			record.Error = err.Error()
		}

		if s.store != nil {
			s.store.UpdateExecution(context.Background(), toStoreExecution(record))
		}
	}
	s.mu.Unlock()

	return response, err
}

// toStoreExecution converts service ExecutionRecord to store ExecutionRecord
func toStoreExecution(r *ExecutionRecord) *store.ExecutionRecord {
	storeSteps := make([]store.StepInfo, len(r.Steps))
	for i, s := range r.Steps {
		storeSteps[i] = store.StepInfo{
			Index:      s.Index,
			Thought:    s.Thought,
			Action:     s.Action,
			ToolName:   s.ToolName,
			ToolInput:  s.ToolInput,
			ToolOutput: s.ToolOutput,
			Timestamp:  s.Timestamp,
		}
	}

	return &store.ExecutionRecord{
		ID:          r.ID,
		AgentID:     r.AgentID,
		Message:     r.Message,
		Status:      r.Status,
		Result:      r.Result,
		Error:       r.Error,
		Steps:       storeSteps,
		ToolsUsed:   r.ToolsUsed,
		StartedAt:   r.StartedAt,
		CompletedAt: r.CompletedAt,
		Duration:    r.Duration.Milliseconds(),
	}
}

// CancelExecution cancels a running execution
func (s *Service) CancelExecution(execID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.executions[execID]
	if !ok {
		return fmt.Errorf("execution not found: %s", execID)
	}

	if record.Status != "running" {
		return fmt.Errorf("execution not running: %s", execID)
	}

	if record.Cancel != nil {
		record.Cancel()
	}

	record.Status = "cancelled"
	record.CompletedAt = time.Now()
	s.logger.Info("Execution cancelled", "id", execID)

	return nil
}

// GetExecution retrieves an execution record
func (s *Service) GetExecution(execID string) (*ExecutionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.executions[execID]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", execID)
	}

	return record, nil
}

// Tool Management Methods

// RegisterCustomTool registers a custom tool
func (s *Service) RegisterCustomTool(tool *CustomTool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	s.customTools[tool.Name] = tool
	s.logger.Info("Custom tool registered", "name", tool.Name)

	return nil
}

// UnregisterCustomTool removes a custom tool
func (s *Service) UnregisterCustomTool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.customTools[name]; !ok {
		return fmt.Errorf("tool not found: %s", name)
	}

	delete(s.customTools, name)
	s.logger.Info("Custom tool unregistered", "name", name)

	return nil
}

// GetCustomTools returns all custom tools
func (s *Service) GetCustomTools() []*CustomTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]*CustomTool, 0, len(s.customTools))
	for _, tool := range s.customTools {
		tools = append(tools, tool)
	}

	return tools
}

// Helper function to safely get string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// YAML Agent Management Methods

// registerYAMLAgent converts a YAML agent definition and registers it
func (s *Service) registerYAMLAgent(yamlAgent *agentloader.AgentYAML) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agentDef := &AgentDefinition{
		ID:           yamlAgent.ID,
		Name:         yamlAgent.Name,
		Description:  yamlAgent.Description,
		SystemPrompt: yamlAgent.SystemPrompt,
		Tools:        yamlAgent.GetToolNames(),
		Model:        yamlAgent.Model,
		MaxSteps:     yamlAgent.MaxSteps,
		Timeout:      yamlAgent.Timeout,
		CreatedAt:    yamlAgent.LoadedAt,
		UpdatedAt:    yamlAgent.LoadedAt,
	}

	s.agents[yamlAgent.ID] = agentDef
	s.logger.Info("YAML agent registered",
		"id", yamlAgent.ID,
		"name", yamlAgent.Name,
		"model", yamlAgent.Model,
		"tools", len(yamlAgent.Tools),
	)
}

// onAgentReloaded is called when a YAML agent is created or updated
func (s *Service) onAgentReloaded(agentID string, yamlAgent *agentloader.AgentYAML) {
	s.registerYAMLAgent(yamlAgent)
	s.logger.Info("Agent hot-reloaded", "id", agentID, "name", yamlAgent.Name)
}

// onAgentDeleted is called when a YAML agent file is deleted
func (s *Service) onAgentDeleted(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[agentID]; exists {
		delete(s.agents, agentID)
		s.logger.Info("YAML agent removed", "id", agentID)
	}
}

// GetAgentLoader returns the YAML agent loader (for external access)
func (s *Service) GetAgentLoader() *agentloader.Loader {
	return s.agentLoader
}

// SetEmbeddingFunc sets the embedding function for agent matching
// This enables RAG-style agent selection based on task similarity
func (s *Service) SetEmbeddingFunc(fn agentloader.EmbeddingFunc) {
	if s.agentLoader == nil {
		s.logger.Warn("Cannot set embedding function: agent loader not initialized")
		return
	}

	s.agentLoader.SetEmbeddingFunc(fn)
	s.logger.Info("Embedding function configured for agent matching")

	// Update embeddings for all loaded agents
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := s.agentLoader.UpdateAllEmbeddings(ctx); err != nil {
			s.logger.Warn("Failed to update initial agent embeddings", "error", err)
		}
	}()
}

// FindBestAgentForTask finds the best matching agent for a task description
// using vector similarity (RAG-style matching)
func (s *Service) FindBestAgentForTask(ctx context.Context, taskDescription string) (*AgentMatch, error) {
	if s.agentLoader == nil {
		return nil, fmt.Errorf("agent loader not initialized")
	}

	match, err := s.agentLoader.FindBestAgentForTask(ctx, taskDescription)
	if err != nil {
		return nil, err
	}

	return &AgentMatch{
		AgentID:    match.AgentID,
		AgentName:  match.AgentName,
		Similarity: match.Similarity,
	}, nil
}

// FindTopAgentsForTask finds the top N matching agents for a task
func (s *Service) FindTopAgentsForTask(ctx context.Context, taskDescription string, topN int) ([]*AgentMatch, error) {
	if s.agentLoader == nil {
		return nil, fmt.Errorf("agent loader not initialized")
	}

	matches, err := s.agentLoader.FindTopAgentsForTask(ctx, taskDescription, topN)
	if err != nil {
		return nil, err
	}

	result := make([]*AgentMatch, len(matches))
	for i, m := range matches {
		result[i] = &AgentMatch{
			AgentID:    m.AgentID,
			AgentName:  m.AgentName,
			Similarity: m.Similarity,
		}
	}

	return result, nil
}

// GetAgentInfoList returns a list of all agents with their info
func (s *Service) GetAgentInfoList() []*AgentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]*AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		infos = append(infos, &AgentInfo{
			ID:          agent.ID,
			Name:        agent.Name,
			Description: agent.Description,
		})
	}
	return infos
}

// AgentMatch represents an agent matching result
type AgentMatch struct {
	AgentID    string
	AgentName  string
	Similarity float64
}

// AgentInfo represents basic agent information
type AgentInfo struct {
	ID          string
	Name        string
	Description string
}

// SaveAgentAsYAML saves an agent definition to a YAML file
func (s *Service) SaveAgentAsYAML(agentDef *AgentDefinition) error {
	if s.agentLoader == nil {
		return fmt.Errorf("YAML agent loader not initialized")
	}

	// Convert to YAML format
	yamlAgent := &agentloader.AgentYAML{
		ID:           agentDef.ID,
		Name:         agentDef.Name,
		Description:  agentDef.Description,
		Model:        agentDef.Model,
		MaxSteps:     agentDef.MaxSteps,
		Timeout:      agentDef.Timeout,
		SystemPrompt: agentDef.SystemPrompt,
	}

	// Convert tool names to tool configs
	for _, toolName := range agentDef.Tools {
		yamlAgent.Tools = append(yamlAgent.Tools, agentloader.ToolConfig{
			Name:    toolName,
			Enabled: true,
		})
	}

	return s.agentLoader.SaveAgent(yamlAgent)
}

// ReloadAgents reloads all YAML agents from disk
func (s *Service) ReloadAgents() error {
	if s.agentLoader == nil {
		return fmt.Errorf("YAML agent loader not initialized")
	}

	// Clear YAML-based agents (keep DB-based ones)
	s.mu.Lock()
	for id := range s.agents {
		if _, isYAML := s.agentLoader.Get(id); isYAML {
			delete(s.agents, id)
		}
	}
	s.mu.Unlock()

	// Reload from YAML
	if err := s.agentLoader.LoadAll(); err != nil {
		return err
	}

	// Re-register all YAML agents
	for _, yamlAgent := range s.agentLoader.GetAll() {
		s.registerYAMLAgent(yamlAgent)
	}

	s.logger.Info("Agents reloaded", "count", len(s.agentLoader.GetAll()))
	return nil
}
