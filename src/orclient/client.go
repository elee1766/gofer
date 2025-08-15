package orclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/elee1766/gofer/src/aisdk"
)

const (
	defaultBaseURL = "https://openrouter.ai/api/v1"
	defaultTimeout = 30 * time.Second
)

var _ aisdk.Provider = (*Client)(nil)

// Client is the OpenRouter API client.
type Client struct {
	config     Config
	httpClient *http.Client
	logger     *slog.Logger
	baseURL    string
	apiKey     string
	modelCache *ModelCache
}

// NewClient creates a new OpenRouter API client.
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "openrouter_client")

	client := &Client{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		baseURL:    config.BaseURL,
		apiKey:     config.APIKey,
	}

	// Initialize model cache with 1 hour TTL
	client.modelCache = NewModelCache(client, time.Hour)

	return client
}

// createChatCompletion sends a chat completion request to OpenRouter (internal method).
func (c *Client) createChatCompletion(ctx context.Context, req *aisdk.ChatCompletionRequest) (*aisdk.ChatCompletionResponse, error) {

	logger := c.logger.With("method", "CreateChatCompletion", "model", req.Model)
	logger.Debug("sending chat completion request")

	// Format the request based on the provider
	formattedReq := c.formatRequestForProvider(req)

	// Debug log the formatted request
	if c.logger.Enabled(ctx, slog.LevelDebug) {
		if debugBody, err := json.MarshalIndent(formattedReq, "", "  "); err == nil {
			logger.Debug("formatted request", "body", string(debugBody))
		}
	}

	body, err := json.Marshal(formattedReq)
	if err != nil {
		logger.Error("failed to marshal request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, "POST", "/chat/completions", body)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestWithRetry(httpReq)
	if err != nil {
		logger.Error("request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("received error response", "status_code", resp.StatusCode)
		return nil, c.handleError(resp)
	}

	var result aisdk.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("failed to decode response", "error", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Info("chat completion successful",
		"usage_total", result.Usage.TotalTokens,
		"usage_cached", result.Usage.PromptTokensCached)
	return &result, nil
}


// newRequest creates a new HTTP request with the appropriate headers.
func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	url := c.config.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Optional headers for ranking
	if c.config.SiteURL != "" {
		req.Header.Set("HTTP-Referer", c.config.SiteURL)
	}
	if c.config.SiteName != "" {
		req.Header.Set("X-Title", c.config.SiteName)
	}

	return req, nil
}

// doRequestWithRetry performs an HTTP request with retry logic.
func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	logger := c.logger.With("method", "doRequestWithRetry", "url", req.URL.String())

	for i := 0; i < c.config.RetryCount; i++ {
		// Clone the request for retry
		reqCopy := req.Clone(req.Context())
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			reqCopy.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.httpClient.Do(reqCopy)
		if err != nil {
			lastErr = err
			logger.Debug("request attempt failed", "attempt", i+1, "error", err)
			time.Sleep(c.config.RetryDelay * time.Duration(i+1))
			continue
		}

		// Don't retry on client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, nil
		}

		// Success or client error - return immediately
		if resp.StatusCode < 400 {
			return resp, nil
		}

		// Server error - retry
		resp.Body.Close()
		lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
		logger.Debug("server error, retrying", "attempt", i+1, "status_code", resp.StatusCode)
		time.Sleep(c.config.RetryDelay * time.Duration(i+1))
	}

	logger.Error("request failed after all retries", "retry_count", c.config.RetryCount, "error", lastErr)
	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.RetryCount, lastErr)
}

// handleError processes error responses from the API.
func (c *Client) handleError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// Return a basic API error if we can't parse the response
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			RequestID:  resp.Header.Get("X-Request-ID"),
		}
	}

	// Create a structured API error
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Type:       errResp.Error.Type,
		Message:    errResp.Error.Message,
		Code:       errResp.Error.Code,
		Param:      errResp.Error.Param,
		Details:    errResp.Error.Details,
		RequestID:  resp.Header.Get("X-Request-ID"),
	}

	// Add retry-after information for rate limits
	if resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if apiErr.Details == nil {
				apiErr.Details = make(map[string]interface{})
			}
			apiErr.Details["retry_after"] = retryAfter
		}
	}

	return apiErr
}


// GetModels implements ModelContextProvider.GetModels
func (c *Client) GetModels(ctx context.Context) ([]*aisdk.ModelInfo, error) {
	return c.ListModels(ctx)
}

// formatRequestForProvider formats the request based on the provider requirements
func (c *Client) formatRequestForProvider(req *aisdk.ChatCompletionRequest) interface{} {
	provider := c.detectProvider(req.Model)

	switch provider {
	case "anthropic":
		return c.formatAnthropicRequest(req)
	case "google":
		return c.formatGoogleRequest(req)
	case "openai":
		return c.formatOpenAIRequest(req)
	default:
		// Return as-is for unknown providers
		return req
	}
}

// detectProvider detects the provider from the model name
func (c *Client) detectProvider(model string) string {
	if strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "claude") {
		return "anthropic"
	}
	if strings.HasPrefix(model, "google/") || strings.HasPrefix(model, "gemini") {
		return "google"
	}
	if strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "gpt") {
		return "openai"
	}
	return "unknown"
}

