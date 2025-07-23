package goferagent

import (
	"strings"
	"testing"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
	jsonschema "github.com/swaggest/jsonschema-go"
)

func TestFormatSchemaForPrompt(t *testing.T) {
	tests := []struct {
		name     string
		schema   *jsonschema.Schema
		expected []string // Lines that should appear in output
	}{
		{
			name: "simple string schema",
			schema: &jsonschema.Schema{
				Type:        &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("string"))},
				Description: ptr("A simple string field"),
			},
			expected: []string{
				"# A simple string field",
				"string",
			},
		},
		{
			name: "object with properties",
			schema: &jsonschema.Schema{
				Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("object"))},
				Properties: map[string]jsonschema.SchemaOrBool{
					"name": {
						TypeObject: &jsonschema.Schema{
							Type:        &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("string"))},
							Description: ptr("The name"),
						},
					},
					"age": {
						TypeObject: &jsonschema.Schema{
							Type:        &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("integer"))},
							Description: ptr("The age"),
						},
					},
				},
				Required: []string{"name"},
			},
			expected: []string{
				"object (required: name)",
				"name: string # The name",
				"age: integer # The age",
			},
		},
		{
			name: "array with items",
			schema: &jsonschema.Schema{
				Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("array"))},
				Items: &jsonschema.Items{
					SchemaOrBool: &jsonschema.SchemaOrBool{
						TypeObject: &jsonschema.Schema{
							Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("string"))},
						},
					},
				},
			},
			expected: []string{
				"array",
				"items: string",
			},
		},
		{
			name: "enum field",
			schema: &jsonschema.Schema{
				Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("string"))},
				Enum: []interface{}{"pending", "in_progress", "completed"},
			},
			expected: []string{
				"string (enum: \"pending\" | \"in_progress\" | \"completed\")",
			},
		},
		{
			name: "object with property without description",
			schema: &jsonschema.Schema{
				Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("object"))},
				Properties: map[string]jsonschema.SchemaOrBool{
					"id": {
						TypeObject: &jsonschema.Schema{
							Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("string"))},
						},
					},
					"count": {
						TypeObject: &jsonschema.Schema{
							Type:        &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("integer"))},
							Description: ptr("The count value"),
						},
					},
				},
			},
			expected: []string{
				"object",
				"id: string",
				"count: integer # The count value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSchemaForPrompt(tt.schema, 0)
			
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestFormatToolsForPrompt(t *testing.T) {
	// Create a test toolbox
	toolbox := agent.NewToolbox[agent.Tool]()
	
	// Add a test tool
	tool := &agent.LegacyTool{
		Type: "function",
		Function: aisdk.ToolFunction{
			Name:        "TestTool",
			Description: "A test tool for testing",
			Parameters: &jsonschema.Schema{
				Type: &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("object"))},
				Properties: map[string]jsonschema.SchemaOrBool{
					"input": {
						TypeObject: &jsonschema.Schema{
							Type:        &jsonschema.Type{SimpleTypes: ptr(jsonschema.SimpleType("string"))},
							Description: ptr("The input string"),
						},
					},
				},
				Required: []string{"input"},
			},
		},
	}
	
	err := toolbox.RegisterTool(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	result := formatToolsForPrompt(toolbox)
	
	expectedLines := []string{
		"You have access to the following tools:",
		"Tool: TestTool",
		"Description: A test tool for testing",
		"Input Schema:",
		"object (required: input)",
		"input: string # The input string",
	}
	
	for _, expected := range expectedLines {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected output to contain %q, but got:\n%s", expected, result)
		}
	}
}

func TestGenerateSystemPrompt(t *testing.T) {
	// Create a test toolbox
	toolbox := agent.NewToolbox[agent.Tool]()
	
	// Generate prompt
	result := GenerateSystemPrompt(toolbox)
	
	// Check that major sections are present
	expectedSections := []string{
		"You are Gofer, a CLI tool for using LLMs",
		"# Tone and style",
		"# Following conventions", 
		"# Tool usage policy",
		"Working directory:",
		"No tools available.",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("Expected system prompt to contain %q", section)
		}
	}
}

// Helper function to create pointers
func ptr[T any](v T) *T {
	return &v
}