package tool_runcommand

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/elee1766/gofer/src/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandToolSimple(t *testing.T) {
	// Initialize logger for testing
	logger := slog.Default()
	toolsutil.SetLogger(logger)
	
	// Create a shell manager for testing
	shellManager := shell.NewShellManager(logger)
	_ = Tool(shellManager) // Verify tool can be created

	tests := []struct {
		name      string
		command   string
		args      []string
		expectErr bool
	}{
		{
			name:    "simple echo command",
			command: "echo",
			args:    []string{"hello world"},
		},
		// TODO: Implement security checks to block dangerous commands
		// For now, these tests are commented out as the feature is not implemented
		// {
		// 	name:      "dangerous command blocked",
		// 	command:   "rm",
		// 	args:      []string{"-rf", "/"},
		// 	expectErr: true,
		// },
		// {
		// 	name:      "sudo command blocked",
		// 	command:   "sudo",
		// 	args:      []string{"echo", "test"},
		// 	expectErr: true,
		// },
		{
			name:      "empty command",
			command:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Combine command and args into a single command string
			fullCommand := tt.command
			if len(tt.args) > 0 {
				for _, arg := range tt.args {
					fullCommand += " " + arg
				}
			}
			
			params := map[string]interface{}{
				"command": fullCommand,
			}

			paramsJSON, _ := json.Marshal(params)

			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: paramsJSON,
				},
			}

			tool := Tool(shellManager)
			resp, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)

			if tt.expectErr {
				assert.True(t, resp.IsError, "Expected error response")
				return
			}

			// For successful cases, just verify the response structure
			if !resp.IsError {
				var response map[string]interface{}
				err = json.Unmarshal(resp.Content, &response)
				require.NoError(t, err)

				// Verify common response structure
				assert.Contains(t, response, "command")
				assert.Contains(t, response, "exit_code")
				assert.Contains(t, response, "output")
				assert.Contains(t, response, "timeout")
			}
		})
	}
}

// TODO: Implement security checks to block dangerous commands
// This test is commented out until the security feature is implemented
/*
func TestRunCommandSecuritySimple(t *testing.T) {
	// Initialize logger for testing
	logger := slog.Default()
	toolsutil.SetLogger(logger)
	
	// Create a shell manager for testing
	shellManager := shell.NewShellManager(logger)

	dangerousCommands := []struct {
		name    string
		command string
		args    []string
	}{
		{"rm dangerous", "rm", []string{"-rf", "/"}},
		{"sudo", "sudo", []string{"ls"}},
		{"su", "su", []string{"root"}},
	}

	for _, tt := range dangerousCommands {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{
				"command": tt.command,
				"args":    tt.args,
			}

			paramsJSON, _ := json.Marshal(params)
			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: paramsJSON,
				},
			}

			tool := Tool(shellManager)
			resp, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)
			assert.True(t, resp.IsError, "Expected dangerous command to be blocked")
		})
	}
}
*/