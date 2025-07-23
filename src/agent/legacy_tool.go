package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elee1766/gofer/src/aisdk"
	jsonschema "github.com/swaggest/jsonschema-go"
)

// LegacyTool represents the old Tool struct that implements the Tool interface
type LegacyTool struct {
	Type     string              `json:"type"` // Always "function" for function tools
	Function aisdk.ToolFunction  `json:"function"`
	Executor aisdk.ToolExecutor  `json:"-"` // Execution function, not serialized
}

// GetType returns the tool type
func (t *LegacyTool) GetType() string {
	return t.Type
}

// GetName returns the tool's name
func (t *LegacyTool) GetName() string {
	return t.Function.Name
}

// GetDescription returns the tool's description
func (t *LegacyTool) GetDescription() string {
	return t.Function.Description
}

// GetParameters returns the tool's parameter schema
func (t *LegacyTool) GetParameters() *jsonschema.Schema {
	return t.Function.Parameters
}

// Execute runs the tool
func (t *LegacyTool) Execute(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
	if t.Executor == nil {
		return nil, fmt.Errorf("tool %s has no executor", t.GetName())
	}
	return t.Executor(ctx, call)
}

// MarshalJSON implements custom JSON marshaling
func (t *LegacyTool) MarshalJSON() ([]byte, error) {
	// Marshal only the type and function fields
	return json.Marshal(struct {
		Type     string             `json:"type"`
		Function aisdk.ToolFunction `json:"function"`
	}{
		Type:     t.Type,
		Function: t.Function,
	})
}

// Ensure LegacyTool implements the Tool interface
var _ Tool = (*LegacyTool)(nil)