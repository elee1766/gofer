package aisdk

import (
	jsonschema "github.com/swaggest/jsonschema-go"
)

// ChatTool represents a tool in the format expected by chat completion APIs
type ChatTool struct {
	Type     string           `json:"type"` // Always "function" for function tools
	Function ChatToolFunction `json:"function"`
}

// ChatToolFunction represents the function definition for chat APIs
type ChatToolFunction struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  *jsonschema.Schema `json:"parameters"` // JSON Schema for parameters
}

// NOTE: ToChatTool and ToChatTools functions have been moved to the agent package
// since they depend on the Tool interface which is now in that package