package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/storage"
	"github.com/google/uuid"
)

// saveAssistantMessage saves an assistant message to the database
func (s *Service) saveAssistantMessage(ctx context.Context, conversationID, model string, response *Response) error {
	// Don't save if both content and tool calls are empty
	if response.Content == "" && len(response.ToolCalls) == 0 {
		return nil
	}

	assistantMsg := &storage.Message{
		ConversationID: conversationID,
		Role:           "assistant",
		Provider:       "openrouter",
		Model:          model,
		Content:        response.Content,
	}

	// Store tool calls as JSON if present
	if len(response.ToolCalls) > 0 {
		toolCallsJSON, err := json.Marshal(response.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsStr := string(toolCallsJSON)
		assistantMsg.ToolCalls = &toolCallsStr
	}

	return storage.CreateMessage(ctx, s.database, assistantMsg)
}

// executeTools executes the given tool calls and returns the results
func (s *Service) executeTools(ctx context.Context, toolbox *agent.DefaultToolbox, conversationID, model string, callbacks *Callbacks, toolCalls []aisdk.ToolCall, emitter *EventEmitter) ([]*aisdk.Message, error) {
	var toolResults []*aisdk.Message

	for _, toolCall := range toolCalls {
		s.logger.Debug("Executing tool", "name", toolCall.Function.Name, "id", toolCall.ID)

		// Emit tool call request event
		if emitter != nil {
			emitter.EmitToolCallRequest(toolCall)
		}

		// Legacy callback support
		if callbacks != nil {
			if err := callbacks.ToolCall(toolCall); err != nil {
				return nil, fmt.Errorf("tool call callback failed: %w", err)
			}
		}

		// Check if toolbox is available
		if toolbox == nil {
			// No toolbox, create error result
			toolResults = append(toolResults, &aisdk.Message{
				Role:       "tool",
				Content:    "Tool execution not available: no toolbox configured",
				Name:       toolCall.Function.Name,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		// Find the tool
		tool, found := toolbox.GetTool(toolCall.Function.Name)
		if !found {
			// Tool not found, create error result
			toolResults = append(toolResults, &aisdk.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("Tool not found: %s", toolCall.Function.Name),
				Name:       toolCall.Function.Name,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		// Execute the tool
		startTime := time.Now()
		result, execErr := tool.Execute(ctx, &toolCall)
		duration := time.Since(startTime)

		// Save tool execution to database
		var output, errorStr string
		if execErr != nil {
			errorStr = execErr.Error()
			output = fmt.Sprintf("Error: %s", execErr.Error())
		} else if result != nil {
			output = string(result.Content)
		}

		// Emit tool response/error event
		if emitter != nil {
			if execErr != nil {
				emitter.EmitToolCallError(toolCall.Function.Name, toolCall.ID, execErr, duration)
			} else {
				emitter.EmitToolCallResponse(toolCall.Function.Name, toolCall.ID, result, duration)
			}
		}

		// Get the assistant message ID from the database
		// This is a bit hacky, but we need it for the tool execution record
		assistantMsgID := uuid.New().String() // For now, generate a new ID

		toolExec := &storage.ToolExecution{
			MessageID:      assistantMsgID,
			ConversationID: conversationID,
			Provider:       "openrouter",
			Model:          model,
			ToolName:       toolCall.Function.Name,
			Input:          string(toolCall.Function.Arguments),
			Output:         output,
			Error:          errorStr,
			DurationMs:     duration.Milliseconds(),
		}
		err := storage.CreateToolExecution(ctx, s.database, toolExec)
		if err != nil {
			s.logger.Error("Failed to save tool execution", "error", err)
		}

		// Callback after tool execution
		if callbacks != nil {
			if err := callbacks.ToolResult(toolCall.Function.Name, result, execErr); err != nil {
				return nil, fmt.Errorf("tool result callback failed: %w", err)
			}
		}

		// Create tool result message
		toolResults = append(toolResults, &aisdk.Message{
			Role:       "tool",
			Content:    output,
			Name:       toolCall.Function.Name,
			ToolCallID: toolCall.ID,
		})
	}

	return toolResults, nil
}

