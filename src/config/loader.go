package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
)

// Loader handles loading and merging configurations from multiple sources
type Loader struct {
	precedence ConfigPrecedence
	validator  *Validator
}

// NewLoader creates a new configuration loader
func NewLoader(precedence ConfigPrecedence) *Loader {
	return &Loader{
		precedence: precedence,
		validator:  NewValidator(),
	}
}

// Load loads configuration from all sources and merges them
func (l *Loader) Load() (*Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Load and merge configurations in order of precedence
	sources := []struct {
		path   string
		source ConfigSource
	}{
		{l.precedence.SystemConfig, SourceSystem},
		{l.precedence.UserConfig, SourceUser},
		{l.precedence.ProjectConfig, SourceProject},
		{l.precedence.LocalConfig, SourceLocal},
	}

	for _, src := range sources {
		if src.path == "" {
			continue
		}

		if cfg, err := l.loadFile(src.path); err == nil {
			config = l.mergeConfigs(config, cfg)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load %s config from %s: %w", src.source, src.path, err)
		}
	}

	// Apply environment variable overrides
	if l.precedence.EnvironmentPrefix != "" {
		l.applyEnvironmentOverrides(config)
	}

	// Validate the final configuration
	if err := l.validator.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadFile loads a single configuration file
func (l *Loader) loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

// SaveFile saves configuration to a file
func (l *Loader) SaveFile(config *Config, path string) error {
	// Validate before saving
	if err := l.validator.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal with pretty printing
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// mergeConfigs merges two configurations with the second taking precedence
func (l *Loader) mergeConfigs(base, override *Config) *Config {
	result := *base

	// Merge API config
	if override.API.Provider != "" {
		result.API.Provider = override.API.Provider
	}
	if override.API.BaseURL != "" {
		result.API.BaseURL = override.API.BaseURL
	}
	if override.API.APIKey != "" {
		result.API.APIKey = override.API.APIKey
	}
	if override.API.APIKeyEnvVar != "" {
		result.API.APIKeyEnvVar = override.API.APIKeyEnvVar
	}
	if override.API.Timeout != 0 {
		result.API.Timeout = override.API.Timeout
	}
	if len(override.API.Headers) > 0 {
		if result.API.Headers == nil {
			result.API.Headers = make(map[string]string)
		}
		for k, v := range override.API.Headers {
			result.API.Headers[k] = v
		}
	}

	// Merge Agent config
	result.Agent = l.mergeAgentConfig(result.Agent, override.Agent)

	// Merge Permissions
	result.Permissions = l.mergePermissions(result.Permissions, override.Permissions)

	// Merge Security
	result.Security = l.mergeSecurity(result.Security, override.Security)

	// Merge Preferences
	result.Preferences = l.mergePreferences(result.Preferences, override.Preferences)

	// Merge Project
	if override.Project.Root != "" {
		result.Project.Root = override.Project.Root
	}
	if override.Project.Name != "" {
		result.Project.Name = override.Project.Name
	}
	if override.Project.Type != "" {
		result.Project.Type = override.Project.Type
	}
	if len(override.Project.IgnorePatterns) > 0 {
		result.Project.IgnorePatterns = override.Project.IgnorePatterns
	}
	result.Project.UseGitIgnore = override.Project.UseGitIgnore

	// Merge Tools
	if len(override.Tools) > 0 {
		if result.Tools == nil {
			result.Tools = make(map[string]ToolConfig)
		}
		for k, v := range override.Tools {
			result.Tools[k] = v
		}
	}

	// Merge MCP Servers
	if len(override.MCPServers) > 0 {
		result.MCPServers = override.MCPServers
	}

	return &result
}

// mergeAgentConfig merges agent configurations
func (l *Loader) mergeAgentConfig(base, override AgentConfig) AgentConfig {
	result := base

	if override.Model != "" {
		result.Model = override.Model
	}
	if override.Temperature != 0 {
		result.Temperature = override.Temperature
	}
	if override.MaxTokens != 0 {
		result.MaxTokens = override.MaxTokens
	}
	if override.SystemPrompt != "" {
		result.SystemPrompt = override.SystemPrompt
	}
	if override.MaxRetries != 0 {
		result.MaxRetries = override.MaxRetries
	}
	if override.RetryDelay != 0 {
		result.RetryDelay = override.RetryDelay
	}

	return result
}

// mergePermissions merges permission configurations
func (l *Loader) mergePermissions(base, override PermissionsConfig) PermissionsConfig {
	result := base

	if override.DefaultMode != "" {
		result.DefaultMode = override.DefaultMode
	}

	// Merge Tools permissions
	if len(override.Tools.Allow) > 0 {
		result.Tools.Allow = override.Tools.Allow
	}
	if len(override.Tools.Deny) > 0 {
		result.Tools.Deny = override.Tools.Deny
	}
	if len(override.Tools.RequireConfirmation) > 0 {
		result.Tools.RequireConfirmation = override.Tools.RequireConfirmation
	}
	if len(override.Tools.CustomRules) > 0 {
		result.Tools.CustomRules = override.Tools.CustomRules
	}

	// Merge FileSystem permissions
	if len(override.FileSystem.ReadPaths) > 0 {
		result.FileSystem.ReadPaths = override.FileSystem.ReadPaths
	}
	if len(override.FileSystem.WritePaths) > 0 {
		result.FileSystem.WritePaths = override.FileSystem.WritePaths
	}
	if len(override.FileSystem.DenyPaths) > 0 {
		result.FileSystem.DenyPaths = override.FileSystem.DenyPaths
	}
	if override.FileSystem.MaxFileSize != 0 {
		result.FileSystem.MaxFileSize = override.FileSystem.MaxFileSize
	}
	result.FileSystem.SandboxMode = override.FileSystem.SandboxMode

	// Merge Commands permissions
	if len(override.Commands.AllowedCommands) > 0 {
		result.Commands.AllowedCommands = override.Commands.AllowedCommands
	}
	if len(override.Commands.DeniedCommands) > 0 {
		result.Commands.DeniedCommands = override.Commands.DeniedCommands
	}
	if override.Commands.MaxTimeout != 0 {
		result.Commands.MaxTimeout = override.Commands.MaxTimeout
	}

	// Merge Network permissions
	if len(override.Network.AllowedDomains) > 0 {
		result.Network.AllowedDomains = override.Network.AllowedDomains
	}
	if len(override.Network.DeniedDomains) > 0 {
		result.Network.DeniedDomains = override.Network.DeniedDomains
	}

	return result
}

// mergeSecurity merges security configurations
func (l *Loader) mergeSecurity(base, override SecurityConfig) SecurityConfig {
	result := base

	result.RequireConfirmation = override.RequireConfirmation
	result.ShowPreview = override.ShowPreview
	result.LogOperations = override.LogOperations

	if override.SessionTimeout != 0 {
		result.SessionTimeout = override.SessionTimeout
	}

	// Merge audit log config
	if override.AuditLog.Enabled {
		result.AuditLog = override.AuditLog
	}

	return result
}

// mergePreferences merges preference configurations
func (l *Loader) mergePreferences(base, override PreferencesConfig) PreferencesConfig {
	result := base

	// Merge Editor
	if override.Editor.Command != "" {
		result.Editor.Command = override.Editor.Command
	}
	if len(override.Editor.Args) > 0 {
		result.Editor.Args = override.Editor.Args
	}
	if override.Editor.TabSize != 0 {
		result.Editor.TabSize = override.Editor.TabSize
	}

	// Merge UI
	if override.UI.Theme != "" {
		result.UI.Theme = override.UI.Theme
	}
	result.UI.ShowLineNumbers = override.UI.ShowLineNumbers
	result.UI.SyntaxHighlighting = override.UI.SyntaxHighlighting

	// Merge Output
	if override.Output.Format != "" {
		result.Output.Format = override.Output.Format
	}
	result.Output.Verbose = override.Output.Verbose
	result.Output.Quiet = override.Output.Quiet

	return result
}

// applyEnvironmentOverrides applies environment variable overrides to config
func (l *Loader) applyEnvironmentOverrides(config *Config) {
	prefix := l.precedence.EnvironmentPrefix

	// Check for API key override
	if apiKey := os.Getenv(prefix + "_API_KEY"); apiKey != "" {
		config.API.APIKey = apiKey
	}
	// Also check OPENROUTER_API_KEY for compatibility
	if config.API.APIKey == "" {
		if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
			config.API.APIKey = apiKey
		}
	}

	// Check for model override
	if model := os.Getenv(prefix + "_MODEL"); model != "" {
		config.Agent.Model = model
	}

	// Check for provider override
	if provider := os.Getenv(prefix + "_PROVIDER"); provider != "" {
		config.API.Provider = provider
	}

	// Check for base URL override
	if baseURL := os.Getenv(prefix + "_BASE_URL"); baseURL != "" {
		config.API.BaseURL = baseURL
	}

	// Check for permission mode override
	if permMode := os.Getenv(prefix + "_PERMISSION_MODE"); permMode != "" {
		config.Permissions.DefaultMode = permMode
	}

	// Check for sandbox mode override
	if sandbox := os.Getenv(prefix + "_SANDBOX"); strings.ToLower(sandbox) == "true" {
		config.Permissions.FileSystem.SandboxMode = true
	}
}

// GetConfigPaths returns the configuration file paths to check
func GetConfigPaths() ConfigPrecedence {
	// Use XDG paths for cross-platform compatibility
	userConfigPath := filepath.Join(xdg.ConfigHome, "gofer", "config.json")
	
	// System config path varies by OS
	systemConfigPath := "/etc/gofer/config.json"
	if runtime.GOOS == "windows" {
		systemConfigPath = filepath.Join(os.Getenv("PROGRAMDATA"), "gofer", "config.json")
	}
	
	return ConfigPrecedence{
		SystemConfig:      systemConfigPath,
		UserConfig:        userConfigPath,
		ProjectConfig:     filepath.Join(".gofer", "config.json"),
		LocalConfig:       filepath.Join(".gofer", "config.local.json"),
		EnvironmentPrefix: "GOFER",
	}
}

// FindConfigFile searches for a configuration file in standard locations
func FindConfigFile() (string, error) {
	paths := GetConfigPaths()
	
	// Check in order of precedence (reversed for finding)
	checkPaths := []string{
		paths.LocalConfig,
		paths.ProjectConfig,
		paths.UserConfig,
		paths.SystemConfig,
	}

	for _, path := range checkPaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no configuration file found")
}