package aisdk

import (
	"sync"
	"time"
)

// Conversation represents an ongoing conversation with an AI agent.
type Conversation struct {
	ID            string
	Messages      []*Message
	SystemPrompt  string
	Tools         []*ChatTool
	CreatedAt     time.Time
	LastMessageAt time.Time
	TurnCount     int
	MaxTurns      int
	mu            sync.Mutex
}
