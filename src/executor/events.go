package executor

import (
	"fmt"
	"time"

	"github.com/elee1766/gofer/src/aisdk"
)

// EventType represents the type of conversation event
type EventType string

const (
	// User events
	EventUserMessage EventType = "user_message"
	
	// Assistant events
	EventAssistantStreamStart EventType = "assistant_stream_start"
	EventAssistantStreamChunk EventType = "assistant_stream_chunk"
	EventAssistantStreamEnd   EventType = "assistant_stream_end"
	EventAssistantMessage     EventType = "assistant_message"
	
	// Tool events
	EventToolCallRequest  EventType = "tool_call_request"
	EventToolCallResponse EventType = "tool_call_response"
	EventToolCallError    EventType = "tool_call_error"
	
	// System events
	EventSystemMessage EventType = "system_message"
	EventError         EventType = "error"
	EventTurnComplete  EventType = "turn_complete"
	EventConversationComplete EventType = "conversation_complete"
)

// ConversationEvent is the base interface for all conversation events
type ConversationEvent interface {
	GetType() EventType
	GetTimestamp() time.Time
	GetConversationID() string
	GetTurnNumber() int
}

// BaseEvent contains common fields for all events
type BaseEvent struct {
	Type           EventType `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	ConversationID string    `json:"conversation_id"`
	TurnNumber     int       `json:"turn_number"`
}

func (e BaseEvent) GetType() EventType          { return e.Type }
func (e BaseEvent) GetTimestamp() time.Time     { return e.Timestamp }
func (e BaseEvent) GetConversationID() string   { return e.ConversationID }
func (e BaseEvent) GetTurnNumber() int          { return e.TurnNumber }

// UserMessageEvent represents a user message
type UserMessageEvent struct {
	BaseEvent
	Message       string `json:"message"`
	IsWrapped     bool   `json:"is_wrapped"`      // Whether message was wrapped with context
	OriginalText  string `json:"original_text"`   // Original text before wrapping
	TurnsRemaining int   `json:"turns_remaining"`
}

// AssistantStreamStartEvent represents the start of assistant streaming
type AssistantStreamStartEvent struct {
	BaseEvent
	Model string `json:"model"`
}

// AssistantStreamChunkEvent represents a chunk of streamed content
type AssistantStreamChunkEvent struct {
	BaseEvent
	Content string `json:"content"`
}

// AssistantStreamEndEvent represents the end of assistant streaming
type AssistantStreamEndEvent struct {
	BaseEvent
}

// AssistantMessageEvent represents a complete assistant message
type AssistantMessageEvent struct {
	BaseEvent
	Content   string           `json:"content"`
	ToolCalls []aisdk.ToolCall `json:"tool_calls,omitempty"`
	Model     string           `json:"model"`
}

// ToolCallRequestEvent represents a tool call request
type ToolCallRequestEvent struct {
	BaseEvent
	ToolCall aisdk.ToolCall `json:"tool_call"`
}

// ToolCallResponseEvent represents a successful tool call response
type ToolCallResponseEvent struct {
	BaseEvent
	ToolName string              `json:"tool_name"`
	ToolID   string              `json:"tool_id"`
	Response *aisdk.ToolResponse `json:"response"`
	Duration time.Duration       `json:"duration"`
}

// ToolCallErrorEvent represents a failed tool call
type ToolCallErrorEvent struct {
	BaseEvent
	ToolName string        `json:"tool_name"`
	ToolID   string        `json:"tool_id"`
	Error    error         `json:"error"`
	Duration time.Duration `json:"duration"`
}

// SystemMessageEvent represents system messages (like continuation prompts)
type SystemMessageEvent struct {
	BaseEvent
	Message string `json:"message"`
	Purpose string `json:"purpose"` // e.g., "continuation", "warning", "info"
}

// ErrorEvent represents an error in the conversation
type ErrorEvent struct {
	BaseEvent
	Error   error  `json:"error"`
	Context string `json:"context"` // Where the error occurred
}

// TurnCompleteEvent represents the completion of a conversation turn
type TurnCompleteEvent struct {
	BaseEvent
	TurnsRemaining int           `json:"turns_remaining"`
	State          ExecutionState `json:"state"`
}

// ConversationCompleteEvent represents the end of a conversation
type ConversationCompleteEvent struct {
	BaseEvent
	Reason         string `json:"reason"` // "max_turns", "task_complete", "error"
	TotalTurns     int    `json:"total_turns"`
	TurnsRemaining int    `json:"turns_remaining"`
}

// EventSink is the interface for handling conversation events
type EventSink interface {
	// Send sends an event to the sink (non-blocking)
	Send(event ConversationEvent) error
	
	// Close closes the event sink
	Close() error
}

// EventProcessor processes conversation events
type EventProcessor interface {
	// Process handles a single event
	Process(event ConversationEvent) error
	
	// Close cleans up any resources
	Close() error
}

// ChannelEventSink implements EventSink using Go channels
type ChannelEventSink struct {
	events     chan ConversationEvent
	processors []EventProcessor
	done       chan struct{}
}

// NewChannelEventSink creates a new channel-based event sink
func NewChannelEventSink(bufferSize int, processors ...EventProcessor) *ChannelEventSink {
	sink := &ChannelEventSink{
		events:     make(chan ConversationEvent, bufferSize),
		processors: processors,
		done:       make(chan struct{}),
	}
	
	// Start processing events
	go sink.processEvents()
	
	return sink
}

// Send sends an event to the sink
func (s *ChannelEventSink) Send(event ConversationEvent) error {
	select {
	case s.events <- event:
		return nil
	case <-s.done:
		return fmt.Errorf("event sink is closed")
	}
}

// Close closes the event sink
func (s *ChannelEventSink) Close() error {
	close(s.events)
	<-s.done
	
	// Close all processors
	for _, p := range s.processors {
		if err := p.Close(); err != nil {
			// Log error but continue closing others
			fmt.Printf("Error closing processor: %v\n", err)
		}
	}
	
	return nil
}

// processEvents processes events from the channel
func (s *ChannelEventSink) processEvents() {
	defer close(s.done)
	
	for event := range s.events {
		for _, processor := range s.processors {
			if err := processor.Process(event); err != nil {
				// Log error but continue processing
				fmt.Printf("Error processing event: %v\n", err)
			}
		}
	}
}