package tool_getfileinfo

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileInfoTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkResult   func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "get info for regular file",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("Hello, World!"), 0644)
			},
			args: map[string]interface{}{
				"path": "/test.txt",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/test.txt", result["path"])
				assert.Equal(t, "test.txt", result["name"])
				assert.Equal(t, false, result["is_dir"])
				assert.Equal(t, float64(13), result["size"]) // "Hello, World!" = 13 bytes
				assert.Equal(t, "text", result["language"])
				assert.NotNil(t, result["mod_time"])
			},
		},
		{
			name: "get info for directory",
			setupFS: func(fs afero.Fs) error {
				return fs.MkdirAll("/testdir", 0755)
			},
			args: map[string]interface{}{
				"path": "/testdir",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/testdir", result["path"])
				assert.Equal(t, "testdir", result["name"])
				assert.Equal(t, true, result["is_dir"])
				assert.NotNil(t, result["mod_time"])
			},
		},
		{
			name: "get info for directory with contents",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/dir", 0755); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/dir/file1.txt", []byte("content1"), 0644); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/dir/file2.py", []byte("content2"), 0644); err != nil {
					return err
				}
				if err := fs.MkdirAll("/dir/subdir", 0755); err != nil {
					return err
				}
				return nil
			},
			args: map[string]interface{}{
				"path": "/dir",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/dir", result["path"])
				assert.Equal(t, true, result["is_dir"])
				
				// Check directory stats
				assert.Equal(t, float64(3), result["entry_count"])
				assert.Equal(t, float64(2), result["file_count"])
				assert.Equal(t, float64(1), result["dir_count"])
			},
		},
		{
			name: "get info for non-existent file",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "/nonexistent.txt",
			},
			expectedError: true,
		},
		{
			name: "get info with unsafe path",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "../../../etc/passwd",
			},
			expectedError: true,
		},
		{
			name: "get info for python file",
			setupFS: func(fs afero.Fs) error {
				content := `#!/usr/bin/env python3
def main():
    print("Hello, World!")

if __name__ == "__main__":
    main()
`
				return afero.WriteFile(fs, "/script.py", []byte(content), 0755)
			},
			args: map[string]interface{}{
				"path": "/script.py",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/script.py", result["path"])
				assert.Equal(t, "python", result["language"])
				assert.Greater(t, result["size"].(float64), float64(0))
			},
		},
		{
			name: "get info for empty directory",
			setupFS: func(fs afero.Fs) error {
				return fs.MkdirAll("/empty", 0755)
			},
			args: map[string]interface{}{
				"path": "/empty",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/empty", result["path"])
				assert.Equal(t, true, result["is_dir"])
				
				// Check directory stats
				assert.Equal(t, float64(0), result["entry_count"])
				assert.Equal(t, float64(0), result["file_count"])
				assert.Equal(t, float64(0), result["dir_count"])
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
				
				// Check result
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestGetFileInfoToolTimeFormat(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	// Create a file
	err := afero.WriteFile(fs, "/test.txt", []byte("test"), 0644)
	require.NoError(t, err)
	
	tool, err := Tool(fs)
	require.NoError(t, err)
	
	args := map[string]interface{}{
		"path": "/test.txt",
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
	
	assert.False(t, response.IsError)
	
	var result map[string]interface{}
	err = json.Unmarshal(response.Content, &result)
	require.NoError(t, err)
	
	// Check that modified time is in RFC3339 format
	modTimeStr := result["mod_time"].(string)
	_, err = time.Parse(time.RFC3339, modTimeStr)
	assert.NoError(t, err)
}