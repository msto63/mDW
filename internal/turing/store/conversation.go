package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Conversation represents a chat conversation
type Conversation struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Model     string            `json:"model"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Message represents a chat message within a conversation
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // system, user, assistant
	Content        string    `json:"content"`
	TokenCount     int       `json:"token_count,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ConversationStore defines the interface for conversation persistence
type ConversationStore interface {
	// Conversation operations
	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	DeleteConversation(ctx context.Context, id string) error
	ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error)

	// Message operations
	AddMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context, conversationID string, limit int) ([]*Message, error)
	DeleteMessages(ctx context.Context, conversationID string) error

	// Utility
	Close() error
	Statistics(ctx context.Context) (map[string]interface{}, error)
}

// SQLiteConversationStore implements ConversationStore using SQLite
type SQLiteConversationStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// SQLiteConversationConfig holds configuration for SQLite store
type SQLiteConversationConfig struct {
	Path string
}

// DefaultConversationConfig returns default configuration
func DefaultConversationConfig() SQLiteConversationConfig {
	return SQLiteConversationConfig{
		Path: "./data/conversations.db",
	}
}

// NewSQLiteConversationStore creates a new SQLite-based conversation store
func NewSQLiteConversationStore(cfg SQLiteConversationConfig) (*SQLiteConversationStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with WAL mode
	db, err := sql.Open("sqlite3", cfg.Path+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteConversationStore{db: db}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *SQLiteConversationStore) initSchema() error {
	schema := `
	-- Conversations table
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL DEFAULT '',
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Messages table
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		token_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	);

	-- Indices
	CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
	CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at DESC);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateConversation creates a new conversation
func (s *SQLiteConversationStore) CreateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv.ID == "" {
		return fmt.Errorf("conversation ID is required")
	}

	now := time.Now()
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = now
	}
	conv.UpdatedAt = now

	var metadataJSON []byte
	if conv.Metadata != nil {
		metadataJSON, _ = json.Marshal(conv.Metadata)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO conversations (id, title, model, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, conv.ID, conv.Title, conv.Model, metadataJSON, conv.CreatedAt, conv.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

// GetConversation retrieves a conversation by ID
func (s *SQLiteConversationStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, model, metadata, created_at, updated_at
		FROM conversations WHERE id = ?
	`, id)

	var conv Conversation
	var metadataJSON sql.NullString

	err := row.Scan(&conv.ID, &conv.Title, &conv.Model, &metadataJSON, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &conv.Metadata)
	}

	return &conv, nil
}

// UpdateConversation updates a conversation
func (s *SQLiteConversationStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv.UpdatedAt = time.Now()

	var metadataJSON []byte
	if conv.Metadata != nil {
		metadataJSON, _ = json.Marshal(conv.Metadata)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE conversations
		SET title = ?, model = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`, conv.Title, conv.Model, metadataJSON, conv.UpdatedAt, conv.ID)

	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found: %s", conv.ID)
	}

	return nil
}

// DeleteConversation deletes a conversation and its messages
func (s *SQLiteConversationStore) DeleteConversation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

// ListConversations returns conversations ordered by last update
func (s *SQLiteConversationStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, model, metadata, created_at, updated_at
		FROM conversations
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		var conv Conversation
		var metadataJSON sql.NullString

		if err := rows.Scan(&conv.ID, &conv.Title, &conv.Model, &metadataJSON, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &conv.Metadata)
		}

		conversations = append(conversations, &conv)
	}

	return conversations, nil
}

// AddMessage adds a message to a conversation
func (s *SQLiteConversationStore) AddMessage(ctx context.Context, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		return fmt.Errorf("message ID is required")
	}
	if msg.ConversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert message
	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (id, conversation_id, role, content, token_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.TokenCount, msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// Update conversation timestamp
	_, err = tx.ExecContext(ctx, `
		UPDATE conversations SET updated_at = ? WHERE id = ?
	`, time.Now(), msg.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	return tx.Commit()
}

// GetMessages retrieves messages for a conversation
func (s *SQLiteConversationStore) GetMessages(ctx context.Context, conversationID string, limit int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var args []interface{}

	if limit > 0 {
		// Get last N messages
		query = `
			SELECT id, conversation_id, role, content, token_count, created_at
			FROM messages
			WHERE conversation_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{conversationID, limit}
	} else {
		// Get all messages
		query = `
			SELECT id, conversation_id, role, content, token_count, created_at
			FROM messages
			WHERE conversation_id = ?
			ORDER BY created_at ASC
		`
		args = []interface{}{conversationID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.TokenCount, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}

	// Reverse if we used DESC order
	if limit > 0 && len(messages) > 1 {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	return messages, nil
}

// DeleteMessages deletes all messages for a conversation
func (s *SQLiteConversationStore) DeleteMessages(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE conversation_id = ?`, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteConversationStore) Close() error {
	return s.db.Close()
}

// Statistics returns store statistics
func (s *SQLiteConversationStore) Statistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total conversations
	var totalConvs int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations`).Scan(&totalConvs)
	stats["total_conversations"] = totalConvs

	// Total messages
	var totalMsgs int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages`).Scan(&totalMsgs)
	stats["total_messages"] = totalMsgs

	// Average messages per conversation
	if totalConvs > 0 {
		stats["avg_messages_per_conversation"] = float64(totalMsgs) / float64(totalConvs)
	}

	// Total tokens
	var totalTokens sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT SUM(token_count) FROM messages`).Scan(&totalTokens)
	if totalTokens.Valid {
		stats["total_tokens"] = totalTokens.Int64
	}

	return stats, nil
}

