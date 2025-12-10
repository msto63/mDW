// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     chatclient
// Description: Message types for async operations in ChatClient
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package chatclient

import (
	"time"
)

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role      string        // user, assistant, system
	Content   string        // message content
	Model     string        // model used (for assistant messages)
	Timestamp time.Time     // when the message was sent/received
	Duration  time.Duration // how long the response took (for assistant messages)
}

// ModelInfo represents an available model
type ModelInfo struct {
	Name        string
	Size        string
	Description string
	Available   bool
}

// Message types for tea.Cmd async operations

// chatResponseMsg is sent when chat response is received
type chatResponseMsg struct {
	content  string
	model    string
	duration time.Duration
	err      error
}

// streamChunkMsg is sent for each streaming chunk
type streamChunkMsg struct {
	delta    string
	done     bool
	err      error
	duration time.Duration // Set when done
}

// streamSession holds the active streaming state
type streamSession struct {
	respCh    <-chan interface{} // *ollama.ChatResponse or gRPC chunk
	errCh     <-chan error
	startTime time.Time
}

// modelsLoadedMsg is sent when models are loaded
type modelsLoadedMsg struct {
	models []ModelInfo
	err    error
}

// serviceStatusMsg is sent when service status is checked
type serviceStatusMsg struct {
	turingOnline bool
	err          error
}

// tickMsg is used for periodic updates
type tickMsg time.Time

// clearInputMsg signals to clear the input
type clearInputMsg struct{}

// focusInputMsg signals to focus the input
type focusInputMsg struct{}

// scrollToBottomMsg signals to scroll viewport to bottom
type scrollToBottomMsg struct{}
