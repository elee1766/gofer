package orclient

import (
	"log/slog"
	"time"
)

// Config holds configuration for the OpenRouter client
type Config struct {
	APIKey    string        // OpenRouter API key
	BaseURL   string        // Base URL for OpenRouter API
	Logger    *slog.Logger  // Logger for debugging
	Timeout   time.Duration // HTTP timeout
	RetryCount int          // Number of retries for failed requests
	RetryDelay time.Duration // Delay between retries
	SiteURL   string        // Site URL for ranking
	SiteName  string        // Site name for ranking
}