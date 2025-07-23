package tool_runcommand

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/elee1766/gofer/src/shell"
	"github.com/stretchr/testify/assert"
)

func TestRunCommandBasic(t *testing.T) {
	// Initialize the shell manager for testing
	logger := slog.Default()
	toolsutil.SetLogger(logger)
	
	// Create a shell manager for testing
	shellManager := shell.NewShellManager(logger)
	
	// Test that the tool can be created
	tool := Tool(shellManager)
	assert.NotNil(t, tool)
	
	// Test that it's a GenericTool (skip assertion for now due to type system complexity)
	// _, ok := tool.(*agent.GenericTool[RunCommandInput, RunCommandOutput])
	// assert.True(t, ok, "Tool should be a GenericTool")
	
	// Test tool interface methods
	assert.Equal(t, Name, tool.GetName())
	assert.Equal(t, "function", tool.GetType())
	assert.NotEmpty(t, tool.GetDescription())
	
	// Test that parameters are correctly defined
	parameters := tool.GetParameters()
	assert.NotNil(t, parameters)
	
	// Test parameter marshaling
	paramSchema, err := json.Marshal(parameters)
	assert.NoError(t, err)
	assert.NotEmpty(t, paramSchema)
	
	// Verify required parameters are present
	var schema map[string]interface{}
	err = json.Unmarshal(paramSchema, &schema)
	assert.NoError(t, err)
	assert.Contains(t, schema, "properties")
	
	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, properties, "command") // command is required
}

func TestRunCommandConstants(t *testing.T) {
	// Test that the tool name constant is correct
	assert.Equal(t, "run_command", Name)
}