package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// server implements the Server interface
type server struct {
	config    ServerConfig
	transport Transport
	
	// Request handling
	requestID   atomic.Int64
	pending     map[interface{}]chan *Message
	pendingMu   sync.Mutex
	
	// State
	initialized bool
	initMu      sync.Mutex
	capabilities ServerCapability
	serverInfo   *ServerInfo
	
	// Background processing
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewServer creates a new MCP server connection
func NewServer(config ServerConfig) (Server, error) {
	// Default timeout
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	// Create transport based on type
	var transport Transport
	var err error
	
	switch config.TransportType {
	case "", "stdio":
		transport, err = NewStdioTransport(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio transport: %w", err)
		}
	case "http":
		return nil, fmt.Errorf("HTTP transport not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.TransportType)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	s := &server{
		config:    config,
		transport: transport,
		pending:   make(map[interface{}]chan *Message),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// Start background message receiver
	s.wg.Add(1)
	go s.receiveLoop()
	
	return s, nil
}

// receiveLoop continuously receives messages in the background
func (s *server) receiveLoop() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		
		msg, err := s.transport.Receive(s.ctx)
		if err != nil {
			if s.ctx.Err() != nil {
				return // Context cancelled
			}
			slog.Error("error receiving message", "error", err)
			continue
		}
		
		// Handle the message
		if msg.ID != nil {
			// This is a response to our request
			s.pendingMu.Lock()
			if ch, ok := s.pending[msg.ID]; ok {
				select {
				case ch <- msg:
				default:
					slog.Warn("response channel full", "id", msg.ID)
				}
				delete(s.pending, msg.ID)
			}
			s.pendingMu.Unlock()
		} else {
			// This is a notification or request from server
			// For now, we just log it
			slog.Info("received server message", "method", msg.Method)
		}
	}
}

// sendRequest sends a request and waits for response
func (s *server) sendRequest(ctx context.Context, method string, params interface{}) (*Message, error) {
	// Generate request ID
	id := s.requestID.Add(1)
	
	// Prepare params
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}
	
	// Create request message
	req := &Message{
		ID:     id,
		Method: method,
		Params: paramsJSON,
	}
	
	// Create response channel
	respCh := make(chan *Message, 1)
	s.pendingMu.Lock()
	s.pending[id] = respCh
	s.pendingMu.Unlock()
	
	// Send the request
	if err := s.transport.Send(ctx, req); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Wait for response
	select {
	case <-ctx.Done():
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return nil, ctx.Err()
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("server error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-time.After(s.config.Timeout):
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// Initialize initializes the connection
func (s *server) Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
	s.initMu.Lock()
	defer s.initMu.Unlock()
	
	if s.initialized {
		return nil, fmt.Errorf("already initialized")
	}
	
	// Send initialize request
	resp, err := s.sendRequest(ctx, MethodInitialize, params)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}
	
	// Store capabilities and info
	s.capabilities = result.Capabilities
	s.serverInfo = result.ServerInfo
	s.initialized = true
	
	slog.Info("MCP server initialized", 
		"server", s.config.Name,
		"serverInfo", s.serverInfo,
		"capabilities", s.capabilities)
	
	return &result, nil
}

// ListTools returns available tools
func (s *server) ListTools(ctx context.Context) ([]Tool, error) {
	if !s.initialized {
		return nil, fmt.Errorf("not initialized")
	}
	
	// Check capability
	if s.capabilities.Tools == nil || !s.capabilities.Tools.ListTools {
		return []Tool{}, nil // Server doesn't support tools
	}
	
	// Send request
	resp, err := s.sendRequest(ctx, MethodListTools, nil)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}
	
	return result.Tools, nil
}

// CallTool executes a tool
func (s *server) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("not initialized")
	}
	
	// Check capability
	if s.capabilities.Tools == nil {
		return nil, fmt.Errorf("server doesn't support tools")
	}
	
	// Prepare params
	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}
	
	// Send request
	resp, err := s.sendRequest(ctx, MethodCallTool, params)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
	}
	
	return &result, nil
}

// ListResources returns available resources
func (s *server) ListResources(ctx context.Context) ([]Resource, error) {
	if !s.initialized {
		return nil, fmt.Errorf("not initialized")
	}
	
	// Check capability
	if s.capabilities.Resources == nil {
		return []Resource{}, nil // Server doesn't support resources
	}
	
	// Send request
	resp, err := s.sendRequest(ctx, MethodListResources, nil)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result struct {
		Resources []Resource `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}
	
	return result.Resources, nil
}

// ReadResource reads a resource
func (s *server) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("not initialized")
	}
	
	// Check capability
	if s.capabilities.Resources == nil {
		return nil, fmt.Errorf("server doesn't support resources")
	}
	
	// Prepare params
	params := ReadResourceParams{
		URI: uri,
	}
	
	// Send request
	resp, err := s.sendRequest(ctx, MethodReadResource, params)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource result: %w", err)
	}
	
	return &result, nil
}

// ListPrompts returns available prompts
func (s *server) ListPrompts(ctx context.Context) ([]Prompt, error) {
	if !s.initialized {
		return nil, fmt.Errorf("not initialized")
	}
	
	// Check capability
	if s.capabilities.Prompts == nil {
		return []Prompt{}, nil // Server doesn't support prompts
	}
	
	// Send request
	resp, err := s.sendRequest(ctx, MethodListPrompts, nil)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompts: %w", err)
	}
	
	return result.Prompts, nil
}

// GetPrompt retrieves a prompt
func (s *server) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*GetPromptResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("not initialized")
	}
	
	// Check capability
	if s.capabilities.Prompts == nil {
		return nil, fmt.Errorf("server doesn't support prompts")
	}
	
	// Prepare params
	params := GetPromptParams{
		Name:      name,
		Arguments: arguments,
	}
	
	// Send request
	resp, err := s.sendRequest(ctx, MethodGetPrompt, params)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt result: %w", err)
	}
	
	return &result, nil
}

// Close closes the connection
func (s *server) Close() error {
	// Cancel context to stop background goroutines
	s.cancel()
	
	// Wait for background goroutines
	s.wg.Wait()
	
	// Close transport
	if err := s.transport.Close(); err != nil {
		return fmt.Errorf("failed to close transport: %w", err)
	}
	
	return nil
}