// MemoryConversationStore is an in-memory implementation for testing
type MemoryConversationStore struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
	messages      map[string][]*Message // conversationID -> messages
}

// NewMemoryConversationStore creates a new in-memory conversation store
func NewMemoryConversationStore() *MemoryConversationStore {
	return &MemoryConversationStore{
		conversations: make(map[string]*Conversation),
		messages:      make(map[string][]*Message),
	}
}

// CreateConversation creates a new conversation
func (s *MemoryConversationStore) CreateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv.ID == "" {
		return fmt.Errorf("conversation ID is required")
	}

	now := time.Now()
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = now
	}
	conv.UpdatedAt = now

	s.conversations[conv.ID] = conv
	s.messages[conv.ID] = []*Message{}
	return nil
}

// GetConversation retrieves a conversation by ID
func (s *MemoryConversationStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, nil
	}
	return conv, nil
}

// UpdateConversation updates a conversation
func (s *MemoryConversationStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.conversations[conv.ID]; !ok {
		return fmt.Errorf("conversation not found: %s", conv.ID)
	}

	conv.UpdatedAt = time.Now()
	s.conversations[conv.ID] = conv
	return nil
}

// DeleteConversation deletes a conversation and its messages
func (s *MemoryConversationStore) DeleteConversation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

// ListConversations returns conversations ordered by last update
func (s *MemoryConversationStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []*Conversation
	for _, conv := range s.conversations {
		all = append(all, conv)
	}

	// Sort by updated_at descending (simple bubble sort for small lists)
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].UpdatedAt.After(all[i].UpdatedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	// Apply pagination
	if offset >= len(all) {
		return []*Conversation{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	return all[offset:end], nil
}

// AddMessage adds a message to a conversation
func (s *MemoryConversationStore) AddMessage(ctx context.Context, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		return fmt.Errorf("message ID is required")
	}

	if _, ok := s.conversations[msg.ConversationID]; !ok {
		return fmt.Errorf("conversation not found: %s", msg.ConversationID)
	}

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	s.messages[msg.ConversationID] = append(s.messages[msg.ConversationID], msg)
	s.conversations[msg.ConversationID].UpdatedAt = time.Now()
	return nil
}

// GetMessages retrieves messages for a conversation
func (s *MemoryConversationStore) GetMessages(ctx context.Context, conversationID string, limit int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs, ok := s.messages[conversationID]
	if !ok {
		return []*Message{}, nil
	}

	if limit <= 0 || limit >= len(msgs) {
		return msgs, nil
	}

	// Return last N messages
	return msgs[len(msgs)-limit:], nil
}

// DeleteMessages deletes all messages for a conversation
func (s *MemoryConversationStore) DeleteMessages(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[conversationID] = []*Message{}
	return nil
}

// Close is a no-op for memory store
func (s *MemoryConversationStore) Close() error {
	return nil
}

// Statistics returns store statistics
func (s *MemoryConversationStore) Statistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalMsgs int
	for _, msgs := range s.messages {
		totalMsgs += len(msgs)
	}

	return map[string]interface{}{
		"total_conversations": len(s.conversations),
		"total_messages":      totalMsgs,
	}, nil
}
