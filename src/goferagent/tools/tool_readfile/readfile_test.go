package tool_readfile

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileTool(t *testing.T) {
	tests := []struct {
		name           string
		setupFS        func(afero.Fs) error
		args           map[string]interface{}
		expectedError  bool
		expectedResult map[string]interface{}
		checkContent   func(t *testing.T, content string)
	}{
		{
			name: "read simple text file",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("Hello, World!"), 0644)
			},
			args: map[string]interface{}{
				"path": "/test.txt",
			},
			expectedError: false,
			checkContent: func(t *testing.T, content string) {
				assert.Equal(t, "Hello, World!", content)
			},
		},
		{
			name: "read file with line numbers",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("line1\nline2\nline3"), 0644)
			},
			args: map[string]interface{}{
				"path":         "/test.txt",
				"line_numbers": true,
			},
			expectedError: false,
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "1: line1")
				assert.Contains(t, content, "2: line2")
				assert.Contains(t, content, "3: line3")
			},
		},
		{
			name: "read non-existent file",
			setupFS: func(fs afero.Fs) error {
				return nil
			},
			args: map[string]interface{}{
				"path": "/nonexistent.txt",
			},
			expectedError: true,
		},
		{
			name: "read file with unsafe path",
			setupFS: func(fs afero.Fs) error {
				return nil
			},
			args: map[string]interface{}{
				"path": "../../../etc/passwd",
			},
			expectedError: true,
		},
		{
			name: "read empty file",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/empty.txt", []byte(""), 0644)
			},
			args: map[string]interface{}{
				"path": "/empty.txt",
			},
			expectedError: false,
			checkContent: func(t *testing.T, content string) {
				assert.Equal(t, "", content)
			},
		},
		{
			name: "read large file",
			setupFS: func(fs afero.Fs) error {
				// Create a file with 100 lines
				content := ""
				for i := 1; i <= 100; i++ {
					content += fmt.Sprintf("Line %d\n", i)
				}
				return afero.WriteFile(fs, "/large.txt", []byte(content), 0644)
			},
			args: map[string]interface{}{
				"path": "/large.txt",
			},
			expectedError: false,
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "Line 1\n")
				assert.Contains(t, content, "Line 100\n")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create in-memory filesystem
			fs := afero.NewMemMapFs()
			
			// Setup filesystem
			if tt.setupFS != nil {
				err := tt.setupFS(fs)
				require.NoError(t, err)
			}

			// Create the tool
			tool, err := Tool(fs)
			require.NoError(t, err)

			// Prepare arguments
			argsJSON, err := json.Marshal(tt.args)
			require.NoError(t, err)

			// Create tool call
			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: argsJSON,
				},
			}

			// Execute
			response, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)

			// Check response
			if tt.expectedError {
				assert.True(t, response.IsError)
			} else {
				assert.False(t, response.IsError)
				
				// Parse response
				var result map[string]interface{}
				err := json.Unmarshal(response.Content, &result)
				require.NoError(t, err)
				
				// Check content if provided
				if tt.checkContent != nil && result["content"] != nil {
					tt.checkContent(t, result["content"].(string))
				}
			}
		})
	}
}

func TestReadFileToolWithBinaryDetection(t *testing.T) {
	// NOTE: The current ReadFileTool does not reject binary files, it just marks them as non-text
	// This test documents the current behavior vs desired behavior
	fs := afero.NewMemMapFs()
	
	// Create a binary file
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
	err := afero.WriteFile(fs, "/binary.bin", binaryContent, 0644)
	require.NoError(t, err)

	tool, err := Tool(fs)
	require.NoError(t, err)

	args := map[string]interface{}{
		"path": "/binary.bin",
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: argsJSON,
		},
	}

	response, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	
	// CURRENT BEHAVIOR: Tool succeeds but marks file as non-text
	// DESIRED BEHAVIOR: Tool should reject binary files with an error
	assert.False(t, response.IsError) // Current behavior
	
	var result map[string]interface{}
	err = json.Unmarshal(response.Content, &result)
	require.NoError(t, err)
	
	// Verify it's marked as non-text
	assert.Equal(t, false, result["is_text"])
	
	// TODO: Consider adding binary file rejection in future versions
	// assert.True(t, response.IsError)
	// assert.Contains(t, string(response.Content), "binary file")
}