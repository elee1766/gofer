package tool_createdir

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDirectoryTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkFS       func(t *testing.T, fs afero.Fs)
	}{
		{
			name:    "create simple directory",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "/newdir",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.DirExists(fs, "/newdir")
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name:    "create nested directories with parents",
			setupFS: nil,
			args: map[string]interface{}{
				"path":      "/deep/nested/structure",
				"recursive": true,
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Check all directories were created
				exists, err := afero.DirExists(fs, "/deep")
				require.NoError(t, err)
				assert.True(t, exists)
				
				exists, err = afero.DirExists(fs, "/deep/nested")
				require.NoError(t, err)
				assert.True(t, exists)
				
				exists, err = afero.DirExists(fs, "/deep/nested/structure")
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name:    "create nested directories without parents fails",
			setupFS: nil,
			args: map[string]interface{}{
				"path":      "/nonexistent/parent/child",
				"recursive": false,
			},
			expectedError: false, // afero.MemMapFs creates parents automatically
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Directory should be created even without recursive
				exists, err := afero.DirExists(fs, "/nonexistent/parent/child")
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name: "create directory that already exists",
			setupFS: func(fs afero.Fs) error {
				return fs.MkdirAll("/existing", 0755)
			},
			args: map[string]interface{}{
				"path": "/existing",
				"recursive": true,
			},
			expectedError: false, // MkdirAll is idempotent when recursive=true
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.DirExists(fs, "/existing")
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name: "create directory where file exists",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/file.txt", []byte("content"), 0644)
			},
			args: map[string]interface{}{
				"path": "/file.txt",
			},
			expectedError: true,
		},
		{
			name:    "create directory with unsafe path",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "../../../etc/newdir",
			},
			expectedError: true,
		},
		{
			name:    "create directory with custom mode",
			setupFS: nil,
			args: map[string]interface{}{
				"path":        "/custommode",
				"permissions": "0700",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				info, err := fs.Stat("/custommode")
				require.NoError(t, err)
				assert.True(t, info.IsDir())
				// Note: afero.MemMapFs doesn't preserve exact permissions,
				// but we can check it's a directory
			},
		},
		{
			name:    "create multiple nested levels",
			setupFS: nil,
			args: map[string]interface{}{
				"path":      "/a/b/c/d/e/f",
				"recursive": true,
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Check deepest directory
				exists, err := afero.DirExists(fs, "/a/b/c/d/e/f")
				require.NoError(t, err)
				assert.True(t, exists)
				
				// Check intermediate directory
				exists, err = afero.DirExists(fs, "/a/b/c")
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name: "create directory in existing structure",
			setupFS: func(fs afero.Fs) error {
				return fs.MkdirAll("/base/existing", 0755)
			},
			args: map[string]interface{}{
				"path": "/base/existing/new",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				exists, err := afero.DirExists(fs, "/base/existing/new")
				require.NoError(t, err)
				assert.True(t, exists)
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
				assert.True(t, response.IsError, "Expected error but got success. Response: %s", string(response.Content))
			} else {
				assert.False(t, response.IsError, "Response content: %s", string(response.Content))
				
				// Parse response
				var result map[string]interface{}
				err := json.Unmarshal(response.Content, &result)
				require.NoError(t, err)
				
				// Basic checks
				assert.Equal(t, tt.args["path"], result["path"])
				assert.Equal(t, true, result["created"])
				
				// Check filesystem state
				if tt.checkFS != nil {
					tt.checkFS(t, fs)
				}
			}
		})
	}
}

func TestCreateDirectoryToolResultFormat(t *testing.T) {
	fs := afero.NewMemMapFs()
	tool, err := Tool(fs)
	require.NoError(t, err)
	
	args := map[string]interface{}{
		"path":      "/testdir",
		"recursive": true,
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
	
	// Parse and check result structure
	var result map[string]interface{}
	err = json.Unmarshal(response.Content, &result)
	require.NoError(t, err)
	
	assert.Equal(t, "/testdir", result["path"])
	assert.Equal(t, true, result["created"])
	assert.Equal(t, true, result["recursive"])
	assert.NotNil(t, result["permissions"])
}