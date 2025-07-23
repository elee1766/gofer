package tool_editfile

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditFileTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkFS       func(t *testing.T, fs afero.Fs)
		checkResult   func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "simple edit",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("Hello, World!"), 0644)
			},
			args: map[string]interface{}{
				"path":        "/test.txt",
				"old_content": "World",
				"new_content": "Universe",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/test.txt")
				require.NoError(t, err)
				assert.Equal(t, "Hello, Universe!", string(content))
			},
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/test.txt", result["path"])
				assert.Equal(t, float64(13), result["old_size"]) // "Hello, World!" = 13
				assert.Equal(t, float64(16), result["new_size"]) // "Hello, Universe!" = 16
				assert.Equal(t, true, result["changes_made"])
				assert.Equal(t, false, result["backup_created"])
			},
		},
		{
			name: "edit with backup",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("Original content"), 0644)
			},
			args: map[string]interface{}{
				"path":          "/test.txt",
				"old_content":   "Original",
				"new_content":   "Modified",
				"create_backup": true,
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Check edited file
				content, err := afero.ReadFile(fs, "/test.txt")
				require.NoError(t, err)
				assert.Equal(t, "Modified content", string(content))
				
				// Check that backup exists
				files, err := afero.ReadDir(fs, "/")
				require.NoError(t, err)
				
				backupFound := false
				for _, file := range files {
					if strings.HasPrefix(file.Name(), "test.txt.backup.") {
						backupFound = true
						// Check backup content
						backupContent, err := afero.ReadFile(fs, "/"+file.Name())
						require.NoError(t, err)
						assert.Equal(t, "Original content", string(backupContent))
						break
					}
				}
				assert.True(t, backupFound, "Backup file not found")
			},
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, true, result["backup_created"])
			},
		},
		{
			name: "edit non-existent file",
			setupFS: nil,
			args: map[string]interface{}{
				"path":        "/nonexistent.txt",
				"old_content": "foo",
				"new_content": "bar",
			},
			expectedError: true,
		},
		{
			name: "edit with content not found",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("Hello, World!"), 0644)
			},
			args: map[string]interface{}{
				"path":        "/test.txt",
				"old_content": "Goodbye",
				"new_content": "Hello",
			},
			expectedError: true,
		},
		{
			name: "edit with unsafe path",
			setupFS: nil,
			args: map[string]interface{}{
				"path":        "../../../etc/passwd",
				"old_content": "root",
				"new_content": "hacked",
			},
			expectedError: true,
		},
		{
			name: "edit multiline content",
			setupFS: func(fs afero.Fs) error {
				content := `Line 1
Line 2
Line 3
Line 4`
				return afero.WriteFile(fs, "/multiline.txt", []byte(content), 0644)
			},
			args: map[string]interface{}{
				"path":        "/multiline.txt",
				"old_content": "Line 2\nLine 3",
				"new_content": "Modified Line 2\nModified Line 3",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/multiline.txt")
				require.NoError(t, err)
				expected := `Line 1
Modified Line 2
Modified Line 3
Line 4`
				assert.Equal(t, expected, string(content))
			},
		},
		{
			name: "edit preserving indentation",
			setupFS: func(fs afero.Fs) error {
				content := `function test() {
    if (true) {
        console.log("Hello");
    }
}`
				return afero.WriteFile(fs, "/code.js", []byte(content), 0644)
			},
			args: map[string]interface{}{
				"path":        "/code.js",
				"old_content": `        console.log("Hello");`,
				"new_content": `        console.log("Hello, World!");`,
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				content, err := afero.ReadFile(fs, "/code.js")
				require.NoError(t, err)
				expected := `function test() {
    if (true) {
        console.log("Hello, World!");
    }
}`
				assert.Equal(t, expected, string(content))
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
				
				// Check filesystem state
				if tt.checkFS != nil {
					tt.checkFS(t, fs)
				}
			}
		})
	}
}

func TestEditFileToolBackupNaming(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	// Create initial file
	err := afero.WriteFile(fs, "/test.txt", []byte("Original"), 0644)
	require.NoError(t, err)
	
	tool, err := Tool(fs)
	require.NoError(t, err)
	
	// First edit with backup
	args1 := map[string]interface{}{
		"path":          "/test.txt",
		"old_content":   "Original",
		"new_content":   "First edit",
		"create_backup": true,
	}
	argsJSON1, err := json.Marshal(args1)
	require.NoError(t, err)
	
	call1 := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: argsJSON1,
		},
	}
	
	response1, err := tool.Execute(context.Background(), call1)
	require.NoError(t, err)
	assert.False(t, response1.IsError)
	
	// Wait a moment to ensure different timestamp
	time.Sleep(time.Millisecond * 10)
	
	// Second edit with backup
	args2 := map[string]interface{}{
		"path":          "/test.txt",
		"old_content":   "First edit",
		"new_content":   "Second edit",
		"create_backup": true,
	}
	argsJSON2, err := json.Marshal(args2)
	require.NoError(t, err)
	
	call2 := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: argsJSON2,
		},
	}
	
	response2, err := tool.Execute(context.Background(), call2)
	require.NoError(t, err)
	assert.False(t, response2.IsError)
	
	// Check that we have at least 2 files (original + at least 1 backup)
	files, err := afero.ReadDir(fs, "/")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 2)
	
	// Count backup files
	backupCount := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "test.txt.backup.") {
			backupCount++
		}
	}
	assert.GreaterOrEqual(t, backupCount, 1)
}