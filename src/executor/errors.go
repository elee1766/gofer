package executor

import "errors"

var (
	// Config validation errors
	ErrPromptTextRequired  = errors.New("prompt text is required")
	ErrModelRequired       = errors.New("model is required")
	ErrModelClientRequired = errors.New("model client is required")
	ErrDatabaseRequired    = errors.New("database is required")
	
	// Session errors
	ErrSessionNotFound = errors.New("session not found")
	
	// Execution errors
	ErrMaxTurnsExceeded = errors.New("maximum turns exceeded")
)