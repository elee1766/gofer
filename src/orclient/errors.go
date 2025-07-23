package orclient

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Common error variables
var (
	// ErrInvalidModel indicates an invalid model was specified
	ErrInvalidModel = errors.New("invalid model specified")
	
	// ErrNoAPIKey indicates the API key is missing
	ErrNoAPIKey = errors.New("API key is required")
	
	// ErrConversationNotFound indicates the conversation doesn't exist
	ErrConversationNotFound = errors.New("conversation not found")
	
	// ErrMaxTurnsReached indicates the conversation has reached its turn limit
	ErrMaxTurnsReached = errors.New("maximum conversation turns reached")
	
	// ErrEmptyResponse indicates the API returned an empty response
	ErrEmptyResponse = errors.New("empty response from API")
	
	// ErrStreamClosed indicates the stream has been closed
	ErrStreamClosed = errors.New("stream closed")
	
	// ErrTimeout indicates a timeout occurred
	ErrTimeout = errors.New("operation timed out")
	
	// ErrRateLimited indicates rate limiting
	ErrRateLimited = errors.New("rate limited")
	
	// ErrInsufficientQuota indicates the account has insufficient quota
	ErrInsufficientQuota = errors.New("insufficient quota")
)

// ErrorResponse represents a standard error response from the API
// This matches the OpenRouter error format: {"error":{"message":"...","code":"..."}}
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError represents an error response from the OpenRouter API.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	Code       string
	Param      string
	Details    map[string]interface{}
	RequestID  string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// IsRetryable returns true if the error is retryable.
func (e *APIError) IsRetryable() bool {
	// 5xx errors are generally retryable
	if e.StatusCode >= 500 && e.StatusCode < 600 {
		return true
	}
	
	// Rate limit errors are retryable after a delay
	if e.StatusCode == http.StatusTooManyRequests {
		return true
	}
	
	// Specific error codes that are retryable
	switch e.Code {
	case "timeout", "connection_error", "server_error":
		return true
	}
	
	return false
}

// IsRateLimit returns true if this is a rate limit error.
func (e *APIError) IsRateLimit() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.Code == "rate_limit_exceeded"
}

// IsAuthError returns true if this is an authentication error.
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == http.StatusUnauthorized || e.Code == "invalid_api_key"
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// MultiError represents multiple errors.
type MultiError struct {
	Errors []error
}

// Error implements the error interface.
func (e *MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors occurred (%d errors)", len(e.Errors))
}

// Add adds an error to the multi-error.
func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors.
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Unwrap returns the errors slice.
func (e *MultiError) Unwrap() []error {
	return e.Errors
}

// RetryableError wraps an error with retry information.
type RetryableError struct {
	Err         error
	RetryAfter  time.Duration
	AttemptNum  int
	MaxAttempts int
}

// Error implements the error interface.
func (e *RetryableError) Error() string {
	return fmt.Sprintf("attempt %d/%d failed: %v (retry after %v)",
		e.AttemptNum, e.MaxAttempts, e.Err, e.RetryAfter)
}

// Unwrap returns the underlying error.
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// ShouldRetry returns true if the operation should be retried.
func (e *RetryableError) ShouldRetry() bool {
	return e.AttemptNum < e.MaxAttempts
}

// TimeoutError represents a timeout error with context.
type TimeoutError struct {
	Operation string
	Duration  time.Duration
	Cause     error
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s timed out after %v: %v", e.Operation, e.Duration, e.Cause)
	}
	return fmt.Sprintf("%s timed out after %v", e.Operation, e.Duration)
}

// Unwrap returns the underlying error.
func (e *TimeoutError) Unwrap() error {
	return e.Cause
}

// Is implements error matching.
func (e *TimeoutError) Is(target error) bool {
	return target == ErrTimeout
}

// StorageError represents a storage-related error.
type StorageError struct {
	Operation string
	Path      string
	Err       error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("storage error during %s on %s: %v", e.Operation, e.Path, e.Err)
	}
	return fmt.Sprintf("storage error during %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error.
func (e *StorageError) Unwrap() error {
	return e.Err
}

// ToolError represents a tool execution error.
type ToolError struct {
	ToolName  string
	CallID    string
	Err       error
	StartTime time.Time
	EndTime   time.Time
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	duration := e.EndTime.Sub(e.StartTime)
	return fmt.Sprintf("tool '%s' (call %s) failed after %v: %v",
		e.ToolName, e.CallID, duration, e.Err)
}

// Unwrap returns the underlying error.
func (e *ToolError) Unwrap() error {
	return e.Err
}

// Duration returns how long the tool execution took.
func (e *ToolError) Duration() time.Duration {
	return e.EndTime.Sub(e.StartTime)
}

// ErrorHandler provides centralized error handling with logging.
type ErrorHandler struct {
	logger *slog.Logger
}

// NewErrorHandler creates a new error handler.
func NewErrorHandler(logger *slog.Logger) *ErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ErrorHandler{
		logger: logger.With("component", "error_handler"),
	}
}

// Handle logs and potentially transforms an error.
func (eh *ErrorHandler) Handle(err error, operation string, attrs ...slog.Attr) error {
	if err == nil {
		return nil
	}
	
	// Build attributes
	logAttrs := []any{"operation", operation, "error", err.Error()}
	for _, attr := range attrs {
		logAttrs = append(logAttrs, attr.Key, attr.Value)
	}
	
	// Log based on error type
	switch e := err.(type) {
	case *APIError:
		if e.IsRateLimit() {
			eh.logger.Warn("rate limited", logAttrs...)
		} else if e.IsAuthError() {
			eh.logger.Error("authentication failed", logAttrs...)
		} else if e.IsRetryable() {
			eh.logger.Warn("retryable API error", logAttrs...)
		} else {
			eh.logger.Error("API error", logAttrs...)
		}
		
	case *ValidationError:
		eh.logger.Warn("validation error", logAttrs...)
		
	case *TimeoutError:
		eh.logger.Error("timeout error", logAttrs...)
		
	case *StorageError:
		eh.logger.Error("storage error", logAttrs...)
		
	case *ToolError:
		eh.logger.Error("tool execution error", logAttrs...)
		
	case *MultiError:
		eh.logger.Error("multiple errors", append(logAttrs, "error_count", len(e.Errors))...)
		
	default:
		eh.logger.Error("error occurred", logAttrs...)
	}
	
	return err
}

// Wrap wraps an error with additional context.
func (eh *ErrorHandler) Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for specific error types
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}
	
	var retryErr *RetryableError
	if errors.As(err, &retryErr) {
		return retryErr.ShouldRetry()
	}
	
	// Check for timeout errors
	if errors.Is(err, ErrTimeout) {
		return true
	}
	
	// Check for rate limiting
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	
	return false
}

// GetRetryDelay returns the appropriate retry delay for an error.
func GetRetryDelay(err error, attempt int) time.Duration {
	// Check for rate limit errors with specific retry-after
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.IsRateLimit() {
		// Look for Retry-After header value in details
		if retryAfter, ok := apiErr.Details["retry_after"].(float64); ok {
			return time.Duration(retryAfter) * time.Second
		}
	}
	
	// Linear backoff based on attempt number
	// attempt 1: 1s, attempt 2: 2s, attempt 3: 4s, etc.
	delay := time.Second * time.Duration(1<<uint(attempt-1))
	maxDelay := time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}