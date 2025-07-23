package tool_searchfiles

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchFilesTool(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create test files with various content
	testFiles := map[string]string{
		"/project/main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	log.Error("This is an error message")
}`,
		"/project/utils.go": `package main

import "log"

func logError(msg string) {
	log.Error(msg)
}`,
		"/project/frontend/app.js": `console.log("Starting app");
var errorCount = 0;

function handleError(err) {
    console.error("Error occurred:", err);
    errorCount++;
}`,
		"/project/docs/readme.md": `# Project Documentation

This project demonstrates error handling patterns.

## Error Handling
- Use log.Error for Go errors
- Use console.error for JavaScript errors`,
		"/project/binary.exe": string([]byte{0x00, 0x01, 0x02, 0x03}), // Binary file
		"/project/empty.txt":   "",
	}

	for path, content := range testFiles {
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0644))
	}

	tool, err := Tool(fs) // Verify tool can be created
	require.NoError(t, err)

	tests := []struct {
		name        string
		pattern     string
		path        string
		filePattern string
		expectErr   bool
		expectFunc  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:    "search for error patterns",
			pattern: "Error",
			path:    "/project",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				// Should find "Error" in multiple files
				assert.GreaterOrEqual(t, len(matches), 3)

				// Verify match structure
				match := matches[0].(map[string]interface{})
				assert.Contains(t, match, "file")
				assert.Contains(t, match, "line")
				assert.Contains(t, match, "content")
				assert.Contains(t, match, "context")
			},
		},
		{
			name:        "search in Go files only",
			pattern:     "log\\.",
			path:        "/project",
			filePattern: "*.go",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				
				// All matches should be from .go files
				for _, match := range matches {
					m := match.(map[string]interface{})
					file := m["file"].(string)
					assert.Contains(t, file, ".go")
				}
				
				assert.GreaterOrEqual(t, len(matches), 2) // log.Error calls
			},
		},
		{
			name:    "regex pattern search",
			pattern: "console\\.(log|error)",
			path:    "/project",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				assert.GreaterOrEqual(t, len(matches), 2) // console.log and console.error
			},
		},
		{
			name:    "simple string search fallback",
			pattern: "[invalid regex",
			path:    "/project",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				// Should fallback to string search and find no matches for this invalid pattern
				assert.Equal(t, 0, len(matches))
			},
		},
		{
			name:    "search in specific subdirectory",
			pattern: "console",
			path:    "/project/frontend",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				assert.GreaterOrEqual(t, len(matches), 2) // console.log and console.error

				// All matches should be from the frontend directory
				for _, match := range matches {
					m := match.(map[string]interface{})
					file := m["file"].(string)
					assert.Contains(t, file, "/frontend/")
				}
			},
		},
		{
			name:    "no matches found",
			pattern: "nonexistent_pattern_xyz",
			path:    "/project",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok, "matches should be an array")
				assert.Len(t, matches, 0)

				count, ok := response["count"].(float64)
				require.True(t, ok, "count should be a number")
				assert.Equal(t, 0.0, count)
			},
		},
		{
			name:      "unsafe path",
			pattern:   "test",
			path:      "/etc",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{
				"pattern": tt.pattern,
				"path":    tt.path,
			}

			if tt.filePattern != "" {
				params["file_pattern"] = tt.filePattern
			}

			paramsJSON, _ := json.Marshal(params)

			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: paramsJSON,
				},
			}

			resp, err := tool.Execute(context.Background(), call)

			if tt.expectErr {
				assert.True(t, resp.IsError, "Expected error response")
				return
			}

			require.NoError(t, err)
			assert.False(t, resp.IsError, "Expected successful response")

			var response map[string]interface{}
			err = json.Unmarshal(resp.Content, &response)
			require.NoError(t, err)

			// Verify common response structure
			assert.Equal(t, tt.pattern, response["pattern"])
			assert.Equal(t, tt.path, response["path"])
			assert.Contains(t, response, "matches")
			assert.Contains(t, response, "count")

			if tt.expectFunc != nil {
				tt.expectFunc(t, response)
			}
		})
	}
}

func TestSearchFilesContext(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	content := `line 1 before
line 2 before
target line with pattern
line 4 after
line 5 after`

	require.NoError(t, afero.WriteFile(fs, "/test.txt", []byte(content), 0644))

	tool, err := Tool(fs)
	require.NoError(t, err)

	params := map[string]interface{}{
		"pattern": "target",
		"path":    "/",
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	var response map[string]interface{}
	err = json.Unmarshal(resp.Content, &response)
	require.NoError(t, err)

	matches, ok := response["matches"].([]interface{})
	require.True(t, ok)
	assert.Len(t, matches, 1)

	match := matches[0].(map[string]interface{})
	assert.Equal(t, "target line with pattern", match["content"])
	assert.Equal(t, 3, int(match["line"].(float64))) // Line numbers are 1-indexed
	
	// Check context is provided (2 lines before and after by default)
	context, ok := match["context"].([]interface{})
	require.True(t, ok)
	assert.Len(t, context, 5) // 2 before + target + 2 after
}

func TestSearchFilesBinarySkip(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create binary file
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	require.NoError(t, afero.WriteFile(fs, "/binary.bin", binaryContent, 0644))

	// Create text file
	textContent := "This is a text file with target pattern"
	require.NoError(t, afero.WriteFile(fs, "/text.txt", []byte(textContent), 0644))

	tool, err := Tool(fs)
	require.NoError(t, err)

	params := map[string]interface{}{
		"pattern": "target",
		"path":    "/",
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	var response map[string]interface{}
	err = json.Unmarshal(resp.Content, &response)
	require.NoError(t, err)

	matches, ok := response["matches"].([]interface{})
	require.True(t, ok)
	assert.Len(t, matches, 1) // Should only find match in text file, binary file should be skipped

	match := matches[0].(map[string]interface{})
	file := match["file"].(string)
	assert.Contains(t, file, "text.txt")
}

func TestSearchFilesEmptyPath(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create file in current directory
	require.NoError(t, afero.WriteFile(fs, "/current.txt", []byte("target pattern"), 0644))

	tool, err := Tool(fs)
	require.NoError(t, err)

	params := map[string]interface{}{
		"pattern": "target",
		// path not specified, should default to "."
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	var response map[string]interface{}
	err = json.Unmarshal(resp.Content, &response)
	require.NoError(t, err)

	// Should default path to "."
	assert.Equal(t, ".", response["path"])
}