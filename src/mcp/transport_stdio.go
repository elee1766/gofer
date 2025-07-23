package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// StdioTransport implements Transport for stdio-based communication
type StdioTransport struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	scanner  *bufio.Scanner
	encoder  *json.Encoder
	mu       sync.Mutex
	closed   atomic.Bool
	stderrBuf []byte
	stderrMu  sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(config ServerConfig) (*StdioTransport, error) {
	cmd := exec.Command(config.Command, config.Args...)
	
	// Set environment
	cmd.Env = os.Environ()
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Set working directory
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}
	
	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	
	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}
	
	transport := &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: bufio.NewScanner(stdout),
		encoder: json.NewEncoder(stdin),
		stderrBuf: make([]byte, 0, 4096),
	}
	
	// Start stderr reader
	go transport.readStderr()
	
	// Set scanner to read line by line (JSON-RPC messages are newline-delimited)
	transport.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max message size
	
	return transport, nil
}

// readStderr reads stderr in the background
func (t *StdioTransport) readStderr() {
	buf := make([]byte, 4096)
	for {
		n, err := t.stderr.Read(buf)
		if err != nil {
			if err != io.EOF {
				slog.Error("error reading stderr", "error", err)
			}
			return
		}
		
		if n > 0 {
			t.stderrMu.Lock()
			t.stderrBuf = append(t.stderrBuf, buf[:n]...)
			// Log stderr output
			slog.Debug("MCP server stderr", "output", string(buf[:n]))
			t.stderrMu.Unlock()
		}
	}
}

// Send sends a message
func (t *StdioTransport) Send(ctx context.Context, message *Message) error {
	if t.closed.Load() {
		return fmt.Errorf("transport is closed")
	}
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Ensure JSON-RPC version
	message.Jsonrpc = "2.0"
	
	// Log outgoing message
	if data, err := json.Marshal(message); err == nil {
		slog.Debug("MCP sending message", "message", string(data))
	}
	
	// Send the message
	if err := t.encoder.Encode(message); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}
	
	return nil
}

// Receive receives a message
func (t *StdioTransport) Receive(ctx context.Context) (*Message, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("transport is closed")
	}
	
	// Create a channel for the result
	resultCh := make(chan struct {
		msg *Message
		err error
	}, 1)
	
	go func() {
		// Read next line
		if !t.scanner.Scan() {
			if err := t.scanner.Err(); err != nil {
				resultCh <- struct {
					msg *Message
					err error
				}{nil, fmt.Errorf("scanner error: %w", err)}
			} else {
				resultCh <- struct {
					msg *Message
					err error
				}{nil, io.EOF}
			}
			return
		}
		
		// Parse the message
		line := t.scanner.Bytes()
		slog.Debug("MCP received line", "line", string(line))
		
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			resultCh <- struct {
				msg *Message
				err error
			}{nil, fmt.Errorf("failed to unmarshal message: %w", err)}
			return
		}
		
		resultCh <- struct {
			msg *Message
			err error
		}{&msg, nil}
	}()
	
	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		return result.msg, result.err
	}
}

// Close closes the transport
func (t *StdioTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Close stdin to signal the process
	if t.stdin != nil {
		t.stdin.Close()
	}
	
	// Terminate the process
	if t.cmd != nil && t.cmd.Process != nil {
		// Try graceful shutdown first
		t.cmd.Process.Signal(os.Interrupt)
		
		// Wait for a short time
		done := make(chan error, 1)
		go func() {
			done <- t.cmd.Wait()
		}()
		
		select {
		case <-done:
			// Process exited gracefully
		default:
			// Force kill after timeout
			t.cmd.Process.Kill()
			<-done
		}
	}
	
	// Close other pipes
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.stderr != nil {
		t.stderr.Close()
	}
	
	// Log any remaining stderr
	t.stderrMu.Lock()
	if len(t.stderrBuf) > 0 {
		slog.Debug("MCP server final stderr", "output", string(t.stderrBuf))
	}
	t.stderrMu.Unlock()
	
	return nil
}

// GetStderr returns accumulated stderr output
func (t *StdioTransport) GetStderr() []byte {
	t.stderrMu.Lock()
	defer t.stderrMu.Unlock()
	
	result := make([]byte, len(t.stderrBuf))
	copy(result, t.stderrBuf)
	return result
}