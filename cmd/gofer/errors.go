package main

import (
	"fmt"
	"log/slog"
	"os"
)

// Exit codes following standard conventions
const (
	ExitSuccess     = 0  // Success
	ExitError       = 1  // General error
	ExitUsage       = 2  // Usage error
	ExitConfig      = 3  // Configuration error
	ExitConfigError = 3  // Alias for ExitConfig
	ExitAuth        = 4  // Authentication error
	ExitPermission  = 5  // Permission error
	ExitNetwork     = 6  // Network error
	ExitTimeout     = 7  // Timeout error
	ExitInterrupted = 8  // Interrupted by user
	ExitInternal    = 9  // Internal error
)

// ErrorHandler handles different types of errors and exits with appropriate codes
type ErrorHandler struct {
	logger *slog.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *slog.Logger) *ErrorHandler {
	return &ErrorHandler{logger: logger}
}

// HandleError handles an error and exits with the appropriate code
func (h *ErrorHandler) HandleError(err error) {
	if err == nil {
		return
	}

	// Log the error
	h.logger.Error("Command failed", "error", err)

	// Determine exit code based on error type
	exitCode := h.getExitCode(err)
	
	// Print user-friendly error message
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	
	os.Exit(exitCode)
}

// getExitCode determines the appropriate exit code for an error
func (h *ErrorHandler) getExitCode(err error) int {
	errStr := err.Error()
	
	switch {
	case contains(errStr, "configuration"):
		return ExitConfig
	case contains(errStr, "API key"):
		return ExitAuth
	case contains(errStr, "permission"):
		return ExitPermission
	case contains(errStr, "network"), contains(errStr, "connection"):
		return ExitNetwork
	case contains(errStr, "timeout"):
		return ExitTimeout
	case contains(errStr, "interrupted"):
		return ExitInterrupted
	case contains(errStr, "usage"), contains(errStr, "invalid"):
		return ExitUsage
	default:
		return ExitError
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && 
		   (str == substr || 
		    (len(str) > len(substr) && 
		     stringContains(str, substr)))
}

// stringContains is a simple string contains check
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FatalError logs a fatal error and exits
func FatalError(logger *slog.Logger, err error) {
	handler := NewErrorHandler(logger)
	handler.HandleError(err)
}

// FatalErrorf logs a fatal error with formatting and exits
func FatalErrorf(logger *slog.Logger, format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	FatalError(logger, err)
}