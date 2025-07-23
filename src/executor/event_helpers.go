package executor

import (
	"time"

	"github.com/elee1766/gofer/src/aisdk"
)

// EventEmitter helps emit events with common fields
type EventEmitter struct {
	sink           EventSink
	conversationID string
	turnNumber     int
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter(sink EventSink, conversationID string, turnNumber int) *EventEmitter {
	return &EventEmitter{
		sink:           sink,
		conversationID: conversationID,
		turnNumber:     turnNumber,
	}
}

// createBaseEvent creates a base event with common fields
func (e *EventEmitter) createBaseEvent(eventType EventType) BaseEvent {
	return BaseEvent{
		Type:           eventType,
		Timestamp:      time.Now(),
		ConversationID: e.conversationID,
		TurnNumber:     e.turnNumber,
	}
}

// EmitUserMessage emits a user message event
func (e *EventEmitter) EmitUserMessage(message string, isWrapped bool, originalText string, turnsRemaining int) error {
	if e.sink == nil {
		return nil
	}
	
	event := &UserMessageEvent{
		BaseEvent:      e.createBaseEvent(EventUserMessage),
		Message:        message,
		IsWrapped:      isWrapped,
		OriginalText:   originalText,
		TurnsRemaining: turnsRemaining,
	}
	
	return e.sink.Send(event)
}

// EmitAssistantStreamStart emits the start of assistant streaming
func (e *EventEmitter) EmitAssistantStreamStart(model string) error {
	if e.sink == nil {
		return nil
	}
	
	event := &AssistantStreamStartEvent{
		BaseEvent: e.createBaseEvent(EventAssistantStreamStart),
		Model:     model,
	}
	
	return e.sink.Send(event)
}

// EmitAssistantStreamChunk emits a chunk of streamed content
func (e *EventEmitter) EmitAssistantStreamChunk(content string) error {
	if e.sink == nil {
		return nil
	}
	
	event := &AssistantStreamChunkEvent{
		BaseEvent: e.createBaseEvent(EventAssistantStreamChunk),
		Content:   content,
	}
	
	return e.sink.Send(event)
}

// EmitAssistantStreamEnd emits the end of assistant streaming
func (e *EventEmitter) EmitAssistantStreamEnd() error {
	if e.sink == nil {
		return nil
	}
	
	event := &AssistantStreamEndEvent{
		BaseEvent: e.createBaseEvent(EventAssistantStreamEnd),
	}
	
	return e.sink.Send(event)
}

// EmitAssistantMessage emits a complete assistant message
func (e *EventEmitter) EmitAssistantMessage(content string, toolCalls []aisdk.ToolCall, model string) error {
	if e.sink == nil {
		return nil
	}
	
	event := &AssistantMessageEvent{
		BaseEvent: e.createBaseEvent(EventAssistantMessage),
		Content:   content,
		ToolCalls: toolCalls,
		Model:     model,
	}
	
	return e.sink.Send(event)
}

// EmitToolCallRequest emits a tool call request
func (e *EventEmitter) EmitToolCallRequest(toolCall aisdk.ToolCall) error {
	if e.sink == nil {
		return nil
	}
	
	event := &ToolCallRequestEvent{
		BaseEvent: e.createBaseEvent(EventToolCallRequest),
		ToolCall:  toolCall,
	}
	
	return e.sink.Send(event)
}

// EmitToolCallResponse emits a successful tool call response
func (e *EventEmitter) EmitToolCallResponse(toolName, toolID string, response *aisdk.ToolResponse, duration time.Duration) error {
	if e.sink == nil {
		return nil
	}
	
	event := &ToolCallResponseEvent{
		BaseEvent: e.createBaseEvent(EventToolCallResponse),
		ToolName:  toolName,
		ToolID:    toolID,
		Response:  response,
		Duration:  duration,
	}
	
	return e.sink.Send(event)
}

// EmitToolCallError emits a failed tool call
func (e *EventEmitter) EmitToolCallError(toolName, toolID string, err error, duration time.Duration) error {
	if e.sink == nil {
		return nil
	}
	
	event := &ToolCallErrorEvent{
		BaseEvent: e.createBaseEvent(EventToolCallError),
		ToolName:  toolName,
		ToolID:    toolID,
		Error:     err,
		Duration:  duration,
	}
	
	return e.sink.Send(event)
}

// EmitSystemMessage emits a system message
func (e *EventEmitter) EmitSystemMessage(message, purpose string) error {
	if e.sink == nil {
		return nil
	}
	
	event := &SystemMessageEvent{
		BaseEvent: e.createBaseEvent(EventSystemMessage),
		Message:   message,
		Purpose:   purpose,
	}
	
	return e.sink.Send(event)
}

// EmitError emits an error event
func (e *EventEmitter) EmitError(err error, context string) error {
	if e.sink == nil {
		return nil
	}
	
	event := &ErrorEvent{
		BaseEvent: e.createBaseEvent(EventError),
		Error:     err,
		Context:   context,
	}
	
	return e.sink.Send(event)
}

// EmitTurnComplete emits a turn completion event
func (e *EventEmitter) EmitTurnComplete(turnsRemaining int, state ExecutionState) error {
	if e.sink == nil {
		return nil
	}
	
	event := &TurnCompleteEvent{
		BaseEvent:      e.createBaseEvent(EventTurnComplete),
		TurnsRemaining: turnsRemaining,
		State:          state,
	}
	
	return e.sink.Send(event)
}

// EmitConversationComplete emits a conversation completion event
func (e *EventEmitter) EmitConversationComplete(reason string, totalTurns, turnsRemaining int) error {
	if e.sink == nil {
		return nil
	}
	
	event := &ConversationCompleteEvent{
		BaseEvent:      e.createBaseEvent(EventConversationComplete),
		Reason:         reason,
		TotalTurns:     totalTurns,
		TurnsRemaining: turnsRemaining,
	}
	
	return e.sink.Send(event)
}