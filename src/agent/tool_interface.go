package agent

import (
	"context"

	"github.com/elee1766/gofer/src/aisdk"
	jsonschema "github.com/swaggest/jsonschema-go"
)

// Tool is the interface that all tools must implement
type Tool interface {
	// GetType returns the tool type (always "function" for now)
	GetType() string
	
	// GetName returns the tool's name
	GetName() string
	
	// GetDescription returns the tool's description
	GetDescription() string
	
	// GetParameters returns the JSON schema for the tool's parameters
	GetParameters() *jsonschema.Schema
	
	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error)
}