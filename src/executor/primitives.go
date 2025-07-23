package executor

import (
	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
)

// ExecutionState represents the current state of conversation execution
type ExecutionState int

const (
	// StateTextResponse means the LLM provided a text response with no tool calls
	StateTextResponse ExecutionState = iota
	// StateToolCallsNeeded means the LLM wants to execute tool calls
	StateToolCallsNeeded
	// StateToolCallsCompleted means tool calls have been executed and results are ready to send back
	StateToolCallsCompleted
	// StateError means an error occurred during execution
	StateError
)


// ToolExecutionRequest represents a request to execute tool calls
type ToolExecutionRequest struct {
	// Tool calls to execute
	ToolCalls []aisdk.ToolCall

	// Toolbox to use for execution
	Toolbox *agent.DefaultToolbox

	// Session and conversation context
	SessionID      string
	ConversationID string

	// Model info for logging
	Model string

	// Optional callbacks (deprecated - use EventSink)
	Callbacks *Callbacks
	
	// Event sink for handling execution events
	EventSink EventSink
	
	// Current turn number
	TurnNumber int
}