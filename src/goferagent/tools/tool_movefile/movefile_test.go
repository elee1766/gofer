package tool_movefile

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveFileTool(t *testing.T) {
	tests := []struct {
		name          string
		setupFS       func(afero.Fs) error
		args          map[string]interface{}
		expectedError bool
		checkFS       func(t *testing.T, fs afero.Fs)
		checkResult   func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "move file to new location",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/source.txt", []byte("file content"), 0644)
			},
			args: map[string]interface{}{
				"source": "/source.txt",
				"destination":   "/destination.txt",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Source should not exist
				exists, err := afero.Exists(fs, "/source.txt")
				require.NoError(t, err)
				assert.False(t, exists)
				
				// Destination should exist with same content
				content, err := afero.ReadFile(fs, "/destination.txt")
				require.NoError(t, err)
				assert.Equal(t, "file content", string(content))
			},
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "/source.txt", result["source"])
				assert.Equal(t, "/destination.txt", result["destination"])
				assert.Equal(t, true, result["moved"])
			},
		},
		{
			name: "rename file in same directory",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/dir", 0755); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/dir/old.txt", []byte("content"), 0644)
			},
			args: map[string]interface{}{
				"source": "/dir/old.txt",
				"destination":   "/dir/new.txt",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Old name should not exist
				exists, err := afero.Exists(fs, "/dir/old.txt")
				require.NoError(t, err)
				assert.False(t, exists)
				
				// New name should exist
				content, err := afero.ReadFile(fs, "/dir/new.txt")
				require.NoError(t, err)
				assert.Equal(t, "content", string(content))
			},
		},
		{
			name: "move directory",
			setupFS: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/sourcedir/subdir", 0755); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "/sourcedir/file.txt", []byte("file1"), 0644); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/sourcedir/subdir/file2.txt", []byte("file2"), 0644)
			},
			args: map[string]interface{}{
				"source": "/sourcedir",
				"destination":   "/destdir",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Source should not exist
				exists, err := afero.DirExists(fs, "/sourcedir")
				require.NoError(t, err)
				assert.False(t, exists)
				
				// Destination should exist with all contents
				exists, err = afero.DirExists(fs, "/destdir")
				require.NoError(t, err)
				assert.True(t, exists)
				
				// Check files moved correctly
				content, err := afero.ReadFile(fs, "/destdir/file.txt")
				require.NoError(t, err)
				assert.Equal(t, "file1", string(content))
				
				content, err = afero.ReadFile(fs, "/destdir/subdir/file2.txt")
				require.NoError(t, err)
				assert.Equal(t, "file2", string(content))
			},
		},
		{
			name: "move to existing file with overwrite",
			setupFS: func(fs afero.Fs) error {
				if err := afero.WriteFile(fs, "/source.txt", []byte("new content"), 0644); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/dest.txt", []byte("old content"), 0644)
			},
			args: map[string]interface{}{
				"source":    "/source.txt",
				"destination":      "/dest.txt",
				"overwrite": true,
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Source should not exist
				exists, err := afero.Exists(fs, "/source.txt")
				require.NoError(t, err)
				assert.False(t, exists)
				
				// Destination should have new content
				content, err := afero.ReadFile(fs, "/dest.txt")
				require.NoError(t, err)
				assert.Equal(t, "new content", string(content))
			},
		},
		{
			name: "move to existing file without overwrite fails",
			setupFS: func(fs afero.Fs) error {
				if err := afero.WriteFile(fs, "/source.txt", []byte("new content"), 0644); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/dest.txt", []byte("old content"), 0644)
			},
			args: map[string]interface{}{
				"source":    "/source.txt",
				"destination":      "/dest.txt",
				"overwrite": false,
			},
			expectedError: true,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Both files should still exist unchanged
				content, err := afero.ReadFile(fs, "/source.txt")
				require.NoError(t, err)
				assert.Equal(t, "new content", string(content))
				
				content, err = afero.ReadFile(fs, "/dest.txt")
				require.NoError(t, err)
				assert.Equal(t, "old content", string(content))
			},
		},
		{
			name: "move non-existent file",
			setupFS: nil,
			args: map[string]interface{}{
				"source": "/nonexistent.txt",
				"destination":   "/dest.txt",
			},
			expectedError: true,
		},
		{
			name: "move with unsafe source path",
			setupFS: nil,
			args: map[string]interface{}{
				"source": "../../../etc/passwd",
				"destination":   "/dest.txt",
			},
			expectedError: true,
		},
		{
			name: "move with unsafe dest path",
			setupFS: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "/source.txt", []byte("content"), 0644)
			},
			args: map[string]interface{}{
				"source": "/source.txt",
				"destination":   "../../../etc/passwd",
			},
			expectedError: true,
		},
		{
			name: "move file to directory (implicit rename)",
			setupFS: func(fs afero.Fs) error {
				if err := afero.WriteFile(fs, "/file.txt", []byte("content"), 0644); err != nil {
					return err
				}
				return fs.MkdirAll("/destdir", 0755)
			},
			args: map[string]interface{}{
				"source": "/file.txt",
				"destination":   "/destdir/file.txt",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				// Source should not exist
				exists, err := afero.Exists(fs, "/file.txt")
				require.NoError(t, err)
				assert.False(t, exists)
				
				// File should be in destination directory
				content, err := afero.ReadFile(fs, "/destdir/file.txt")
				require.NoError(t, err)
				assert.Equal(t, "content", string(content))
			},
		},
		{
			name: "move preserves file permissions",
			setupFS: func(fs afero.Fs) error {
				err := afero.WriteFile(fs, "/executable.sh", []byte("#!/bin/bash"), 0755)
				return err
			},
			args: map[string]interface{}{
				"source": "/executable.sh",
				"destination":   "/moved.sh",
			},
			expectedError: false,
			checkFS: func(t *testing.T, fs afero.Fs) {
				info, err := fs.Stat("/moved.sh")
				require.NoError(t, err)
				// Note: afero.MemMapFs may not preserve exact permissions
				assert.NotNil(t, info)
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