package shell

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// ShellManager manages persistent shell sessions per conversation
type ShellManager struct {
	shells map[string]*PersistentShell // conversationID -> shell
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewShellManager creates a new shell manager
func NewShellManager(logger *slog.Logger) *ShellManager {
	return &ShellManager{
		shells: make(map[string]*PersistentShell),
		logger: logger,
	}
}

// GetShell returns the shell for a conversation, creating one if needed
func (sm *ShellManager) GetShell(conversationID string) (*PersistentShell, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if shell already exists
	if shell, exists := sm.shells[conversationID]; exists {
		return shell, nil
	}

	// Create new shell
	shell, err := NewPersistentShell(sm.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create shell for conversation %s: %w", conversationID, err)
	}

	sm.shells[conversationID] = shell
	sm.logger.Info("created new shell session", "conversation_id", conversationID, "session_id", shell.GetSessionID())
	
	return shell, nil
}

// CloseShell closes and removes the shell for a conversation
func (sm *ShellManager) CloseShell(conversationID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	shell, exists := sm.shells[conversationID]
	if !exists {
		return nil // Already closed or never existed
	}

	err := shell.Close()
	delete(sm.shells, conversationID)
	
	sm.logger.Info("closed shell session", "conversation_id", conversationID)
	return err
}

// CloseAllShells closes all shell sessions
func (sm *ShellManager) CloseAllShells() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var lastErr error
	for conversationID, shell := range sm.shells {
		if err := shell.Close(); err != nil {
			sm.logger.Error("failed to close shell", "conversation_id", conversationID, "error", err)
			lastErr = err
		}
	}

	sm.shells = make(map[string]*PersistentShell)
	sm.logger.Info("closed all shell sessions")
	
	return lastErr
}

// ExecuteCommand executes a command in the shell for a specific conversation
func (sm *ShellManager) ExecuteCommand(ctx context.Context, conversationID, command string, timeout time.Duration) (*ShellResult, error) {
	shell, err := sm.GetShell(conversationID)
	if err != nil {
		return nil, err
	}

	// Validate command safety
	if err := shell.ValidateCommand(command); err != nil {
		return nil, err
	}

	// Execute the command
	result, err := shell.ExecuteCommand(ctx, command, timeout)
	if err != nil {
		return nil, err
	}

	// Security check: if we've navigated outside the original directory, reset
	if !shell.IsPathWithinOriginal() {
		sm.logger.Warn("shell navigated outside original directory, resetting", 
			"conversation_id", conversationID,
			"current_dir", shell.GetCurrentDirectory(), 
			"original_dir", shell.GetOriginalDirectory())
		
		if resetErr := shell.ResetToOriginalDirectory(ctx); resetErr != nil {
			sm.logger.Error("failed to reset shell to original directory", "error", resetErr)
			// Close the compromised shell
			sm.CloseShell(conversationID)
			return nil, fmt.Errorf("shell navigated outside allowed directory and reset failed: %w", resetErr)
		}
	}

	return result, nil
}

// GetCurrentDirectory returns the current directory for a conversation's shell
func (sm *ShellManager) GetCurrentDirectory(conversationID string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shell, exists := sm.shells[conversationID]
	if !exists {
		// Return empty string if no shell exists yet
		return "", nil
	}

	return shell.GetCurrentDirectory(), nil
}

// GetShellInfo returns information about the shell session
func (sm *ShellManager) GetShellInfo(conversationID string) map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shell, exists := sm.shells[conversationID]
	if !exists {
		return map[string]interface{}{
			"exists": false,
		}
	}

	return map[string]interface{}{
		"exists":             true,
		"session_id":         shell.GetSessionID(),
		"current_directory":  shell.GetCurrentDirectory(),
		"original_directory": shell.GetOriginalDirectory(),
		"within_original":    shell.IsPathWithinOriginal(),
	}
}

// Package-level shell manager instance
var globalShellManager *ShellManager

// Package-level variable to store current conversation ID
var currentConversationID string

// InitializeShellManager initializes the global shell manager
func InitializeShellManager(logger *slog.Logger) {
	globalShellManager = NewShellManager(logger)
}

// GetGlobalShellManager returns the global shell manager
func GetGlobalShellManager() *ShellManager {
	return globalShellManager
}

// GetCurrentWorkingDirectory returns the current working directory for a conversation
// This is a convenience function for use in system prompts and other contexts
func GetCurrentWorkingDirectory(conversationID string) string {
	if globalShellManager == nil {
		return ""
	}
	
	dir, err := globalShellManager.GetCurrentDirectory(conversationID)
	if err != nil || dir == "" {
		// Fallback to process working directory
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
	}
	
	return dir
}

// SetConversationContext sets the current conversation ID for tool execution
func SetConversationContext(conversationID string) {
	currentConversationID = conversationID
}

// GetConversationContext returns the current conversation ID
func GetConversationContext() string {
	return currentConversationID
}