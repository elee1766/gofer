package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/elee1766/gofer/src/orclient"
	"github.com/elee1766/gofer/src/storage"
)

// App represents the main application with all services
type App struct {
	ModelProvider *orclient.Client
	Store         *storage.DB
	ProjectDir    string
	Logger        *slog.Logger
	Config        *AppConfig
}

// AppConfig holds configuration for creating a new App instance
type AppConfig struct {
	APIKey       string
	BaseURL      string
	Model        string
	SystemPrompt string
	EnableTools  bool
	Logger       *slog.Logger
	ProjectDir   string
}

// New creates a new App instance with all services initialized
func New(ctx context.Context, cfg AppConfig) (*App, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// Initialize storage
	projectDir := cfg.ProjectDir
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	storageDir := filepath.Join(projectDir, ".gofer")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	storagePath := filepath.Join(storageDir, "sqlite.db")
	store, err := storage.Open(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}

	// Initialize AI client with OpenRouter provider
	provider := orclient.NewClient(orclient.Config{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Logger:  logger,
	})

	return &App{
		ModelProvider: provider,
		Store:         store,
		ProjectDir:    projectDir,
		Logger:        logger,
		Config:        &cfg,
	}, nil
}

// InitializeAgentAppWithTools initializes the app and agent/tools consistently
func InitializeAgentAppWithTools(ctx context.Context, cfg AppConfig) (*App, error) {
	appInstance, err := New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return appInstance, nil
}

// Close closes all resources held by the app
func (a *App) Close() error {
	if a.Store != nil {
		return a.Store.Close()
	}
	return nil
}
