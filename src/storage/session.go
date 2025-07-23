package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/google/uuid"
)

// GetSessionByID retrieves a session by its ID
func GetSessionByID(ctx context.Context, db sqlscan.Querier, sessionID string) (*Session, error) {
	query := `SELECT id, current_conversation_id, json(conversation_ids) as conversation_ids, created_at, updated_at FROM sessions WHERE id = ?`
	var s Session
	err := sqlscan.Get(ctx, db, &s, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &s, nil
}

// GetLatestSession retrieves the most recently updated session
func GetLatestSession(ctx context.Context, db sqlscan.Querier) (*Session, error) {
	query := `SELECT id, current_conversation_id, json(conversation_ids) as conversation_ids, created_at, updated_at FROM sessions ORDER BY updated_at DESC LIMIT 1`
	var s Session
	err := sqlscan.Get(ctx, db, &s, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No sessions exist
		}
		return nil, err
	}
	return &s, nil
}

// CreateSession creates a new session in the database
func CreateSession(ctx context.Context, db Execer, session *Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.ConversationIDs == nil {
		session.ConversationIDs = JSONStringArray{}
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = time.Now()
	}

	query := `INSERT INTO sessions (id, conversation_ids, created_at, updated_at) VALUES (?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query, session.ID, session.ConversationIDs, session.CreatedAt, session.UpdatedAt)
	return err
}

// UpdateSession updates an existing session
func UpdateSession(ctx context.Context, db Execer, session *Session) error {
	session.UpdatedAt = time.Now()

	query := `UPDATE sessions SET current_conversation_id = ?, conversation_ids = ?, updated_at = ? WHERE id = ?`
	_, err := db.ExecContext(ctx, query, session.CurrentConversationID, session.ConversationIDs, session.UpdatedAt, session.ID)
	return err
}

// GetConversationByID retrieves a conversation by its ID
func GetConversationByID(ctx context.Context, db sqlscan.Querier, conversationID string) (*Conversation, error) {
	query := `SELECT id, title, project_directory, created_at, updated_at FROM conversations WHERE id = ?`
	var conv Conversation
	err := sqlscan.Get(ctx, db, &conv, query, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &conv, nil
}

// CreateConversation creates a new conversation in the database
func CreateConversation(ctx context.Context, db Execer, conversation *Conversation) error {
	if conversation.ID == "" {
		conversation.ID = uuid.New().String()
	}
	if conversation.CreatedAt.IsZero() {
		conversation.CreatedAt = time.Now()
	}
	if conversation.UpdatedAt.IsZero() {
		conversation.UpdatedAt = time.Now()
	}

	query := `INSERT INTO conversations (id, title, project_directory, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query, conversation.ID, conversation.Title, conversation.ProjectDirectory, conversation.CreatedAt, conversation.UpdatedAt)
	return err
}

// GetMessagesByConversationID retrieves all messages for a conversation ordered by creation time
func GetMessagesByConversationID(ctx context.Context, db sqlscan.Querier, conversationID string) ([]Message, error) {
	query := `SELECT id, conversation_id, role, provider, model, content, tool_calls, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at`
	var messages []Message
	err := sqlscan.Select(ctx, db, &messages, query, conversationID)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// CreateMessage creates a new message in the database
func CreateMessage(ctx context.Context, db Execer, message *Message) error {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	query := `INSERT INTO messages (id, conversation_id, role, provider, model, content, tool_calls, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query, message.ID, message.ConversationID, message.Role, message.Provider, message.Model, message.Content, message.ToolCalls, message.CreatedAt)
	return err
}

// CreateToolExecution creates a new tool execution record in the database
func CreateToolExecution(ctx context.Context, db Execer, execution *ToolExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}
	if execution.CreatedAt.IsZero() {
		execution.CreatedAt = time.Now()
	}

	query := `INSERT INTO tool_executions (id, message_id, conversation_id, provider, model, tool_name, input, output, error, duration_ms, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query,
		execution.ID,
		execution.MessageID,
		execution.ConversationID,
		execution.Provider,
		execution.Model,
		execution.ToolName,
		execution.Input,
		execution.Output,
		execution.Error,
		execution.DurationMs,
		execution.CreatedAt,
	)
	return err
}