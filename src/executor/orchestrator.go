package executor

import (
	"context"
	"fmt"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
)

// Step represents a single execution step in a conversation
// The caller is responsible for managing turns and deciding when to stop
type StepRequest struct {
	// The conversation context  
	Conversation *aisdk.Conversation

	// The message to send (for first turn, or nil for tool-result-only turns)
	Message *aisdk.Message

	// Model client to use
	ModelClient aisdk.ModelClient

	// Session and conversation IDs for persistence
	SessionID      string
	ConversationID string

	// Optional toolbox for function calling
	Toolbox *agent.DefaultToolbox

	// Optional callbacks (deprecated - use EventSink)
	Callbacks *Callbacks
	
	// Event sink for handling conversation events
	EventSink EventSink
	
	// Current turn number
	TurnNumber int
}

// StepResult represents the result of a single execution step
type StepResult struct {
	// The current state after this execution step
	State ExecutionState

	// The response from the model (if any)
	Response *StreamResponse

	// Tool calls that need to be executed (if State == StateToolCallsNeeded)
	ToolCalls []aisdk.ToolCall

	// Tool results to send back to model (if State == StateToolCallsCompleted)
	ToolResults []*aisdk.Message

	// Error information (if State == StateError)
	Error error

	// Updated conversation with new messages added
	UpdatedConversation *aisdk.Conversation
}

// Step executes a single conversation step and returns the immediate result
// The caller is responsible for turn tracking and deciding whether to continue
func (s *Service) Step(ctx context.Context, req *StepRequest) (*StepResult, error) {
	if req.ModelClient == nil {
		return &StepResult{State: StateError, Error: ErrModelClientRequired}, nil
	}
	if req.Conversation == nil {
		return &StepResult{State: StateError, Error: fmt.Errorf("conversation is required")}, nil
	}

	// Create event emitter if we have an event sink
	var emitter *EventEmitter
	if req.EventSink != nil {
		emitter = NewEventEmitter(req.EventSink, req.ConversationID, req.TurnNumber)
	}

	// Emit user message event if we have a message
	if req.Message != nil && emitter != nil {
		// Note: We don't have wrapping info here, so we emit basic info
		// The wrapping happens at a higher level in prompt.go
		emitter.EmitUserMessage(req.Message.Content, false, "", 0)
	}

	// Create agent
	agent := &agent.Agent{
		SystemPrompt: s.systemPrompt,
		Model:        req.ModelClient,
		Toolbox:      req.Toolbox,
		Logger:       s.logger,
	}

	// Send message and get stream
	stream, err := agent.SendMessageStream(ctx, req.Conversation, req.Message)
	if err != nil {
		if emitter != nil {
			emitter.EmitError(err, "agent.SendMessageStream")
		}
		return &StepResult{State: StateError, Error: err}, nil
	}
	defer stream.Close()

	// Emit stream start event
	if emitter != nil {
		emitter.EmitAssistantStreamStart(req.ModelClient.GetModelInfo().ID)
	}

	// Process stream chunks
	response, err := s.processStreamChunks(ctx, stream, req.Callbacks, emitter)
	if err != nil {
		if emitter != nil {
			emitter.EmitError(err, "processStreamChunks")
		}
		return &StepResult{State: StateError, Error: err}, nil
	}

	// Emit stream end event
	if emitter != nil {
		emitter.EmitAssistantStreamEnd()
	}

	// Emit assistant message event
	if emitter != nil {
		emitter.EmitAssistantMessage(response.Content, response.ToolCalls, req.ModelClient.GetModelInfo().ID)
	}

	// Save assistant response if we have session info
	if req.ConversationID != "" {
		model := req.ModelClient.GetModelInfo().ID
		if err := s.saveAssistantMessage(ctx, req.ConversationID, model, response); err != nil {
			s.logger.Error("Failed to save assistant message", "error", err)
			// Don't fail the whole operation, just log the error
		}
	}

	// Update conversation with new messages
	updatedConv := &aisdk.Conversation{
		Messages: make([]*aisdk.Message, len(req.Conversation.Messages)),
	}
	copy(updatedConv.Messages, req.Conversation.Messages)

	// Add user message if provided
	if req.Message != nil {
		updatedConv.Messages = append(updatedConv.Messages, req.Message)
	}

	// Add assistant response
	assistantMsg := &aisdk.Message{
		Role:      "assistant",
		Content:   response.Content,
		ToolCalls: response.ToolCalls,
	}
	updatedConv.Messages = append(updatedConv.Messages, assistantMsg)

	// Determine next state based on response
	if len(response.ToolCalls) > 0 {
		return &StepResult{
			State:               StateToolCallsNeeded,
			Response:            response,
			ToolCalls:           response.ToolCalls,
			UpdatedConversation: updatedConv,
		}, nil
	}

	return &StepResult{
		State:               StateTextResponse,
		Response:            response,
		UpdatedConversation: updatedConv,
	}, nil
}

// ExecuteToolCalls executes the given tool calls and returns results ready to send back
func (s *Service) ExecuteToolCalls(ctx context.Context, req *ToolExecutionRequest) (*StepResult, error) {
	// Create event emitter if we have an event sink
	var emitter *EventEmitter
	if req.EventSink != nil {
		emitter = NewEventEmitter(req.EventSink, req.ConversationID, req.TurnNumber)
	}

	// Execute tools using updated helper
	toolResults, err := s.executeTools(ctx, req.Toolbox, req.ConversationID, req.Model, req.Callbacks, req.ToolCalls, emitter)
	if err != nil {
		return &StepResult{State: StateError, Error: err}, nil
	}

	return &StepResult{
		State:       StateToolCallsCompleted,
		ToolResults: toolResults,
	}, nil
}