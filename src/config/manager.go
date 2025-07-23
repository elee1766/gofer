package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager manages configuration loading, validation, and access
type Manager struct {
	config           *Config
	loader           *Loader
	validator        *Validator
	permissionChecker *PermissionChecker
	configPath       string
	mu               sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	precedence := GetConfigPaths()
	loader := NewLoader(precedence)
	
	// Load configuration
	config, err := loader.Load()
	if err != nil {
		// If no config found, use defaults
		if os.IsNotExist(err) {
			config = DefaultConfig()
		} else {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	// Find which config file was actually loaded
	configPath, _ := FindConfigFile()

	return &Manager{
		config:           config,
		loader:           loader,
		validator:        NewValidator(),
		permissionChecker: NewPermissionChecker(&config.Permissions),
		configPath:       configPath,
	}, nil
}

// NewManagerWithConfig creates a manager with a specific configuration
func NewManagerWithConfig(config *Config) (*Manager, error) {
	validator := NewValidator()
	if err := validator.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Manager{
		config:           config,
		loader:           NewLoader(GetConfigPaths()),
		validator:        validator,
		permissionChecker: NewPermissionChecker(&config.Permissions),
	}, nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetAPIConfig returns the API configuration
func (m *Manager) GetAPIConfig() APIConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.API
}

// GetAgentConfig returns the agent configuration
func (m *Manager) GetAgentConfig() AgentConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Agent
}

// GetPermissionChecker returns the permission checker
func (m *Manager) GetPermissionChecker() *PermissionChecker {
	return m.permissionChecker
}

// Reload reloads the configuration from disk
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, err := m.loader.Load()
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	m.config = config
	m.permissionChecker = NewPermissionChecker(&config.Permissions)
	return nil
}

// Save saves the current configuration to disk
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.configPath == "" {
		return fmt.Errorf("no configuration file path set")
	}

	return m.loader.SaveFile(m.config, m.configPath)
}

// SaveTo saves the configuration to a specific path
func (m *Manager) SaveTo(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loader.SaveFile(m.config, path)
}

// Update updates the configuration with new values
func (m *Manager) Update(updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert updates to JSON and back to apply to config
	updateJSON, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal updates: %w", err)
	}

	var partialConfig Config
	if err := json.Unmarshal(updateJSON, &partialConfig); err != nil {
		return fmt.Errorf("failed to unmarshal updates: %w", err)
	}

	// Merge with current config
	m.config = m.loader.mergeConfigs(m.config, &partialConfig)

	// Validate the new configuration
	if err := m.validator.Validate(m.config); err != nil {
		return fmt.Errorf("invalid configuration after update: %w", err)
	}

	// Update permission checker
	m.permissionChecker = NewPermissionChecker(&m.config.Permissions)

	return nil
}

// CheckToolPermission checks if a tool operation is allowed
func (m *Manager) CheckToolPermission(toolName string, args map[string]interface{}) (PermissionResult, error) {
	return m.permissionChecker.CheckToolPermission(toolName, args)
}

// CheckFileReadPermission checks if reading a file is allowed
func (m *Manager) CheckFileReadPermission(path string) (PermissionResult, error) {
	return m.permissionChecker.CheckFileReadPermission(path)
}

// CheckFileWritePermission checks if writing a file is allowed
func (m *Manager) CheckFileWritePermission(path string) (PermissionResult, error) {
	return m.permissionChecker.CheckFileWritePermission(path)
}

// CheckCommandPermission checks if executing a command is allowed
func (m *Manager) CheckCommandPermission(command string, args []string) (PermissionResult, error) {
	return m.permissionChecker.CheckCommandPermission(command, args)
}

// CheckNetworkPermission checks if a network request is allowed
func (m *Manager) CheckNetworkPermission(url string) (PermissionResult, error) {
	return m.permissionChecker.CheckNetworkPermission(url)
}

// GetConfigPath returns the path of the loaded configuration file
func (m *Manager) GetConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configPath
}

// SetConfigPath sets the configuration file path
func (m *Manager) SetConfigPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configPath = path
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validator.Validate(m.config)
}

// GetInfo returns configuration information
func (m *Manager) GetInfo() (*ConfigInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := &ConfigInfo{
		ActiveConfig: m.configPath,
		Provider:     m.config.API.Provider,
		Model:        m.config.Agent.Model,
		Errors:       []string{},
		Warnings:     []string{},
	}

	// Check for API key
	if m.config.API.APIKey == "" && m.config.API.APIKeyEnvVar != "" {
		if os.Getenv(m.config.API.APIKeyEnvVar) == "" {
			info.Warnings = append(info.Warnings, fmt.Sprintf("API key environment variable %s is not set", m.config.API.APIKeyEnvVar))
		}
	}

	// Validate configuration
	if err := m.validator.Validate(m.config); err != nil {
		info.Errors = append(info.Errors, fmt.Sprintf("Configuration validation error: %v", err))
	}

	return info, nil
}

// ExportConfig exports the configuration as JSON
func (m *Manager) ExportConfig(includeSecrets bool) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := *m.config

	// Optionally remove sensitive information
	if !includeSecrets {
		config.API.APIKey = ""
		// Remove other sensitive fields as needed
	}

	return json.MarshalIndent(config, "", "  ")
}

// ImportConfig imports configuration from JSON
func (m *Manager) ImportConfig(data []byte) error {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate the imported configuration
	if err := m.validator.Validate(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = &config
	m.permissionChecker = NewPermissionChecker(&config.Permissions)

	return nil
}

// Global configuration manager instance
var globalManager *Manager
var globalOnce sync.Once

// InitGlobalManager initializes the global configuration manager
func InitGlobalManager() error {
	var err error
	globalOnce.Do(func() {
		globalManager, err = NewManager()
	})
	return err
}

// Get returns the global configuration
func Get() *Config {
	if globalManager == nil {
		// Initialize with default config if not set
		InitGlobalManager()
	}
	if globalManager == nil {
		return DefaultConfig()
	}
	return globalManager.GetConfig()
}

// WorkingDirectory returns the current working directory or project root
func WorkingDirectory() string {
	config := Get()
	if config.Project.Root != "" {
		return config.Project.Root
	}
	
	// Fall back to current working directory
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	
	// Last resort
	return "."
}

// ShouldShowInitDialog checks if the init dialog should be shown
func ShouldShowInitDialog() (bool, error) {
	// Check if there's a config file in current directory or parents
	_, err := FindConfigFile()
	if err != nil {
		// No config file found, should show init dialog
		return true, nil
	}
	
	// Config exists, check if it's properly initialized
	config := Get()
	if config.API.Provider == "" || (config.API.APIKey == "" && config.API.APIKeyEnvVar == "") {
		return true, nil
	}
	
	return false, nil
}

// MarkProjectInitialized marks the project as initialized by ensuring config exists
func MarkProjectInitialized() error {
	_, err := FindConfigFile()
	if err != nil {
		// Create a basic config file in current directory
		configPath := filepath.Join(".", ".gofer.json")
		config := DefaultConfig()
		return saveConfigToFile(config, configPath)
	}
	return nil
}

// saveConfigToFile saves config to a file
func saveConfigToFile(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return os.WriteFile(path, data, 0644)
}

// UpdateTheme updates the theme configuration (stub implementation)
func UpdateTheme(theme string) error {
	// TODO: Implement theme updating
	return nil
}