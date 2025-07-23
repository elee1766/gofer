package executor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ConsoleProcessorConfig configures the console event processor
type ConsoleProcessorConfig struct {
	ShowTimestamps      bool
	ShowTurnNumbers     bool
	ShowToolArguments   bool
	ShowToolResults     bool
	ShowIntermediateAI  bool
	RawMode            bool
	StreamMode         bool
	MaxResultPreview    int // Max characters to show in result preview
}

// ConsoleEventProcessor processes events and outputs to console
type ConsoleEventProcessor struct {
	config ConsoleProcessorConfig
}

// NewConsoleEventProcessor creates a new console event processor
func NewConsoleEventProcessor(config ConsoleProcessorConfig) *ConsoleEventProcessor {
	// Set defaults
	if config.MaxResultPreview == 0 {
		config.MaxResultPreview = 200
	}
	
	return &ConsoleEventProcessor{
		config: config,
	}
}

// Process handles a single event
func (p *ConsoleEventProcessor) Process(event ConversationEvent) error {
	// Skip output in raw mode for most events
	if p.config.RawMode {
		// In raw mode, only output assistant final messages
		if msg, ok := event.(*AssistantMessageEvent); ok && len(msg.ToolCalls) == 0 {
			fmt.Print(msg.Content)
		}
		return nil
	}
	
	switch e := event.(type) {
	case *UserMessageEvent:
		p.processUserMessage(e)
		
	case *AssistantStreamStartEvent:
		// No output needed for stream start
		
	case *AssistantStreamChunkEvent:
		if p.config.StreamMode {
			fmt.Print(e.Content)
		}
		
	case *AssistantStreamEndEvent:
		// No output needed for stream end
		
	case *AssistantMessageEvent:
		p.processAssistantMessage(e)
		
	case *ToolCallRequestEvent:
		p.processToolCallRequest(e)
		
	case *ToolCallResponseEvent:
		p.processToolCallResponse(e)
		
	case *ToolCallErrorEvent:
		p.processToolCallError(e)
		
	case *SystemMessageEvent:
		p.processSystemMessage(e)
		
	case *ErrorEvent:
		p.processError(e)
		
	case *TurnCompleteEvent:
		// Could add turn summary if needed
		
	case *ConversationCompleteEvent:
		p.processConversationComplete(e)
	}
	
	return nil
}

// Close cleans up resources
func (p *ConsoleEventProcessor) Close() error {
	return nil
}

// processUserMessage handles user message events
func (p *ConsoleEventProcessor) processUserMessage(e *UserMessageEvent) {
	// In non-streaming mode, we might want to echo the user message
	// Skip this if it's a wrapped continuation message
	if e.IsWrapped && e.OriginalText == "" {
		// This is a system-generated continuation prompt, skip it
		return
	}
}

// processAssistantMessage handles assistant message events
func (p *ConsoleEventProcessor) processAssistantMessage(e *AssistantMessageEvent) {
	if !p.config.StreamMode {
		// Only show intermediate AI responses if configured
		if len(e.ToolCalls) > 0 && p.config.ShowIntermediateAI && e.Content != "" {
			fmt.Printf("\n💭 Assistant: %s\n", e.Content)
		} else if len(e.ToolCalls) == 0 {
			// Final response
			fmt.Println(e.Content)
		}
	}
}

// processToolCallRequest handles tool call request events
func (p *ConsoleEventProcessor) processToolCallRequest(e *ToolCallRequestEvent) {
	fmt.Printf("\n🔧 Calling tool: %s\n", e.ToolCall.Function.Name)
	
	if p.config.ShowToolArguments {
		// Pretty-print JSON arguments
		var prettyArgs interface{}
		if err := json.Unmarshal(e.ToolCall.Function.Arguments, &prettyArgs); err == nil {
			if prettyJSON, err := json.MarshalIndent(prettyArgs, "   ", "  "); err == nil {
				fmt.Printf("   Arguments:\n   %s\n", string(prettyJSON))
			} else {
				fmt.Printf("   Arguments: %s\n", string(e.ToolCall.Function.Arguments))
			}
		} else {
			fmt.Printf("   Arguments: %s\n", string(e.ToolCall.Function.Arguments))
		}
	}
}

// processToolCallResponse handles tool call response events
func (p *ConsoleEventProcessor) processToolCallResponse(e *ToolCallResponseEvent) {
	fmt.Printf("   ✓ Tool completed")
	if e.Duration > 0 {
		fmt.Printf(" (%v)", e.Duration.Round(10*time.Millisecond))
	}
	fmt.Println()
	
	if p.config.ShowToolResults && e.Response != nil && len(e.Response.Content) > 0 {
		preview := string(e.Response.Content)
		if len(preview) > p.config.MaxResultPreview {
			preview = preview[:p.config.MaxResultPreview] + "..."
		}
		// Clean up the preview (remove newlines for single-line display)
		preview = strings.ReplaceAll(preview, "\n", " ")
		fmt.Printf("   Result preview: %s\n", preview)
	}
}

// processToolCallError handles tool call error events
func (p *ConsoleEventProcessor) processToolCallError(e *ToolCallErrorEvent) {
	fmt.Printf("   ❌ Tool failed: %v", e.Error)
	if e.Duration > 0 {
		fmt.Printf(" (%v)", e.Duration.Round(10*time.Millisecond))
	}
	fmt.Println()
}

// processSystemMessage handles system message events
func (p *ConsoleEventProcessor) processSystemMessage(e *SystemMessageEvent) {
	// Only show certain system messages
	switch e.Purpose {
	case "continuation":
		if p.config.ShowIntermediateAI {
			fmt.Printf("\n📋 System: %s\n", e.Message)
		}
	case "warning":
		fmt.Printf("\n⚠️  %s\n", e.Message)
	case "info":
		fmt.Printf("\nℹ️  %s\n", e.Message)
	}
}

// processError handles error events
func (p *ConsoleEventProcessor) processError(e *ErrorEvent) {
	fmt.Printf("\n❌ Error in %s: %v\n", e.Context, e.Error)
}

// processConversationComplete handles conversation completion
func (p *ConsoleEventProcessor) processConversationComplete(e *ConversationCompleteEvent) {
	if e.Reason == "max_turns" && e.TurnsRemaining == 0 {
		fmt.Printf("\n⚠️  Maximum turns reached (%d turns used)\n", e.TotalTurns)
	}
}