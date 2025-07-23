package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/elee1766/gofer/src/config"
	"github.com/lmittmann/tint"
)

// createTUILogger creates a logger that doesn't interfere with the TUI
// by writing to a file instead of stdout/stderr
func createTUILogger(logLevel string) *slog.Logger {
	// Create log directory
	storagePaths := config.GetDefaultStoragePaths()
	logDir := filepath.Join(filepath.Dir(storagePaths.DatabasePath), "logs")
	
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create log directory, use discard logger
		return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError, // Only show errors
		}))
	}
	
	// Create log file
	logFile := filepath.Join(logDir, "gofer.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open log file, use discard logger
		return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	}
	
	// Parse log level
	level := parseLogLevel(logLevel)
	
	// Create file-based logger
	return slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	}))
}

// createCLILogger creates a logger for CLI commands that can write to stdout/stderr
func createCLILogger(logLevel string) *slog.Logger {
	level := parseLogLevel(logLevel)
	
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: level,
	}))
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}