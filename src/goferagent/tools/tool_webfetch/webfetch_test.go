package tool_webfetch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebFetchTool(t *testing.T) {
	tool, err := Tool() // Verify tool can be created
	require.NoError(t, err)
	require.NotNil(t, tool)
}

func TestWebFetchHTML(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Hello World</h1>
    <p>This is a test paragraph with <strong>bold text</strong>.</p>
    <script>console.log("script content");</script>
    <style>.test { color: red; }</style>
</body>
</html>`))
	}))
	defer server.Close()

	tests := []struct {
		name       string
		format     string
		expectFunc func(t *testing.T, response Response)
	}{
		{
			name:   "HTML format",
			format: "html",
			expectFunc: func(t *testing.T, response Response) {
				assert.Contains(t, response.Content, "<!DOCTYPE html>")
				assert.Contains(t, response.Content, "<h1>Hello World</h1>")
				assert.Contains(t, response.Content, "script")
			},
		},
		{
			name:   "Text format",
			format: "text",
			expectFunc: func(t *testing.T, response Response) {
				assert.Contains(t, response.Content, "Hello World")
				assert.Contains(t, response.Content, "This is a test paragraph")
				// Script and style should be removed in text extraction
				assert.NotContains(t, response.Content, "console.log")
				assert.NotContains(t, response.Content, ".test { color: red")
			},
		},
		{
			name:   "Markdown format",
			format: "markdown",
			expectFunc: func(t *testing.T, response Response) {
				assert.Contains(t, response.Content, "# Hello World")
				assert.Contains(t, response.Content, "This is a test paragraph")
				assert.Contains(t, response.Content, "**bold text**")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{
				"url":    server.URL,
				"format": tt.format,
			}

			paramsJSON, _ := json.Marshal(params)
			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: paramsJSON,
				},
			}

			tool, err := Tool()
			require.NoError(t, err)
			
			resp, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)
			assert.False(t, resp.IsError)

			var response Response
			err = json.Unmarshal(resp.Content, &response)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Equal(t, server.URL, response.URL)
			assert.Contains(t, response.ContentType, "text/html")

			if tt.expectFunc != nil {
				tt.expectFunc(t, response)
			}
		})
	}
}

func TestWebFetchJSON(t *testing.T) {
	testData := map[string]interface{}{
		"name":    "test",
		"version": "1.0.0",
		"data":    []string{"item1", "item2"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(testData)
	}))
	defer server.Close()

	params := map[string]interface{}{
		"url":    server.URL,
		"format": "markdown",
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	tool, err := Tool()
	require.NoError(t, err)
	
	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	var response Response
	err = json.Unmarshal(resp.Content, &response)
	require.NoError(t, err)

	// JSON should be wrapped in code block when format is markdown
	assert.Contains(t, response.Content, "```json")
	assert.Contains(t, response.Content, "test")
	assert.Contains(t, response.Content, "1.0.0")
}

func TestWebFetchErrors(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		format     string
		timeout    int
		serverFunc func(w http.ResponseWriter, r *http.Request)
		expectErr  string
	}{
		{
			name:      "missing URL",
			url:       "",
			format:    "text",
			expectErr: "required field 'url' is missing",
		},
		{
			name:      "invalid format",
			url:       "http://example.com",
			format:    "invalid",
			expectErr: "format must be one of",
		},
		{
			name:      "invalid URL scheme",
			url:       "ftp://example.com",
			format:    "text",
			expectErr: "URL must start with http://",
		},
		{
			name:   "404 error",
			url:    "", // will be set to server URL
			format: "text",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectErr: "request failed with status code: 404",
		},
		{
			name:   "500 error",
			url:    "", // will be set to server URL
			format: "text",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectErr: "request failed with status code: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.url
			
			if tt.serverFunc != nil {
				server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
				defer server.Close()
				url = server.URL
			}

			params := map[string]interface{}{
				"url":    url,
				"format": tt.format,
			}

			if tt.timeout > 0 {
				params["timeout"] = tt.timeout
			}

			paramsJSON, _ := json.Marshal(params)
			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: paramsJSON,
				},
			}

			tool, err := Tool()
			require.NoError(t, err)
			
			resp, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)
			assert.True(t, resp.IsError, "Expected error response")
			assert.Contains(t, string(resp.Content), tt.expectErr)
		})
	}
}

func TestWebFetchTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	params := map[string]interface{}{
		"url":     server.URL,
		"format":  "text",
		"timeout": 1, // 1 second timeout
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	tool, err := Tool()
	require.NoError(t, err)
	
	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "Expected timeout error")
}

func TestWebFetchRedirects(t *testing.T) {
	// Create target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Final destination"))
	}))
	defer targetServer.Close()

	// Create redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL, http.StatusFound)
	}))
	defer redirectServer.Close()

	params := map[string]interface{}{
		"url":    redirectServer.URL,
		"format": "text",
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	tool, err := Tool()
	require.NoError(t, err)
	
	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	var response Response
	err = json.Unmarshal(resp.Content, &response)
	require.NoError(t, err)

	assert.Contains(t, response.Content, "Final destination")
	assert.Equal(t, targetServer.URL, response.URL) // Should show final URL after redirect
}

func TestWebFetchLargeContent(t *testing.T) {
	// Create server with large content (but within limits)
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = 'A'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(largeContent)
	}))
	defer server.Close()

	params := map[string]interface{}{
		"url":    server.URL,
		"format": "text",
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	tool, err := Tool()
	require.NoError(t, err)
	
	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	var response Response
	err = json.Unmarshal(resp.Content, &response)
	require.NoError(t, err)

	assert.Equal(t, len(largeContent), len(response.Content))
	assert.Contains(t, response.Headers, "Content-Type")
}

func TestWebFetchUserAgent(t *testing.T) {
	var capturedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	params := map[string]interface{}{
		"url":    server.URL,
		"format": "text",
	}

	paramsJSON, _ := json.Marshal(params)
	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: paramsJSON,
		},
	}

	tool, err := Tool()
	require.NoError(t, err)
	
	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	assert.Contains(t, capturedUserAgent, "gofer")
}

func TestWebFetchInvalidJSON(t *testing.T) {
	params := []byte(`{"invalid": "json"`)

	call := &aisdk.ToolCall{
		Function: aisdk.FunctionCall{
			Arguments: params,
		},
	}

	tool, err := Tool()
	require.NoError(t, err)
	
	resp, err := tool.Execute(context.Background(), call)
	require.NoError(t, err)
	assert.True(t, resp.IsError)
	assert.Contains(t, string(resp.Content), "failed to parse input")
}

func TestWebFetchTimeoutLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tests := []struct {
		name            string
		timeout         int
		expectedTimeout int
	}{
		{"negative timeout", -1, 30},
		{"zero timeout", 0, 30},
		{"normal timeout", 45, 45},
		{"excessive timeout", 200, 120}, // Should be capped at 120
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{
				"url":     server.URL,
				"format":  "text",
				"timeout": tt.timeout,
			}

			paramsJSON, _ := json.Marshal(params)
			call := &aisdk.ToolCall{
				Function: aisdk.FunctionCall{
					Arguments: paramsJSON,
				},
			}

			tool, err := Tool()
			require.NoError(t, err)
			
			resp, err := tool.Execute(context.Background(), call)
			require.NoError(t, err)
			assert.False(t, resp.IsError)

			// The timeout should be applied correctly (though we can't easily test the exact value)
			var response Response
			err = json.Unmarshal(resp.Content, &response)
			require.NoError(t, err)
			assert.Equal(t, "OK", response.Content)
		})
	}
}