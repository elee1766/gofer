package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/storage"
)

// StreamResponse represents the aggregated response from a streaming completion
type StreamResponse struct {
	Content      string
	ToolCalls    []aisdk.ToolCall
	FinishReason string
}

// processStreamChunks reads and aggregates chunks from a stream into a complete response
func processStreamChunks(ctx context.Context, stream aisdk.StreamInterface, logger *slog.Logger, onContent func(string) error, emitter *EventEmitter) (*StreamResponse, error) {
	var responseContent strings.Builder
	var toolCalls []aisdk.ToolCall
	var finishReason string

	for {
		chunk, err := stream.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to read stream: %w", err)
		}

		// Process chunk
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			
			// Accumulate content
			if choice.Delta != nil && choice.Delta.Content != "" {
				responseContent.WriteString(choice.Delta.Content)
				
				// Emit stream chunk event
				if emitter != nil {
					emitter.EmitAssistantStreamChunk(choice.Delta.Content)
				}
				
				// Call content callback if provided (legacy support)
				if onContent != nil {
					if err := onContent(choice.Delta.Content); err != nil {
						return nil, fmt.Errorf("content callback failed: %w", err)
					}
				}
			}
			
			// Track finish reason
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			// Handle tool calls - in streaming, tool calls might come in multiple chunks
			if choice.Message.ToolCalls != nil && len(choice.Message.ToolCalls) > 0 {
				// Complete tool calls from message
				toolCalls = choice.Message.ToolCalls
				logger.Debug("Received complete tool calls from message", "count", len(toolCalls))
			} else if choice.Delta != nil && len(choice.Delta.ToolCalls) > 0 {
				// Accumulate delta tool calls
				for _, deltaTC := range choice.Delta.ToolCalls {
					// Find existing tool call by ID and update it
					found := false
					for i := range toolCalls {
						if toolCalls[i].ID == deltaTC.ID {
							// Update existing tool call
							if deltaTC.Function.Name != "" {
								toolCalls[i].Function.Name = deltaTC.Function.Name
							}
							if len(deltaTC.Function.Arguments) > 0 {
								// Append params (they might come in chunks)
								toolCalls[i].Function.Arguments = append(toolCalls[i].Function.Arguments, deltaTC.Function.Arguments...)
							}
							found = true
							break
						}
					}
					
					if !found && (deltaTC.ID != "" || deltaTC.Function.Name != "" || len(deltaTC.Function.Arguments) > 0) {
						// New tool call
						toolCalls = append(toolCalls, deltaTC)
					}
				}
			}
		}
	}

	return &StreamResponse{
		Content:      responseContent.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}, nil
}

// buildAISDKConversation creates an aisdk.Conversation from storage messages
func buildAISDKConversation(conversation *storage.Conversation, messages []storage.Message, systemPrompt string) *aisdk.Conversation {
	aisdkConv := &aisdk.Conversation{
		ID:           conversation.ID,
		SystemPrompt: systemPrompt,
		Messages:     make([]*aisdk.Message, 0, len(messages)+1),
		CreatedAt:    conversation.CreatedAt,
	}

	// Add system prompt if no messages exist
	if len(messages) == 0 && systemPrompt != "" {
		aisdkConv.Messages = append(aisdkConv.Messages, &aisdk.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	} else {
		// Convert existing messages
		for _, msg := range messages {
			aisdkMsg := &aisdk.Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
			
			// Parse tool calls if present
			if msg.ToolCalls != nil && *msg.ToolCalls != "" {
				var toolCalls []aisdk.ToolCall
				if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err == nil {
					aisdkMsg.ToolCalls = toolCalls
				}
			}
			
			aisdkConv.Messages = append(aisdkConv.Messages, aisdkMsg)
		}
	}

	return aisdkConv
}