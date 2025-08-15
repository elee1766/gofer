package executor

import (
	"encoding/json"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/storage"
)

// Response represents a response from the model
type Response struct {
	Content   string
	ToolCalls []aisdk.ToolCall
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