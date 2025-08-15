package executor

import (
	"github.com/elee1766/gofer/src/aisdk"
)

// Callbacks holds optional callback functions for prompt execution
type Callbacks struct {
	// OnToolCall is called before executing a tool
	OnToolCall func(toolCall aisdk.ToolCall) error
	
	// OnToolResult is called after tool execution
	OnToolResult func(toolName string, result *aisdk.ToolResponse, err error) error
}

// ToolCall calls the OnToolCall callback if it's set
func (c *Callbacks) ToolCall(toolCall aisdk.ToolCall) error {
	if c == nil || c.OnToolCall == nil {
		return nil
	}
	return c.OnToolCall(toolCall)
}

// ToolResult calls the OnToolResult callback if it's set
func (c *Callbacks) ToolResult(toolName string, result *aisdk.ToolResponse, err error) error {
	if c == nil || c.OnToolResult == nil {
		return nil
	}
	return c.OnToolResult(toolName, result, err)
}