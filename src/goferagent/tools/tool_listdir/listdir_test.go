package tool_listdir

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

func TestListDirectoryTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkResult   func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "list empty directory",
			setupFS: func(fs afero.Fs) error {
				return fs.MkdirAll("/empty", 0755)
			},
			args: map[string]interface{}{
				"path": "/empty",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/empty", result["path"])
				assert.Equal(t, float64(0), result["count"])
				if result["files"] != nil {
					files := result["files"].([]interface{})
					assert.Empty(t, files)
				}
			},
		},
		{
			name: "list directory with files",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/test", 0755); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/test/file1.txt", []byte("content1"), 0644); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/test/file2.py", []byte("content2"), 0644); err != nil {
					return err
				}
				if err := fs.MkdirAll("/test/subdir", 0755); err != nil {
					return err
				}
				return nil
			},
			args: map[string]interface{}{
				"path": "/test",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/test", result["path"])
				assert.Equal(t, float64(3), result["count"])
				
				files := result["files"].([]interface{})
				assert.Len(t, files, 3)
				
				// Check file properties
				fileNames := make(map[string]interface{})
				for _, f := range files {
					file := f.(map[string]interface{})
					fileNames[file["name"].(string)] = file
				}
				
				// Check file1.txt
				file1 := fileNames["file1.txt"].(map[string]interface{})
				assert.Equal(t, "file1.txt", file1["name"])
				assert.Equal(t, "/test/file1.txt", file1["path"])
				assert.Equal(t, false, file1["is_dir"])
				assert.Equal(t, float64(8), file1["size"]) // "content1" = 8 bytes
				assert.Equal(t, "text", file1["language"])
				
				// Check file2.py
				file2 := fileNames["file2.py"].(map[string]interface{})
				assert.Equal(t, "python", file2["language"])
				
				// Check subdir
				subdir := fileNames["subdir"].(map[string]interface{})
				assert.Equal(t, true, subdir["is_dir"])
			},
		},
		{
			name: "list directory recursively",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/testdir/sub1/sub2", 0755); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/testdir/file1.txt", []byte("root file"), 0644); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/testdir/sub1/file2.txt", []byte("sub1 file"), 0644); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/testdir/sub1/sub2/file3.txt", []byte("sub2 file"), 0644); err != nil {
					return err
				}
				return nil
			},
			args: map[string]interface{}{
				"path":      "/testdir",
				"recursive": true,
			},
			expectedError: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				files := result["files"].([]interface{})
				// Should include: /testdir, /testdir/file1.txt, /testdir/sub1, /testdir/sub1/file2.txt, 
				// /testdir/sub1/sub2, /testdir/sub1/sub2/file3.txt
				assert.Equal(t, float64(6), result["count"])
				
				// Collect all paths
				paths := make(map[string]bool)
				for _, f := range files {
					file := f.(map[string]interface{})
					paths[file["path"].(string)] = true
				}
				
				assert.True(t, paths["/testdir"])
				assert.True(t, paths["/testdir/file1.txt"])
				assert.True(t, paths["/testdir/sub1"])
				assert.True(t, paths["/testdir/sub1/file2.txt"])
				assert.True(t, paths["/testdir/sub1/sub2"])
				assert.True(t, paths["/testdir/sub1/sub2/file3.txt"])
			},
		},
		{
			name: "list non-existent directory",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "/nonexistent",
			},
			expectedError: true,
		},
		{
			name: "list with unsafe path",
			setupFS: nil,
			args: map[string]interface{}{
				"path": "../../../etc",
			},
			expectedError: true,
		},
		{
			name: "list file instead of directory",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/file.txt", []byte("not a directory"), 0644)
			},
			args: map[string]interface{}{
				"path": "/file.txt",
			},
			expectedError: true,
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
				assert.False(t, response.IsError, "Response error: %s", string(response.Content))
				
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

func TestListDirectoryToolFileInfo(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	// Create a file with known timestamp
	err := afero.WriteFile(fs, "/test.txt", []byte("test content"), 0644)
	require.NoError(t, err)
	
	// Note: afero.MemMapFs doesn't support changing mod times, so we'll just check format
	
	tool, err := Tool(fs)
	require.NoError(t, err)
	
	args := map[string]interface{}{
		"path": "/",
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
	
	var result map[string]interface{}
	err = json.Unmarshal(response.Content, &result)
	require.NoError(t, err)
	
	files := result["files"].([]interface{})
	assert.Len(t, files, 1)
	
	file := files[0].(map[string]interface{})
	assert.Equal(t, "test.txt", file["name"])
	assert.Equal(t, float64(12), file["size"]) // "test content" = 12 bytes
	
	// Check that mod_time is in RFC3339 format
	modTime := file["mod_time"].(string)
	_, err = time.Parse(time.RFC3339, modTime)
	assert.NoError(t, err)
}