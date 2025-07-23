package tool_deletefile

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteFileTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkFS       func(t *testing.T, fs afero.Fs)
		checkResult   func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "delete single file",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/test.txt", []byte("content"), 0644)
			},
			args: map[string]interface{}{
				"path": "/test.txt",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.Exists(fs, "/test.txt")
				require.NoError(t, err)
				assert.False(t, exists)
			},
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/test.txt", result["path"])
				assert.Equal(t, true, result["deleted"])
				assert.Equal(t, false, result["was_directory"])
			},
		},
		{
			name: "delete empty directory",
			setupFS: func(fs afero.Fs) error {
				return fs.MkdirAll("/emptydir", 0755)
			},
			args: map[string]interface{}{
				"path": "/emptydir",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.DirExists(fs, "/emptydir")
				require.NoError(t, err)
				assert.False(t, exists)
			},
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/emptydir", result["path"])
				assert.Equal(t, true, result["deleted"])
				assert.Equal(t, true, result["was_directory"])
			},
		},
		{
			name: "delete directory with contents recursively",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/dir/subdir", 0755); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/dir/file1.txt", []byte("content1"), 0644); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/dir/subdir/file2.txt", []byte("content2"), 0644); err != nil {
					return err
				}
				return nil
			},
			args: map[string]interface{}{
				"path":      "/dir",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Check that entire directory tree is gone
				exists, err := afero.DirExists(fs, "/dir")
				require.NoError(t, err)
				assert.False(t, exists)
				
				// Verify subdirectories and files are also gone
				exists, err = afero.Exists(fs, "/dir/file1.txt")
				require.NoError(t, err)
				assert.False(t, exists)
				
				exists, err = afero.Exists(fs, "/dir/subdir/file2.txt")
				require.NoError(t, err)
				assert.False(t, exists)
			},
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, true, result["was_directory"])
			},
		},
		{
			name: "delete non-empty directory succeeds (uses RemoveAll)",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/dir", 0755); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/dir/file.txt", []byte("content"), 0644)
			},
			args: map[string]interface{}{
				"path": "/dir",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.DirExists(fs, "/dir")
				require.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "delete non-existent file",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "/nonexistent.txt",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/nonexistent.txt", result["path"])
				assert.Equal(t, false, result["deleted"])
				assert.Equal(t, "file does not exist", result["reason"])
			},
		},
		{
			name: "delete with unsafe path",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "../../../etc/passwd",
			},
			expectedError: true,
		},
		{
			name: "delete file with special characters",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/file with spaces.txt", []byte("content"), 0644)
			},
			args: map[string]interface{}{
				"path": "/file with spaces.txt",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.Exists(fs, "/file with spaces.txt")
				require.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "delete symlink",
			setupFS: func(fs afero.Fs) error {
				// Note: afero.MemMapFs doesn't support symlinks,
				// but we'll test the logic anyway
				return afero.WriteFile(fs, "/link", []byte("link content"), 0644)
			},
			args: map[string]interface{}{
				"path": "/link",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.Exists(fs, "/link")
				require.NoError(t, err)
				assert.False(t, exists)
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

func TestDeleteFileToolComplexDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	// Create complex directory structure
	structure := []string{
		"/testroot/file1.txt",
		"/testroot/file2.txt",
		"/testroot/sub1/file3.txt",
		"/testroot/sub1/file4.txt",
		"/testroot/sub1/sub2/file5.txt",
		"/testroot/sub3/file6.txt",
	}
	
	// Create all files
	for _, path := range structure {
		dir := path[:len(path)-len("/fileX.txt")]
		err := fs.MkdirAll(dir, 0755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, path, []byte("content"), 0644)
		require.NoError(t, err)
	}
	
	// Verify structure exists
	for _, path := range structure {
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists)
	}
	
	// Delete root recursively
	tool, err := Tool(fs)
	require.NoError(t, err)
	
	args := map[string]interface{}{
		"path": "/testroot", // Changed from "/root" which is considered unsafe
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
	
	if response.IsError {
		t.Fatalf("Delete failed: %s", string(response.Content))
	}
	
	// Verify everything is gone
	for _, path := range structure {
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		assert.False(t, exists, "Path %s should not exist", path)
	}
	
	// Verify root directory is gone
	exists, err := afero.DirExists(fs, "/testroot")
	require.NoError(t, err)
	assert.False(t, exists)
}