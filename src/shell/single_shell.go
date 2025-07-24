package shell

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// SingleShellManager manages a single persistent shell session for CLI usage
type SingleShellManager struct {
	shell  *PersistentShell
	logger *slog.Logger
}

// NewSingleShellManager creates a new manager with a single persistent shell
func NewSingleShellManager(logger *slog.Logger) (*SingleShellManager, error) {
	shell, err := NewPersistentShell(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create persistent shell: %w", err)
	}

	return &SingleShellManager{
		shell:  shell,
		logger: logger,
	}, nil
}

// ExecuteCommand executes a command in the persistent shell
func (sm *SingleShellManager) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (*ShellResult, error) {
	// Validate command safety
	if err := sm.shell.ValidateCommand(command); err != nil {
		return nil, err
	}

	// Execute the command
	result, err := sm.shell.ExecuteCommand(ctx, command, timeout)
	if err != nil {
		return nil, err
	}

	// Security check: if we've navigated outside the original directory, fail the command
	if !sm.shell.IsPathWithinOriginal() {
		sm.logger.Error("command attempted to navigate outside original directory", 
			"current_dir", sm.shell.GetCurrentDirectory(), 
			"original_dir", sm.shell.GetOriginalDirectory(),
			"command", command)
		
		// Reset to original directory
		if resetErr := sm.shell.ResetToOriginalDirectory(ctx); resetErr != nil {
			sm.logger.Error("failed to reset shell to original directory", "error", resetErr)
			// Close and recreate the shell
			sm.Close()
			newShell, err := NewPersistentShell(sm.logger)
			if err != nil {
				return nil, fmt.Errorf("failed to recreate shell after reset failure: %w", err)
			}
			sm.shell = newShell
		}
		
		// Return error indicating the command failed due to security violation
		return &ShellResult{
			Output:      "",
			Error:       fmt.Sprintf("Security violation: command attempted to navigate outside project directory (%s)", sm.shell.GetOriginalDirectory()),
			ExitCode:    1,
			WorkingDir:  sm.shell.GetOriginalDirectory(),
			CommandLine: command,
		}, fmt.Errorf("security violation: cannot navigate outside project directory")
	}

	return result, nil
}

// GetCurrentDirectory returns the cached current working directory
func (sm *SingleShellManager) GetCurrentDirectory() string {
	if sm.shell == nil {
		return ""
	}
	return sm.shell.GetCurrentDirectory()
}

// GetWorkingDirectory queries the shell for its actual current directory
func (sm *SingleShellManager) GetWorkingDirectory(ctx context.Context) (string, error) {
	if sm.shell == nil {
		return "", fmt.Errorf("no shell available")
	}
	return sm.shell.GetWorkingDirectory(ctx)
}

// UpdateWorkingDirectory forces an update of the cached working directory
func (sm *SingleShellManager) UpdateWorkingDirectory(ctx context.Context) error {
	if sm.shell == nil {
		return fmt.Errorf("no shell available")
	}
	return sm.shell.UpdateWorkingDirectory(ctx)
}

// GetShellInfo returns information about the shell session
func (sm *SingleShellManager) GetShellInfo() map[string]interface{} {
	if sm.shell == nil {
		return map[string]interface{}{
			"exists": false,
		}
	}

	return map[string]interface{}{
		"exists":             true,
		"session_id":         sm.shell.GetSessionID(),
		"current_directory":  sm.shell.GetCurrentDirectory(),
		"original_directory": sm.shell.GetOriginalDirectory(),
		"within_original":    sm.shell.IsPathWithinOriginal(),
	}
}

// Close closes the persistent shell
func (sm *SingleShellManager) Close() error {
	if sm.shell != nil {
		return sm.shell.Close()
	}
	return nil
}

// GetShell returns the underlying persistent shell (for direct access if needed)
func (sm *SingleShellManager) GetShell() *PersistentShell {
	return sm.shell
}