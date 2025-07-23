package orclient

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAPIError(t *testing.T) {
	tests := []struct {
		name         string
		err          *APIError
		expectedMsg  string
		isRetryable  bool
		isRateLimit  bool
		isAuthError  bool
	}{
		{
			name: "basic error",
			err: &APIError{
				StatusCode: 400,
				Message:    "Bad request",
			},
			expectedMsg: "API error 400: Bad request",
			isRetryable: false,
			isRateLimit: false,
			isAuthError: false,
		},
		{
			name: "error with code",
			err: &APIError{
				StatusCode: 403,
				Message:    "Forbidden",
				Code:       "insufficient_permissions",
			},
			expectedMsg: "API error 403 (insufficient_permissions): Forbidden",
			isRetryable: false,
			isRateLimit: false,
			isAuthError: false,
		},
		{
			name: "server error",
			err: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
			},
			expectedMsg: "API error 500: Internal server error",
			isRetryable: true,
			isRateLimit: false,
			isAuthError: false,
		},
		{
			name: "rate limit error",
			err: &APIError{
				StatusCode: 429,
				Message:    "Too many requests",
				Code:       "rate_limit_exceeded",
			},
			expectedMsg: "API error 429 (rate_limit_exceeded): Too many requests",
			isRetryable: true,
			isRateLimit: true,
			isAuthError: false,
		},
		{
			name: "auth error",
			err: &APIError{
				StatusCode: 401,
				Message:    "Invalid API key",
				Code:       "invalid_api_key",
			},
			expectedMsg: "API error 401 (invalid_api_key): Invalid API key",
			isRetryable: false,
			isRateLimit: false,
			isAuthError: true,
		},
		{
			name: "timeout error",
			err: &APIError{
				StatusCode: 504,
				Message:    "Gateway timeout",
				Code:       "timeout",
			},
			expectedMsg: "API error 504 (timeout): Gateway timeout",
			isRetryable: true,
			isRateLimit: false,
			isAuthError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", tt.err.Error(), tt.expectedMsg)
			}
			if tt.err.IsRetryable() != tt.isRetryable {
				t.Errorf("IsRetryable() = %v, want %v", tt.err.IsRetryable(), tt.isRetryable)
			}
			if tt.err.IsRateLimit() != tt.isRateLimit {
				t.Errorf("IsRateLimit() = %v, want %v", tt.err.IsRateLimit(), tt.isRateLimit)
			}
			if tt.err.IsAuthError() != tt.isAuthError {
				t.Errorf("IsAuthError() = %v, want %v", tt.err.IsAuthError(), tt.isAuthError)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name        string
		err         *ValidationError
		expectedMsg string
	}{
		{
			name: "with field",
			err: &ValidationError{
				Field:   "email",
				Message: "invalid email format",
			},
			expectedMsg: "validation error on field 'email': invalid email format",
		},
		{
			name: "without field",
			err: &ValidationError{
				Message: "request body is required",
			},
			expectedMsg: "validation error: request body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", tt.err.Error(), tt.expectedMsg)
			}
		})
	}
}

func TestMultiError(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		me := &MultiError{}
		me.Add(errors.New("first error"))

		if me.Error() != "first error" {
			t.Errorf("Error() = %v, want %v", me.Error(), "first error")
		}
		if !me.HasErrors() {
			t.Error("HasErrors() = false, want true")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		me := &MultiError{}
		me.Add(errors.New("error 1"))
		me.Add(errors.New("error 2"))
		me.Add(nil) // Should be ignored

		if !strings.Contains(me.Error(), "2 errors") {
			t.Errorf("Error() = %v, want message containing '2 errors'", me.Error())
		}
		if len(me.Unwrap()) != 2 {
			t.Errorf("Unwrap() returned %d errors, want 2", len(me.Unwrap()))
		}
	})

	t.Run("no errors", func(t *testing.T) {
		me := &MultiError{}
		if me.HasErrors() {
			t.Error("HasErrors() = true, want false")
		}
	})
}

func TestRetryableError(t *testing.T) {
	baseErr := errors.New("connection failed")
	
	t.Run("should retry", func(t *testing.T) {
		err := &RetryableError{
			Err:         baseErr,
			RetryAfter:  time.Second,
			AttemptNum:  2,
			MaxAttempts: 3,
		}

		if !strings.Contains(err.Error(), "attempt 2/3") {
			t.Errorf("Error() = %v, want message containing 'attempt 2/3'", err.Error())
		}
		if !err.ShouldRetry() {
			t.Error("ShouldRetry() = false, want true")
		}
		if err.Unwrap() != baseErr {
			t.Error("Unwrap() returned wrong error")
		}
	})

	t.Run("should not retry", func(t *testing.T) {
		err := &RetryableError{
			Err:         baseErr,
			RetryAfter:  time.Second,
			AttemptNum:  3,
			MaxAttempts: 3,
		}

		if err.ShouldRetry() {
			t.Error("ShouldRetry() = true, want false")
		}
	})
}

