package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// Message represents an MCP JSON-RPC message
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Client is an MCP client that communicates with MCP servers
type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	reader    *bufio.Reader
	logger    *logging.Logger
	mu        sync.Mutex
	nextID    int
	pending   map[interface{}]chan *Message
	tools     []Tool
	resources []Resource
	connected bool
}

// ServerConfig holds MCP server configuration
type ServerConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// NewClient creates a new MCP client
func NewClient(cfg ServerConfig) (*Client, error) {
	logger := logging.New("mcp-client")

	cmd := exec.Command(cfg.Command, cfg.Args...)

	// Set environment
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	client := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		reader:  bufio.NewReader(stdout),
		logger:  logger,
		pending: make(map[interface{}]chan *Message),
	}

	return client, nil
}

// Connect starts the MCP server and initializes the connection
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Start message reader
	go c.readMessages()

	// Start stderr reader
	go c.readStderr()

	// Send initialize request
	initResp, err := c.sendRequest(ctx, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "meinDENKWERK",
			"version": "1.0.0",
		},
	})
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	c.logger.Info("MCP server initialized", "response", initResp)

	// Send initialized notification
	if err := c.sendNotification("notifications/initialized", nil); err != nil {
		c.logger.Warn("Failed to send initialized notification", "error", err)
	}

	// List available tools
	if err := c.refreshTools(ctx); err != nil {
		c.logger.Warn("Failed to list tools", "error", err)
	}

	// List available resources
	if err := c.refreshResources(ctx); err != nil {
		c.logger.Warn("Failed to list resources", "error", err)
	}

	c.connected = true
	return nil
}

// refreshTools fetches available tools from the server
func (c *Client) refreshTools(ctx context.Context) error {
	resp, err := c.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return err
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if toolsList, ok := result["tools"].([]interface{}); ok {
			c.tools = make([]Tool, 0, len(toolsList))
			for _, t := range toolsList {
				if toolMap, ok := t.(map[string]interface{}); ok {
					tool := Tool{
						Name:        getString(toolMap, "name"),
						Description: getString(toolMap, "description"),
					}
					if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
						tool.InputSchema = schema
					}
					c.tools = append(c.tools, tool)
				}
			}
		}
	}

	c.logger.Info("Tools refreshed", "count", len(c.tools))
	return nil
}

// refreshResources fetches available resources from the server
func (c *Client) refreshResources(ctx context.Context) error {
	resp, err := c.sendRequest(ctx, "resources/list", nil)
	if err != nil {
		return err
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if resourcesList, ok := result["resources"].([]interface{}); ok {
			c.resources = make([]Resource, 0, len(resourcesList))
			for _, r := range resourcesList {
				if resMap, ok := r.(map[string]interface{}); ok {
					res := Resource{
						URI:         getString(resMap, "uri"),
						Name:        getString(resMap, "name"),
						Description: getString(resMap, "description"),
						MimeType:    getString(resMap, "mimeType"),
					}
					c.resources = append(c.resources, res)
				}
			}
		}
	}

	c.logger.Info("Resources refreshed", "count", len(c.resources))
	return nil
}

// ListTools returns available tools
func (c *Client) ListTools() []Tool {
	return c.tools
}

// ListResources returns available resources
func (c *Client) ListResources() []Resource {
	return c.resources
}

// CallTool calls an MCP tool
func (c *Client) CallTool(ctx context.Context, call ToolCall) (*ToolResult, error) {
	resp, err := c.sendRequest(ctx, "tools/call", map[string]interface{}{
		"name":      call.Name,
		"arguments": call.Arguments,
	})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return &ToolResult{
			Content: resp.Error.Message,
			IsError: true,
		}, nil
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		content := ""
		if contentList, ok := result["content"].([]interface{}); ok {
			for _, item := range contentList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						content += text
					}
				}
			}
		}
		return &ToolResult{
			Content: content,
			IsError: false,
		}, nil
	}

	return &ToolResult{
		Content: fmt.Sprintf("%v", resp.Result),
		IsError: false,
	}, nil
}

// ReadResource reads a resource
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	resp, err := c.sendRequest(ctx, "resources/read", map[string]interface{}{
		"uri": uri,
	})
	if err != nil {
		return "", err
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if contents, ok := result["contents"].([]interface{}); ok {
			for _, item := range contents {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						return text, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no content returned")
}

// sendRequest sends a JSON-RPC request and waits for response
func (c *Client) sendRequest(ctx context.Context, method string, params interface{}) (*Message, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	responseCh := make(chan *Message, 1)
	c.pending[id] = responseCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	msg := Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.sendMessage(&msg); err != nil {
		return nil, err
	}

	select {
	case resp := <-responseCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// sendNotification sends a JSON-RPC notification (no response expected)
func (c *Client) sendNotification(method string, params interface{}) error {
	msg := Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.sendMessage(&msg)
}

// sendMessage sends a JSON-RPC message
func (c *Client) sendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	_, err = c.stdin.Write(append(data, '\n'))
	c.mu.Unlock()

	return err
}

// readMessages continuously reads messages from stdout
func (c *Client) readMessages() {
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				c.logger.Error("Error reading message", "error", err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			c.logger.Warn("Failed to parse message", "error", err, "line", string(line))
			continue
		}

		// Handle response
		if msg.ID != nil {
			c.mu.Lock()
			if ch, ok := c.pending[msg.ID]; ok {
				ch <- &msg
			}
			c.mu.Unlock()
		}
	}
}

// readStderr reads stderr output
func (c *Client) readStderr() {
	reader := bufio.NewReader(c.stderr)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		c.logger.Debug("MCP stderr", "line", line)
	}
}

// Close closes the client and stops the server
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	c.stdin.Close()
	return c.cmd.Process.Kill()
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Helper function to safely get string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
