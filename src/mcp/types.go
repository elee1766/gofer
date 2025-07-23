package mcp

import (
	"context"
	"encoding/json"
	"time"
)

// Protocol version
const ProtocolVersion = "1.0.0"

// Message types for JSON-RPC
const (
	MessageTypeRequest      = "request"
	MessageTypeResponse     = "response"
	MessageTypeNotification = "notification"
	MessageTypeError        = "error"
)

// Standard JSON-RPC error codes
const (
	ErrorCodeParse          = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternal       = -32603
)

// Request methods
const (
	MethodInitialize      = "initialize"
	MethodListTools       = "tools/list"
	MethodCallTool        = "tools/call"
	MethodListResources   = "resources/list"
	MethodReadResource    = "resources/read"
	MethodListPrompts     = "prompts/list"
	MethodGetPrompt       = "prompts/get"
	MethodPing            = "ping"
	MethodSetLoggingLevel = "logging/setLevel"
)

// Message represents a JSON-RPC message
type Message struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeParams for the initialize request
type InitializeParams struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    ClientCapability  `json:"capabilities"`
	ClientInfo      *ClientInfo       `json:"clientInfo,omitempty"`
}

// InitializeResult from the initialize response
type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    ServerCapability  `json:"capabilities"`
	ServerInfo      *ServerInfo       `json:"serverInfo,omitempty"`
}

// ClientCapability describes client capabilities
type ClientCapability struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
}

// ServerCapability describes server capabilities
type ServerCapability struct {
	Tools        *ToolsCapability       `json:"tools,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// ToolsCapability indicates tool support
type ToolsCapability struct {
	ListTools bool `json:"listTools,omitempty"`
}

// ResourcesCapability indicates resource support
type ResourcesCapability struct {
	Subscribe   bool     `json:"subscribe,omitempty"`
	ListChanged bool     `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompt support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// SamplingCapability for message sampling
type SamplingCapability struct{}

// ClientInfo provides client identification
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo provides server identification
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema *SchemaObject  `json:"inputSchema"`
}

// SchemaObject represents a JSON Schema
type SchemaObject struct {
	Type        interface{}            `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Items       interface{}            `json:"items,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	// Additional JSON Schema fields as needed
}

// CallToolParams for tool execution
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult from tool execution
type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents a piece of content
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReadResourceParams for resource reading
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult from resource reading
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents resource content
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // Base64 encoded
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Arguments   []PromptArgument      `json:"arguments,omitempty"`
}

// PromptArgument describes a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// GetPromptParams for prompt retrieval
type GetPromptParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResult from prompt retrieval
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string          `json:"role"`
	Content PromptContent   `json:"content"`
}

// PromptContent can be string or structured content
type PromptContent interface{}

// LogEntry represents a log message
type LogEntry struct {
	Level   string `json:"level"`
	Logger  string `json:"logger,omitempty"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SetLoggingLevelParams for logging configuration
type SetLoggingLevelParams struct {
	Level string `json:"level"`
}

// Transport defines the interface for MCP communication
type Transport interface {
	// Send sends a message
	Send(ctx context.Context, message *Message) error
	
	// Receive receives a message
	Receive(ctx context.Context) (*Message, error)
	
	// Close closes the transport
	Close() error
}

// Server represents an MCP server connection
type Server interface {
	// Initialize initializes the connection
	Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error)
	
	// ListTools returns available tools
	ListTools(ctx context.Context) ([]Tool, error)
	
	// CallTool executes a tool
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error)
	
	// ListResources returns available resources
	ListResources(ctx context.Context) ([]Resource, error)
	
	// ReadResource reads a resource
	ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error)
	
	// ListPrompts returns available prompts
	ListPrompts(ctx context.Context) ([]Prompt, error)
	
	// GetPrompt retrieves a prompt
	GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*GetPromptResult, error)
	
	// Close closes the connection
	Close() error
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env,omitempty"`
	WorkingDir    string            `json:"workingDir,omitempty"`
	TransportType string            `json:"transportType,omitempty"` // "stdio" (default) or "http"
	URL           string            `json:"url,omitempty"`           // For HTTP transport
	Timeout       time.Duration     `json:"timeout,omitempty"`
}

// Manager manages multiple MCP servers
type Manager interface {
	// AddServer adds a server
	AddServer(config ServerConfig) error
	
	// RemoveServer removes a server
	RemoveServer(name string) error
	
	// GetServer gets a server by name
	GetServer(name string) Server
	
	// ListServers lists all servers
	ListServers() []string
	
	// Close closes all servers
	Close() error
}