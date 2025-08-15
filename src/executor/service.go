package executor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/storage"
)

// Service handles prompt execution with all necessary dependencies
type Service struct {
	database     *sql.DB
	projectDir   string
	logger       *slog.Logger
	systemPrompt string
	maxTurns     int
}

// ServiceConfig holds configuration for creating a new Service
type ServiceConfig struct {
	Database     *sql.DB
	ProjectDir   string
	SystemPrompt string
	MaxTurns     int
	Logger       *slog.Logger
}

// NewService creates a new prompt service
func NewService(config ServiceConfig) *Service {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	// Default max turns if not specified
	if config.MaxTurns <= 0 {
		config.MaxTurns = 3
	}

	return &Service{
		database:     config.Database,
		projectDir:   config.ProjectDir,
		logger:       config.Logger,
		systemPrompt: config.SystemPrompt,
		maxTurns:     config.MaxTurns,
	}
}



// getOrCreateConversation retrieves or creates a conversation for the session
func (s *Service) getOrCreateConversation(ctx context.Context, session *storage.Session) (*storage.Conversation, error) {
	// Try to use session's current conversation
	if session.CurrentConversationID != nil {
		conversation, err := storage.GetConversationByID(ctx, s.database, *session.CurrentConversationID)
		if err != nil {
			return nil, err
		}
		if conversation != nil {
			return conversation, nil
		}
	}

	// Create new conversation
	conversation := &storage.Conversation{
		Title:            "New Conversation",
		ProjectDirectory: s.projectDir,
	}
	err := storage.CreateConversation(ctx, s.database, conversation)
	if err != nil {
		return nil, err
	}

	// Update session
	session.ConversationIDs = append(session.ConversationIDs, conversation.ID)
	session.CurrentConversationID = &conversation.ID
	err = storage.UpdateSession(ctx, s.database, session)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}



// GetOrCreateSession retrieves or creates a session based on provided parameters
func (s *Service) GetOrCreateSession(ctx context.Context, sessionID string, resume bool) (*storage.Session, error) {
	if sessionID != "" {
		session, err := storage.GetSessionByID(ctx, s.database, sessionID)
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
		}
		return session, nil
	}

	if resume {
		session, err := storage.GetLatestSession(ctx, s.database)
		if err != nil {
			return nil, err
		}
		if session != nil {
			return session, nil
		}
		// No sessions exist, create new one
	}

	// Create new session
	session := &storage.Session{}
	err := storage.CreateSession(ctx, s.database, session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// GetOrCreateConversation retrieves or creates a conversation for the session
func (s *Service) GetOrCreateConversation(ctx context.Context, session *storage.Session) (*storage.Conversation, error) {
	return s.getOrCreateConversation(ctx, session)
}

// BuildConversationFromDB builds an aisdk.Conversation from database messages
func (s *Service) BuildConversationFromDB(ctx context.Context, conversation *storage.Conversation, systemPrompt string) (*aisdk.Conversation, error) {
	// Get existing messages
	messages, err := storage.GetMessagesByConversationID(ctx, s.database, conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Build conversation
	return buildAISDKConversation(conversation, messages, systemPrompt), nil
}

// SaveUserMessage saves a user message to the database
func (s *Service) SaveUserMessage(ctx context.Context, conversationID, content string) error {
	userMsg := &storage.Message{
		ConversationID: conversationID,
		Role:           "user",
		Content:        content,
	}
	return storage.CreateMessage(ctx, s.database, userMsg)
}



