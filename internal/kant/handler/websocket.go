package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/kant/client"
	"github.com/msto63/mDW/pkg/core/logging"
)

// WebSocket upgrader with permissive settings for local development
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

// WebSocketHandler handles WebSocket connections for real-time chat
type WebSocketHandler struct {
	clients *client.ServiceClients
	logger  *logging.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(clients *client.ServiceClients) *WebSocketHandler {
	return &WebSocketHandler{
		clients: clients,
		logger:  logging.New("kant-websocket"),
	}
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string          `json:"type"`    // "chat", "ping", "config"
	Payload json.RawMessage `json:"payload"` // Message-specific payload
}

// WSChatPayload represents the chat message payload
type WSChatPayload struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// WSResponse represents a WebSocket response
type WSResponse struct {
	Type    string      `json:"type"`    // "chunk", "done", "error", "pong"
	Payload interface{} `json:"payload"` // Response-specific payload
}

// WSChunkPayload represents a streaming chunk payload
type WSChunkPayload struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// WSErrorPayload represents an error payload
type WSErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ServeHTTP handles WebSocket upgrade and connections
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}
	h.handleConnection(conn)
}

// handleConnection handles a single WebSocket connection
func (h *WebSocketHandler) handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	h.logger.Info("WebSocket connection established", "remote", conn.RemoteAddr().String())

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set read deadline for ping/pong
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	// Read messages in a loop
	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket read error", "error", err)
			} else {
				h.logger.Info("WebSocket connection closed")
			}
			return
		}

		switch msg.Type {
		case "ping":
			h.sendResponse(conn, WSResponse{Type: "pong", Payload: nil})

		case "chat":
			var payload WSChatPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				h.sendError(conn, "invalid_payload", "Invalid chat payload")
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				h.handleChatMessage(ctx, conn, payload)
			}()

		default:
			h.sendError(conn, "unknown_type", "Unknown message type: "+msg.Type)
		}
	}
}

// handleChatMessage processes a chat message and streams the response
func (h *WebSocketHandler) handleChatMessage(ctx context.Context, conn *websocket.Conn, payload WSChatPayload) {
	if h.clients.Turing == nil {
		h.sendError(conn, "service_unavailable", "Turing service not available")
		return
	}

	if len(payload.Messages) == 0 {
		h.sendError(conn, "invalid_request", "Messages required")
		return
	}

	// Convert messages to protobuf format
	pbMessages := make([]*turingpb.Message, len(payload.Messages))
	for i, m := range payload.Messages {
		pbMessages[i] = &turingpb.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	grpcCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	grpcReq := &turingpb.ChatRequest{
		Messages:    pbMessages,
		Model:       payload.Model,
		MaxTokens:   int32(payload.MaxTokens),
		Temperature: float32(payload.Temperature),
	}

	stream, err := h.clients.Turing.StreamChat(grpcCtx, grpcReq)
	if err != nil {
		h.sendError(conn, "stream_error", "Failed to start chat stream: "+err.Error())
		return
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			h.sendResponse(conn, WSResponse{
				Type: "done",
				Payload: WSChunkPayload{
					Content: "",
					Done:    true,
				},
			})
			return
		}
		if err != nil {
			h.sendError(conn, "stream_error", "Stream error: "+err.Error())
			return
		}

		h.sendResponse(conn, WSResponse{
			Type: "chunk",
			Payload: WSChunkPayload{
				Content: chunk.Delta,
				Done:    chunk.Done,
			},
		})
	}
}

// sendResponse sends a response message via WebSocket
func (h *WebSocketHandler) sendResponse(conn *websocket.Conn, resp WSResponse) {
	if err := conn.WriteJSON(resp); err != nil {
		h.logger.Error("WebSocket send error", "error", err)
	}
}

// sendError sends an error response via WebSocket
func (h *WebSocketHandler) sendError(conn *websocket.Conn, code, message string) {
	h.sendResponse(conn, WSResponse{
		Type: "error",
		Payload: WSErrorPayload{
			Code:    code,
			Message: message,
		},
	})
}