// formatAnthropicRequest formats the request for Anthropic models
func (c *Client) formatAnthropicRequest(req *aisdk.ChatCompletionRequest) interface{} {
	// Create a custom type to handle Anthropic-specific formatting
	type AnthropicMessage struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		Name      string           `json:"name,omitempty"`
		ToolUseID string           `json:"tool_use_id,omitempty"` // Anthropic uses tool_use_id
		ToolCalls []aisdk.ToolCall `json:"tool_calls,omitempty"`
	}

	type AnthropicRequest struct {
		Model       string             `json:"model"`
		Messages    []AnthropicMessage `json:"messages"`
		Temperature *float64           `json:"temperature,omitempty"`
		MaxTokens   *int               `json:"max_tokens,omitempty"`
		Stream      bool               `json:"stream,omitempty"`
		Tools       []*aisdk.ChatTool  `json:"tools,omitempty"`
	}

	// Convert messages
	anthropicMessages := make([]AnthropicMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg == nil {
			continue
		}
		// Need to handle tool calls conversion for Anthropic
		var toolCalls []aisdk.ToolCall
		if len(msg.ToolCalls) > 0 {
			// Anthropic doesn't use the nested function structure
			toolCalls = make([]aisdk.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i] = aisdk.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: aisdk.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Name:      msg.Name,
			ToolUseID: msg.ToolCallID, // Map ToolCallID to tool_use_id
			ToolCalls: toolCalls,
		})
	}

	return AnthropicRequest{
		Model:       req.Model,
		Messages:    anthropicMessages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		Tools:       req.Tools,
	}
}

// formatGoogleRequest formats the request for Google models
func (c *Client) formatGoogleRequest(req *aisdk.ChatCompletionRequest) interface{} {
	// For Google, ensure tool response names are not empty
	type GoogleMessage struct {
		Role       string           `json:"role"`
		Content    string           `json:"content"`
		Name       string           `json:"name,omitempty"`
		ToolCallID string           `json:"tool_call_id,omitempty"`
		ToolCalls  []aisdk.ToolCall `json:"tool_calls,omitempty"`
	}

	// Convert messages and ensure tool names are present
	googleMessages := make([]GoogleMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg == nil {
			continue
		}
		googleMsg := GoogleMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  msg.ToolCalls,
		}

		// Ensure tool responses have names for Google
		if msg.Role == "tool" && googleMsg.Name == "" {
			googleMsg.Name = "tool_response"
		}
		
		// Google requires all messages to have content
		// If assistant message has only tool calls, add a placeholder
		if msg.Role == "assistant" && googleMsg.Content == "" && len(googleMsg.ToolCalls) > 0 {
			googleMsg.Content = "I'll help you with that."
		}

		googleMessages = append(googleMessages, googleMsg)
	}

	// Return modified request
	type GoogleRequest struct {
		Model       string          `json:"model"`
		Messages    []GoogleMessage `json:"messages"`
		Temperature *float64        `json:"temperature,omitempty"`
		MaxTokens   *int            `json:"max_tokens,omitempty"`
		Stream      bool            `json:"stream,omitempty"`
		Tools       []*aisdk.ChatTool `json:"tools,omitempty"`
	}

	return GoogleRequest{
		Model:       req.Model,
		Messages:    googleMessages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		Tools:       req.Tools,
	}
}

// formatOpenAIRequest formats the request for OpenAI models
func (c *Client) formatOpenAIRequest(req *aisdk.ChatCompletionRequest) interface{} {
	// Since we're now using OpenAI format natively, just ensure Arguments are not null
	type OpenAIMessage struct {
		Role       string            `json:"role"`
		Content    string            `json:"content"`
		Name       string            `json:"name,omitempty"`
		ToolCallID string            `json:"tool_call_id,omitempty"`
		ToolCalls  []aisdk.ToolCall  `json:"tool_calls,omitempty"`
	}

	// Convert messages
	openaiMessages := make([]OpenAIMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg == nil {
			continue
		}
		openaiMsg := OpenAIMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  msg.ToolCalls,
		}

		// Ensure tool calls have proper type and non-null arguments
		for i := range openaiMsg.ToolCalls {
			if openaiMsg.ToolCalls[i].Type == "" {
				openaiMsg.ToolCalls[i].Type = "function"
			}
			if openaiMsg.ToolCalls[i].Function.Arguments == nil {
				openaiMsg.ToolCalls[i].Function.Arguments = json.RawMessage("{}")
			}
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Return the formatted request
	type OpenAIRequest struct {
		Model       string            `json:"model"`
		Messages    []OpenAIMessage   `json:"messages"`
		Temperature *float64          `json:"temperature,omitempty"`
		MaxTokens   *int              `json:"max_tokens,omitempty"`
		Stream      bool              `json:"stream,omitempty"`
		Tools       []*aisdk.ChatTool   `json:"tools,omitempty"`
	}

	return OpenAIRequest{
		Model:       req.Model,
		Messages:    openaiMessages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		Tools:       req.Tools,
	}
}
