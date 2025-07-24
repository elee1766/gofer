package toolsutil

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"unicode/utf8"

)

// Package-level logger for tools
var logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
	Level: slog.LevelError, // Default to only showing errors
}))

// SetLogger allows setting a custom logger for the tools package
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}

// Custom error types for better error handling
var (
	ErrUnsafePath       = errors.New("unsafe path")
	ErrFileTooLarge     = errors.New("file too large")
	ErrNotTextFile      = errors.New("not a text file")
	ErrContentNotFound  = errors.New("content not found")
	ErrCommandDangerous = errors.New("command not allowed")
	ErrInvalidParams    = errors.New("invalid parameters")
)

// ToolError represents an error with additional context
type ToolError struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ToolError) Unwrap() error {
	return e.Cause
}

// NewToolError creates a new tool error with context
func NewToolError(errorType, message, code string, cause error) *ToolError {
	return &ToolError{
		Type:    errorType,
		Message: message,
		Code:    code,
		Cause:   cause,
		Details: make(map[string]interface{}),
	}
}

// DetectLanguage detects the programming language of a file based on its extension and content
func DetectLanguage(filePath string, content []byte) string {
	// First try by extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "c"
	case ".rs":
		return "rust"
	case ".php":
		return "php"
	case ".sh", ".bash":
		return "bash"
	case ".ps1":
		return "powershell"
	case ".sql":
		return "sql"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".md":
		return "markdown"
	case ".tex":
		return "latex"
	case ".dockerfile":
		return "dockerfile"
	case ".makefile":
		return "makefile"
	}

	// Check by filename
	fileName := strings.ToLower(filepath.Base(filePath))
	switch fileName {
	case "dockerfile":
		return "dockerfile"
	case "makefile":
		return "makefile"
	case "rakefile":
		return "ruby"
	case "gemfile", "gemfile.lock":
		return "ruby"
	case "package.json", "package-lock.json":
		return "json"
	case "cargo.toml", "cargo.lock":
		return "toml"
	case "go.mod", "go.sum":
		return "go"
	case "requirements.txt", "pyproject.toml":
		return "text"
	}

	// Try to detect by content for files without clear extensions
	if content != nil && len(content) > 0 {
		contentStr := string(content[:min(len(content), 1024)]) // Check first 1KB
		contentStr = strings.ToLower(contentStr)

		// Look for shebangs
		if strings.HasPrefix(contentStr, "#!/bin/bash") || strings.HasPrefix(contentStr, "#!/bin/sh") {
			return "bash"
		}
		if strings.HasPrefix(contentStr, "#!/usr/bin/env python") {
			return "python"
		}
		if strings.HasPrefix(contentStr, "#!/usr/bin/env node") {
			return "javascript"
		}

		// Look for common patterns
		if strings.Contains(contentStr, "package main") && strings.Contains(contentStr, "func ") {
			return "go"
		}
		if strings.Contains(contentStr, "def ") && (strings.Contains(contentStr, "import ") || strings.Contains(contentStr, "from ")) {
			return "python"
		}
		if strings.Contains(contentStr, "function ") || strings.Contains(contentStr, "const ") || strings.Contains(contentStr, "let ") {
			return "javascript"
		}
	}

	return "text"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsPathSafe checks if a path is safe for file operations
func IsPathSafe(path string) bool {
	// Convert to clean path
	cleanPath := filepath.Clean(path)
	
	// Check for dangerous paths
	dangerousPaths := []string{
		"/etc",
		"/bin",
		"/sbin",
		"/usr/bin",
		"/usr/sbin",
		"/boot",
		"/sys",
		"/proc",
		"/dev",
		"/root",
		"/var/log",
		"/var/lib",
		"/var/run",
		"/lib",
		"/lib64",
		"/usr/lib",
		"/usr/lib64",
	}
	
	for _, dangerous := range dangerousPaths {
		if cleanPath == dangerous || strings.HasPrefix(cleanPath, dangerous+"/") {
			return false
		}
	}
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "../") || strings.Contains(cleanPath, "..\\") {
		return false
	}
	
	// Check for null bytes
	if strings.Contains(cleanPath, "\x00") {
		return false
	}
	
	return true
}

// ValidateFileSize checks if file size is within limits
func ValidateFileSize(size int64) error {
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if size > maxFileSize {
		return fmt.Errorf("%w: file size %s exceeds maximum %s", ErrFileTooLarge, FormatBytes(size), FormatBytes(maxFileSize))
	}
	return nil
}

// IsTextFile checks if content appears to be text
func IsTextFile(content []byte) bool {
	if len(content) == 0 {
		return true // Empty file is considered text
	}
	
	// Check for null bytes (binary files often contain them)
	for i := 0; i < len(content) && i < 8192; i++ {
		if content[i] == 0 {
			return false
		}
	}
	
	// Check if content is valid UTF-8
	if !utf8.Valid(content) {
		return false
	}
	
	// Count printable vs non-printable characters in first 8KB
	sampleSize := len(content)
	if sampleSize > 8192 {
		sampleSize = 8192
	}
	
	printable := 0
	for _, b := range content[:sampleSize] {
		if b >= 32 && b <= 126 || b == '\t' || b == '\n' || b == '\r' {
			printable++
		}
	}
	
	// If more than 70% of characters are printable, consider it text
	ratio := float64(printable) / float64(sampleSize)
	return ratio > 0.70
}

// FormatBytes formats byte count as human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetLogger returns the package logger
func GetLogger() *slog.Logger {
	return logger
}