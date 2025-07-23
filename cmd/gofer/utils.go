package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/elee1766/gofer/src/config"
)

// loadConfig loads the configuration from the specified path or default locations
func loadConfig(path string) (*config.Config, error) {
	precedence := config.GetConfigPaths()
	if path != "" {
		// Override with specific path
		precedence.UserConfig = path
	}

	loader := config.NewLoader(precedence)
	return loader.Load()
}

// loadConfigWithoutValidation loads config without validation for fallback cases
func loadConfigWithoutValidation(path string) *config.Config {
	cfg := config.DefaultConfig()

	// Try to read the file directly
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	// Try to unmarshal, ignore errors
	_ = json.Unmarshal(data, cfg)

	return cfg
}

// overrideConfigFromCLI overrides configuration values with CLI flags
func overrideConfigFromCLI(cfg *config.Config, cli *CLI) {
	if cli.APIKey != "" {
		cfg.API.APIKey = cli.APIKey
	}
	if cli.BaseURL != "" {
		cfg.API.BaseURL = cli.BaseURL
	}
}

// getLogFilePath returns the path for the current session's log file
func getLogFilePath() (string, error) {
	// Create logs directory under XDG_STATE_HOME
	logDir := filepath.Join(xdg.StateHome, "gofer", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create session-based log filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logDir, fmt.Sprintf("session_%s.log", timestamp))

	return logFile, nil
}

// initLogger creates a basic slog logger with the specified configuration
func initLogger(level string, verbose, quiet bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(level),
	}

	if verbose {
		opts.Level = slog.LevelDebug
	}
	if quiet {
		opts.Level = slog.LevelError
	}

	// Try to create log file
	logPath, err := getLogFilePath()
	if err != nil {
		// Fallback to null handler if we can't create log file
		return slog.New(slog.NewTextHandler(io.Discard, opts))
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fallback to null handler
		return slog.New(slog.NewTextHandler(io.Discard, opts))
	}

	return slog.New(slog.NewTextHandler(logFile, opts))
}
