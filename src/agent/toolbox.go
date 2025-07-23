package agent

import (
	"context"
	"fmt"

	"github.com/elee1766/gofer/src/aisdk"
)

// ToolExecutor is a function type for tool execution
type ToolExecutor func(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error)

// DefaultToolbox is a type alias for backward compatibility using the Tool interface
type DefaultToolbox = Toolbox[Tool]

// Toolbox handles tool/function calling functionality.
type Toolbox[T Tool] struct {
	tools      map[string]T
	middleware []ToolMiddleware
}

// ToolMiddleware is a function that wraps a ToolExecutor to add functionality.
type ToolMiddleware func(next ToolExecutor) ToolExecutor

// ToolContext provides context for tool execution middleware.
type ToolContext struct {
	Context  context.Context
	ToolName string
	Tool     Tool
}

// NewToolbox creates a new tool manager.
func NewToolbox[T Tool]() *Toolbox[T] {
	return &Toolbox[T]{
		tools: make(map[string]T),
	}
}

// RegisterTool registers a tool.
func (tm *Toolbox[T]) RegisterTool(tool T) error {
	if tool.GetName() == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Check for duplicate tool names
	if _, exists := tm.tools[tool.GetName()]; exists {
		return fmt.Errorf("tool %s is already registered", tool.GetName())
	}

	tm.tools[tool.GetName()] = tool
	return nil
}

// RegisterMiddleware registers middleware that will be applied to all tool executions.
// Middleware is applied in the order it's registered (first registered = outermost layer).
func (tm *Toolbox[T]) RegisterMiddleware(middleware ToolMiddleware) {
	tm.middleware = append(tm.middleware, middleware)
}

// ClearMiddleware removes all registered middleware.
func (tm *Toolbox[T]) ClearMiddleware() {
	tm.middleware = nil
}

func (tm *Toolbox[T]) ToolMap() map[string]T {
	return tm.tools
}

// Tools returns the list of available tools
func (tm *Toolbox[T]) Tools() []T {
	out := make([]T, 0, len(tm.tools))
	for _, tool := range tm.tools {
		out = append(out, tool)
	}
	return out
}

// ExecuteTool executes a tool call with middleware applied.
func (tm *Toolbox[T]) ExecuteTool(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
	tool, exists := tm.tools[call.Function.Name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", call.Function.Name)
	}

	// Create a wrapper for the tool's Execute method
	toolExecutor := ToolExecutor(func(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
		return tool.Execute(ctx, call)
	})

	// Apply middleware chain
	finalExecutor := toolExecutor
	for i := len(tm.middleware) - 1; i >= 0; i-- {
		finalExecutor = tm.middleware[i](finalExecutor)
	}

	return finalExecutor(ctx, call)
}

// GetTool returns a specific tool by name.
func (tm *Toolbox[T]) GetTool(name string) (T, bool) {
	tool, exists := tm.tools[name]
	return tool, exists
}

// HasTool checks if a tool is available.
func (tm *Toolbox[T]) HasTool(name string) bool {
	_, exists := tm.tools[name]
	return exists
}

// Common middleware implementations

// LoggingMiddleware logs tool execution details.
func LoggingMiddleware(logger interface {
	Info(msg string, args ...interface{})
}) ToolMiddleware {
	return func(next ToolExecutor) ToolExecutor {
		return func(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
			logger.Info("executing tool", "tool", call.Function.Name, "params", string(call.Function.Arguments))
			result, err := next(ctx, call)
			if err != nil {
				logger.Info("tool execution failed", "error", err)
			} else {
				logger.Info("tool execution completed successfully")
			}
			return result, err
		}
	}
}
