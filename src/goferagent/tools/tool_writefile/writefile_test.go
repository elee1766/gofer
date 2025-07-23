package tool_writefile

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkFS       func(t *testing.T, fs afero.Fs)
	}{
		{
			name:    "write new file",
			setupFS: nil,
			args: map[string]interface{}{
				"path":    "/test.txt",
				"content": "Hello, World!",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/test.txt")
				require.NoError(t, err)
				assert.Equal(t, "Hello, World!", string(content))
			},
		},
		{
			name: "overwrite existing file",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("Old content"), 0644)
			},
			args: map[string]interface{}{
				"path":    "/test.txt",
				"content": "New content",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/test.txt")
				require.NoError(t, err)
				assert.Equal(t, "New content", string(content))
			},
		},
		{
			name: "write file with create_dirs",
			setupFS: nil,
			args: map[string]interface{}{
				"path":        "/deep/nested/dir/file.txt",
				"content":     "Nested file",
				"create_dirs": true,
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Check that directories were created
				exists, err := afero.DirExists(fs, "/deep/nested/dir")
				require.NoError(t, err)
				assert.True(t, exists)
				
				// Check file content
				content, err := afero.ReadFile(fs, "/deep/nested/dir/file.txt")
				require.NoError(t, err)
				assert.Equal(t, "Nested file", string(content))
			},
		},
		{
			name: "write file without create_dirs should fail", // Now controlled by create_dirs parameter
			setupFS: nil,
			args: map[string]interface{}{
				"path":        "/nonexistent/dir/file.txt",
				"content":     "Should fail",
				"create_dirs": false, // Test the new create_dirs parameter
			},
			expectedError: true,
		},
		{
			name: "write file with unsafe path",
			setupFS: nil,
			args: map[string]interface{}{
				"path":    "../../../etc/passwd",
				"content": "Malicious content",
			},
			expectedError: true,
		},
		{
			name: "write empty file",
			setupFS: nil,
			args: map[string]interface{}{
				"path":    "/empty.txt",
				"content": "",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/empty.txt")
				require.NoError(t, err)
				assert.Equal(t, "", string(content))
			},
		},
		{
			name: "write multiline content",
			setupFS: nil,
			args: map[string]interface{}{
				"path":    "/multiline.txt",
				"content": "Line 1\nLine 2\nLine 3\n",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/multiline.txt")
				require.NoError(t, err)
				assert.Equal(t, "Line 1\nLine 2\nLine 3\n", string(content))
			},
		},
		{
			name: "write with custom mode",
			setupFS: nil,
			args: map[string]interface{}{
				"path":    "/executable.sh",
				"content": "#!/bin/bash\necho 'Hello'",
				"mode":    float64(0755), // JSON numbers are float64
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				info, err := fs.Stat("/executable.sh")
				require.NoError(t, err)
				// Check if executable bit is set
				assert.Equal(t, os.FileMode(0755), info.Mode())
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
			// Execute directly
			response, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)

			// Check response
			if tt.expectedError {
				if !response.IsError {
					t.Logf("Expected error but got success. Response: %s", string(response.Content))
				}
				assert.True(t, response.IsError)
			} else {
				assert.False(t, response.IsError)
				
				// Check filesystem state
				if tt.checkFS != nil {
					tt.checkFS(t, fs)
				}
			}
		})
	}
}

func TestWriteFileToolLargeContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	tool, err := Tool(fs)
	require.NoError(t, err)

	// Create large content (just under the size limit)
	largeContent := make([]byte, 10*1024*1024-1) // Just under 10MB
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}

	args := map[string]interface{}{
		"path":    "/large.txt",
		"content": string(largeContent),
	}
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: argsJSON,
		},
	}

	// Execute directly
	response, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	
	assert.False(t, response.IsError)
	
	// Verify file was written
	content, err := afero.ReadFile(fs, "/large.txt")
	require.NoError(t, err)
	assert.Equal(t, largeContent, content)
}