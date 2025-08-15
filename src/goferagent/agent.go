package goferagent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
)

// Config holds the configuration for the Gofer agent
type Config struct {
	// Model to use for conversations
	Model string

	// System prompt that defines the agent's behavior
	SystemPrompt string

	// Tools available to the agent
	Tools []agent.Tool

	// Model parameters
	Temperature *float64
	MaxTokens   *int
	TopP        *float64

	// Whether tools are enabled
	EnableTools bool

	// Logger for the agent
	Logger *slog.Logger
}

// GoferAgent implements the Agent interface for our specific use case
type GoferAgent struct {
	config      Config
	modelClient aisdk.ModelClient
	logger      *slog.Logger
}

// NewGoferAgent creates a new Gofer agent with the given configuration
func NewGoferAgent(config Config, modelClient aisdk.ModelClient) *GoferAgent {
	return &GoferAgent{
		config:      config,
		modelClient: modelClient,
		logger:      config.Logger,
	}
}

// SendMessage sends a message and returns a response
func (a *GoferAgent) SendMessage(ctx context.Context, messages []aisdk.Message) (*aisdk.Message, error) {
	// Convert messages to pointers
	msgPtrs := make([]*aisdk.Message, len(messages))
	for i := range messages {
		msgPtrs[i] = &messages[i]
	}
	
	// Build the request
	req := &aisdk.ChatCompletionRequest{
		Messages:    msgPtrs,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
		TopP:        a.config.TopP,
	}

	// Add tools if enabled
	if a.config.EnableTools && len(a.config.Tools) > 0 {
		req.Tools = agent.ToChatTools(a.config.Tools)
		req.ToolChoice = "auto"
	}

	a.logger.Debug("Starting message to AI",
		"model", a.config.Model,
		"message_count", len(messages),
	)

	// Send the request
	response, err := a.modelClient.CreateChatCompletion(ctx, req)
	if err != nil {
		a.logger.Error("Failed to send message", "error", err)
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	a.logger.Debug("got response from ai", "response", response)

	return &response.Choices[0].Message, nil
}
