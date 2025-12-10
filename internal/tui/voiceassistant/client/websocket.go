// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     client
// Description: WebSocket client for streaming chat
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient is a WebSocket client for streaming chat
type WSClient struct {
	mu      sync.RWMutex
	url     string
	conn    *websocket.Conn
	model   string
	running bool
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSChatPayload is the payload for chat messages
type WSChatPayload struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// WSChunkPayload is the payload for response chunks
// Kant sends "content", but we also support "delta" for compatibility
type WSChunkPayload struct {
	Content string `json:"content,omitempty"` // Kant WebSocket format
	Delta   string `json:"delta,omitempty"`   // Alternative format
	Done    bool   `json:"done,omitempty"`
	Error   string `json:"error,omitempty"`
}

// GetText returns the text content from either Content or Delta field
func (p *WSChunkPayload) GetText() string {
	if p.Content != "" {
		return p.Content
	}
	return p.Delta
}

// NewWSClient creates a new WebSocket client
func NewWSClient(url, model string) *WSClient {
	return &WSClient{
		url:   url,
		model: model,
	}
}

// Connect establishes a WebSocket connection
func (c *WSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.running = true

	return nil
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = false

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

// ChatStream sends a chat message and streams the response
func (c *WSClient) ChatStream(ctx context.Context, userMessage string, onChunk func(chunk string, done bool)) error {
	return c.ChatStreamWithHistory(ctx, []Message{{Role: "user", Content: userMessage}}, onChunk)
}

// ChatStreamWithHistory sends a chat with full history and streams the response
func (c *WSClient) ChatStreamWithHistory(ctx context.Context, messages []Message, onChunk func(chunk string, done bool)) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		if err := c.Connect(ctx); err != nil {
			return err
		}
		c.mu.RLock()
		conn = c.conn
		c.mu.RUnlock()
	}

	// Send chat message with full history
	msg := WSMessage{
		Type: "chat",
		Payload: WSChatPayload{
			Model:    c.model,
			Messages: messages,
		},
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Read response chunks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var resp WSMessage
		if err := conn.ReadJSON(&resp); err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		switch resp.Type {
		case "chunk":
			payloadBytes, _ := json.Marshal(resp.Payload)
			var chunk WSChunkPayload
			if err := json.Unmarshal(payloadBytes, &chunk); err != nil {
				continue
			}
			if chunk.Error != "" {
				return fmt.Errorf("server error: %s", chunk.Error)
			}
			if onChunk != nil {
				// Use GetText() to handle both "content" (Kant) and "delta" formats
				onChunk(chunk.GetText(), chunk.Done)
			}
			if chunk.Done {
				return nil
			}

		case "done":
			if onChunk != nil {
				onChunk("", true)
			}
			return nil

		case "error":
			payloadBytes, _ := json.Marshal(resp.Payload)
			var errPayload struct {
				Error string `json:"error"`
			}
			json.Unmarshal(payloadBytes, &errPayload)
			return fmt.Errorf("server error: %s", errPayload.Error)

		case "pong":
			// Ignore pong messages
			continue
		}
	}
}

// SendPing sends a ping message
func (c *WSClient) SendPing() error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	msg := WSMessage{Type: "ping"}
	return conn.WriteJSON(msg)
}

// IsConnected returns whether the client is connected
func (c *WSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && c.running
}

// SetModel sets the model to use
func (c *WSClient) SetModel(model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.model = model
}
