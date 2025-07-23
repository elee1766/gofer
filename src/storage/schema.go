package storage

import "time"

// Conversation keyed by project dir, no model
type Conversation struct {
	ID               string    `json:"id" db:"id"`
	Title            string    `json:"title" db:"title"`
	ProjectDirectory string    `json:"project_directory" db:"project_directory"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type Message struct {
	ID             string    `json:"id" db:"id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	Role           string    `json:"role" db:"role"`
	Provider       string    `json:"provider" db:"provider"`
	Model          string    `json:"model" db:"model"`
	Content        string    `json:"content" db:"content"`
	ToolCalls      *string   `json:"tool_calls,omitempty" db:"tool_calls"` // JSON array of tool calls
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type ToolExecution struct {
	ID             string    `json:"id" db:"id"`
	MessageID      string    `json:"message_id" db:"message_id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	Provider       string    `json:"provider" db:"provider"`
	Model          string    `json:"model" db:"model"`
	ToolName       string    `json:"tool_name" db:"tool_name"`
	Input          string    `json:"input" db:"input"`
	Output         string    `json:"output" db:"output"`
	Error          string    `json:"error" db:"error"`
	DurationMs     int64     `json:"duration_ms" db:"duration_ms"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type Session struct {
	ID                    string          `json:"id" db:"id"`
	CurrentConversationID *string         `json:"current_conversation_id,omitempty" db:"current_conversation_id"`
	ConversationIDs       JSONStringArray `json:"conversation_ids" db:"conversation_ids"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}

type Setting struct {
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