func TestTimeoutError(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("context deadline exceeded")
		err := &TimeoutError{
			Operation: "API call",
			Duration:  5 * time.Second,
			Cause:     cause,
		}

		expectedMsg := "API call timed out after 5s: context deadline exceeded"
		if err.Error() != expectedMsg {
			t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
		}
		if err.Unwrap() != cause {
			t.Error("Unwrap() returned wrong error")
		}
		if !errors.Is(err, ErrTimeout) {
			t.Error("errors.Is(err, ErrTimeout) = false, want true")
		}
	})

	t.Run("without cause", func(t *testing.T) {
		err := &TimeoutError{
			Operation: "request",
			Duration:  10 * time.Second,
		}

		expectedMsg := "request timed out after 10s"
		if err.Error() != expectedMsg {
			t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
		}
	})
}

func TestStorageError(t *testing.T) {
	baseErr := errors.New("disk full")

	t.Run("with path", func(t *testing.T) {
		err := &StorageError{
			Operation: "write",
			Path:      "/data/conversations.db",
			Err:       baseErr,
		}

		expectedMsg := "storage error during write on /data/conversations.db: disk full"
		if err.Error() != expectedMsg {
			t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
		}
	})

	t.Run("without path", func(t *testing.T) {
		err := &StorageError{
			Operation: "migrate",
			Err:       baseErr,
		}

		expectedMsg := "storage error during migrate: disk full"
		if err.Error() != expectedMsg {
			t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
		}
	})
}

func TestToolError(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)
	
	err := &ToolError{
		ToolName:  "calculator",
		CallID:    "call-123",
		Err:       errors.New("division by zero"),
		StartTime: startTime,
		EndTime:   endTime,
	}

	if !strings.Contains(err.Error(), "calculator") {
		t.Errorf("Error() should contain tool name")
	}
	if !strings.Contains(err.Error(), "call-123") {
		t.Errorf("Error() should contain call ID")
	}
	if !strings.Contains(err.Error(), "100ms") {
		t.Errorf("Error() should contain duration")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("Error() should contain underlying error")
	}

	duration := err.Duration()
	if duration != 100*time.Millisecond {
		t.Errorf("Duration() = %v, want %v", duration, 100*time.Millisecond)
	}
}

func TestErrorHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	eh := NewErrorHandler(logger)

	t.Run("handle nil error", func(t *testing.T) {
		err := eh.Handle(nil, "test operation")
		if err != nil {
			t.Errorf("Handle(nil) = %v, want nil", err)
		}
	})

	t.Run("handle API error", func(t *testing.T) {
		buf.Reset()
		apiErr := &APIError{
			StatusCode: 429,
			Message:    "Rate limited",
			Code:       "rate_limit_exceeded",
		}
		
		err := eh.Handle(apiErr, "send message")
		if err != apiErr {
			t.Error("Handle should return the same error")
		}
		
		logOutput := buf.String()
		if !strings.Contains(logOutput, "rate limited") {
			t.Error("Log should contain 'rate limited'")
		}
		if !strings.Contains(logOutput, "send message") {
			t.Error("Log should contain operation name")
		}
	})

	t.Run("wrap error", func(t *testing.T) {
		baseErr := errors.New("base error")
		wrapped := eh.Wrap(baseErr, "failed to process")
		
		if !strings.Contains(wrapped.Error(), "failed to process: base error") {
			t.Errorf("Wrapped error = %v", wrapped)
		}
		if !errors.Is(wrapped, baseErr) {
			t.Error("Wrapped error should contain base error")
		}
	})
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "retryable API error",
			err: &APIError{
				StatusCode: 500,
			},
			want: true,
		},
		{
			name: "non-retryable API error",
			err: &APIError{
				StatusCode: 400,
			},
			want: false,
		},
		{
			name: "retryable error",
			err: &RetryableError{
				Err:         errors.New("test"),
				AttemptNum:  1,
				MaxAttempts: 3,
			},
			want: true,
		},
		{
			name: "timeout error",
			err:  ErrTimeout,
			want: true,
		},
		{
			name: "rate limited error",
			err:  ErrRateLimited,
			want: true,
		},
		{
			name: "wrapped timeout error",
			err:  fmt.Errorf("operation failed: %w", ErrTimeout),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		attempt  int
		wantMin  time.Duration
		wantMax  time.Duration
	}{
		{
			name:    "first attempt",
			err:     errors.New("error"),
			attempt: 1,
			wantMin: 1 * time.Second,
			wantMax: 1 * time.Second,
		},
		{
			name:    "second attempt",
			err:     errors.New("error"),
			attempt: 2,
			wantMin: 2 * time.Second,
			wantMax: 2 * time.Second,
		},
		{
			name:    "third attempt",
			err:     errors.New("error"),
			attempt: 3,
			wantMin: 4 * time.Second,
			wantMax: 4 * time.Second,
		},
		{
			name: "rate limit with retry-after",
			err: &APIError{
				StatusCode: http.StatusTooManyRequests,
				Details: map[string]interface{}{
					"retry_after": float64(5),
				},
			},
			attempt: 1,
			wantMin: 5 * time.Second,
			wantMax: 5 * time.Second,
		},
		{
			name:    "very high attempt",
			err:     errors.New("error"),
			attempt: 10,
			wantMin: time.Minute,
			wantMax: time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := GetRetryDelay(tt.err, tt.attempt)
			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("GetRetryDelay() = %v, want between %v and %v", delay, tt.wantMin, tt.wantMax)
			}
		})
	}
}