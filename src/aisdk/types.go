// Package aisdk provides a framework for building AI-powered applications with tool/agent support.
package aisdk

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	jsonschema "github.com/swaggest/jsonschema-go"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// Name is required for tool responses to identify the function
	Name string `json:"name,omitempty"`
	// ToolCallID is required for tool responses to reference the original call
	ToolCallID string `json:"tool_call_id,omitempty"`
	// CacheControl is used for prompt caching with Anthropic models.
	CacheControl *CacheControl `json:"cache_control,omitempty"`
	// ToolCalls contains function calls requested by the assistant.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// Metadata for message tracking
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// CacheControl specifies caching behavior for a message.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral" for prompt caching
}


// ToolFunction represents the actual function definition within a tool
type ToolFunction struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  *jsonschema.Schema `json:"parameters"` // JSON Schema for parameters
}

// ToolExecutor is a function that executes a tool with given parameters
type ToolExecutor func(ctx context.Context, call *ToolCall) (*ToolResponse, error)

// ToolCall represents a function call request from the model (OpenAI format).
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // Always "function" for now
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments.
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResponse struct {
	Type     string `json:"type"`
	Content  []byte `json:"content"`
	Metadata string `json:"metadata,omitempty"`
	IsError  bool   `json:"is_error"`
}

// ChatCompletionRequest represents a request to the chat completions endpoint.
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []*Message             `json:"messages"`
	Temperature      *float64               `json:"temperature,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	Tools            []*ChatTool            `json:"tools,omitempty"`
	ToolChoice       string                 `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	User             string                 `json:"user,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

// ChatCompletionResponse represents a response from the chat completions endpoint.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"`
	Delta        *Message `json:"delta,omitempty"` // For streaming
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	// Provider specific fields
	PromptTokensCached int `json:"prompt_tokens_cached,omitempty"`
}

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Error represents an API error response.
type Error struct {
	Message string                 `json:"message"`
	Type    string                 `json:"type"`
	Code    string                 `json:"code,omitempty"`
	Param   string                 `json:"param,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorResponse wraps an error from the API.
type ErrorResponse struct {
	Error Error `json:"error"`
}

// ConversationOptions holds options for creating a new conversation.
type ConversationOptions struct {
	ID           string
	Model        string
	SystemPrompt string
	Temperature  *float64
	MaxTokens    *int
	Tools        map[string]*ChatTool
	MaxTurns     int
}

// ClientConfig holds the configuration for AI clients.
type ClientConfig struct {
	APIKey     string
	BaseURL    string
	RetryCount int
	RetryDelay time.Duration
	// Optional headers for ranking/identification
	SiteURL  string
	SiteName string
	// Optional logger
	Logger *slog.Logger
}

// StreamInterface defines the interface for reading streaming responses.
type StreamInterface interface {
	// Read reads the next chunk from the stream.
	Read() (*StreamChunk, error)

	// Close closes the stream.
	Close() error
}

// StreamReader defines a more comprehensive streaming interface with additional methods.
type StreamReader interface {
	StreamInterface

	// Err returns any error that occurred during streaming.
	Err() error

	// Done returns a channel that is closed when the stream is complete.
	Done() <-chan struct{}
}

// ChatCompletionStreamResponse represents the response for streaming chat completions.
type ChatCompletionStreamResponse struct {
	Stream StreamInterface

	// Optional: Metadata about the stream
	RequestID string
	Model     string
	Created   int64
}

// ModelInfo contains comprehensive information about a specific model
// Includes all fields from OpenRouter models endpoint
type ModelInfo struct {
	// Core identification
	ID            string `json:"id"`
	CanonicalSlug string `json:"canonical_slug,omitempty"` // OpenRouter's permanent slug
	Name          string `json:"name"`
	Created       int64  `json:"created,omitempty"` // Unix timestamp
	Description   string `json:"description"`
	ContextLength int    `json:"context_length"`

	// Model capabilities and settings
	Architecture        *Architecture `json:"architecture,omitempty"`
	Pricing             *Pricing      `json:"pricing,omitempty"`
	TopProvider         *TopProvider  `json:"top_provider,omitempty"`
	PerRequestLimits    interface{}   `json:"per_request_limits,omitempty"`
	SupportedParameters []string      `json:"supported_parameters,omitempty"`

	// Legacy fields for backward compatibility
	ContextLen          int         `json:"context_len,omitempty"`          // Deprecated: use ContextLength
	Modality            []string    `json:"modality,omitempty"`             // Deprecated: use Architecture.InputModalities
	SupportsAttachments bool        `json:"supports_attachments,omitempty"` // Backward compatibility field
	Parameters          *Parameters `json:"parameters,omitempty"`
}

// Pricing contains model pricing information from OpenRouter
type Pricing struct {
	Prompt            string `json:"prompt"`                       // Cost per input token
	Completion        string `json:"completion"`                   // Cost per output token
	Request           string `json:"request,omitempty"`            // Fixed cost per API request
	Image             string `json:"image,omitempty"`              // Cost per image input
	WebSearch         string `json:"web_search,omitempty"`         // Cost per web search
	InternalReasoning string `json:"internal_reasoning,omitempty"` // Cost for reasoning tokens
	InputCacheRead    string `json:"input_cache_read,omitempty"`   // Cost per cached input token read
	InputCacheWrite   string `json:"input_cache_write,omitempty"`  // Cost per cached input token write
}

// Architecture contains model architecture information from OpenRouter
type Architecture struct {
	InputModalities  []string `json:"input_modalities,omitempty"`  // e.g., ["text", "image"]
	OutputModalities []string `json:"output_modalities,omitempty"` // e.g., ["text"]
	Tokenizer        string   `json:"tokenizer,omitempty"`         // Tokenization method
	InstructType     *string  `json:"instruct_type,omitempty"`     // Instruction format type, can be null

	// Legacy field for backward compatibility
	Modality string `json:"modality,omitempty"` // Deprecated: use InputModalities
}

// Parameters contains model parameter information
type Parameters struct {
	MaxTemperature *float64 `json:"max_temperature,omitempty"`
	MinP           *float64 `json:"min_p,omitempty"`
	MaxTokens      *int     `json:"max_tokens,omitempty"`
}

// CostEstimate represents estimated costs for a request
type CostEstimate struct {
	PromptCost     float64 `json:"prompt_cost"`
	CompletionCost float64 `json:"completion_cost"`
	TotalCost      float64 `json:"total_cost"`
	Currency       string  `json:"currency"`
}

// TopProvider contains provider-specific information from OpenRouter
type TopProvider struct {
	ContextLength       int  `json:"context_length,omitempty"`        // Provider-specific context limit
	MaxCompletionTokens int  `json:"max_completion_tokens,omitempty"` // Maximum response tokens
	IsModerated         bool `json:"is_moderated,omitempty"`          // Content moderation status
}
