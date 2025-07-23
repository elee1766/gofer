package tool_grepfiles

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrepFilesTool(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create test files with various content
	testFiles := map[string]string{
		"/test/file1.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
		"/test/file2.js": `function hello() {
    console.log("Hello from JavaScript");
}

function goodbye() {
    console.log("Goodbye from JavaScript");
}`,
		"/test/file3.py": `def greet(name):
    print(f"Hello, {name}!")

def farewell(name):
    print(f"Goodbye, {name}!")`,
		"/test/binary.exe": string([]byte{0x00, 0x01, 0x02, 0x03}), // Binary file
		"/test/empty.txt":  "",
	}

	for path, content := range testFiles {
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0644))
	}

	tool, err := Tool(fs) // Verify tool can be created
	require.NoError(t, err)

	tests := []struct {
		name       string
		pattern    string
		path       string
		expectErr  bool
		expectFunc func(t *testing.T, response map[string]interface{})
	}{
		{
			name:    "find function declarations",
			pattern: "^func ",
			path:    "/test",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				assert.Len(t, matches, 1) // Should find main() in file1.go

				match := matches[0].(map[string]interface{})
				assert.Contains(t, match["file"], "file1.go")
				assert.Contains(t, match["content"], "func main()")
			},
		},
		{
			name:    "find console.log statements",
			pattern: "console\\.log",
			path:    "/test",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				assert.Len(t, matches, 2) // Should find 2 console.log statements in file2.js
			},
		},
		{
			name:    "case insensitive search",
			pattern: "hello",
			path:    "/test",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				// Should find "Hello" in multiple files (case insensitive by default)
				assert.GreaterOrEqual(t, len(matches), 3)
			},
		},
		{
			name:    "no matches",
			pattern: "nonexistent",
			path:    "/test",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok, "matches should be an array")
				assert.Len(t, matches, 0)
			},
		},
		{
			name:    "file pattern filter",
			pattern: "function",
			path:    "/test",
			expectFunc: func(t *testing.T, response map[string]interface{}) {
				matches, ok := response["matches"].([]interface{})
				require.True(t, ok)
				// Should find function declarations in JS file
				assert.GreaterOrEqual(t, len(matches), 2)
			},
		},
		{
			name:      "unsafe path",
			pattern:   "test",
			path:      "/etc",
			expectErr: true,
		},
		{
			name:      "invalid regex",
			pattern:   "[invalid",
			path:      "/test",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{
				"pattern": tt.pattern,
				"path":    tt.path,
			}

			// Special case for file pattern test
			if tt.name == "file pattern filter" {
				params["file_pattern"] = "*.js"
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
			assert.Contains(t, response, "total_matches")

			if tt.expectFunc != nil {
				tt.expectFunc(t, response)
			}
		})
	}
}

func TestGrepFilesContextLines(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	content := `line1
line2
target line
line4
line5`

	require.NoError(t, afero.WriteFile(fs, "/test.txt", []byte(content), 0644))

	tool, err := Tool(fs) // Verify tool can be created
	require.NoError(t, err)

	params := map[string]interface{}{
		"pattern":       "target",
		"path":          "/",
		"context_lines": 2,
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
	assert.Equal(t, "target line", match["content"])
	assert.Equal(t, 3, int(match["line"].(float64))) // Line numbers are 1-indexed

	// Check context is included
	context, ok := match["context"].([]interface{})
	require.True(t, ok)
	assert.Len(t, context, 5) // 2 lines before + target line + 2 lines after
}

func TestGrepFilesMaxResults(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a file with many matches
	content := ""
	for i := 0; i < 200; i++ {
		content += "target line\n"
	}

	require.NoError(t, afero.WriteFile(fs, "/many_matches.txt", []byte(content), 0644))

	tool, err := Tool(fs)
	require.NoError(t, err)

	params := map[string]interface{}{
		"pattern":     "target",
		"path":        "/",
		"max_results": 50,
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
	assert.Len(t, matches, 50) // Should be limited to max_results

	truncated, ok := response["truncated"].(bool)
	require.True(t, ok)
	assert.True(t, truncated) // Should indicate results were truncated
}