package config

import (
	"time"
)

// Config represents the complete configuration for gofer
type Config struct {
	// Version of the configuration format
	Version string `json:"version"`

	// API configuration
	API APIConfig `json:"api"`

	// Agent configuration (basic settings)
	Agent AgentConfig `json:"agent"`

	// Permissions configuration
	Permissions PermissionsConfig `json:"permissions"`

	// Security configuration
	Security SecurityConfig `json:"security"`

	// User preferences
	Preferences PreferencesConfig `json:"preferences"`

	// Project-specific settings
	Project ProjectConfig `json:"project"`

	// Tool-specific configurations
	Tools map[string]ToolConfig `json:"tools,omitempty"`

	// MCP server configurations
	MCPServers []MCPServerConfig `json:"mcp_servers,omitempty"`

	// Observability configuration
	Observability ObservabilityConfig `json:"observability,omitempty"`

	// DebugLSP enables debug logging for LSP
	DebugLSP bool `json:"debug_lsp,omitempty"`

	// Debug enables general debug logging
	Debug bool `json:"debug,omitempty"`

	// Agents configuration
	Agents map[string]AgentConfig `json:"agents,omitempty"`

	// Data directory configuration
	Data DataConfig `json:"data,omitempty"`

	// Providers configuration for model providers
	Providers map[string]ProviderConfig `json:"providers,omitempty"`

	// LSP configuration for Language Server Protocol
	LSP LSPConfig `json:"lsp,omitempty"`

	// AutoCompact configuration for automatic session compaction
	AutoCompact bool `json:"auto_compact,omitempty"`
}

// LSPConfig defines LSP configuration
type LSPConfig struct {
	// Enabled indicates if LSP is enabled
	Enabled bool `json:"enabled"`

	// Servers defines configured language servers
	Servers map[string]LSPServerConfig `json:"servers,omitempty"`
}

// LSPServerConfig defines configuration for a language server
type LSPServerConfig struct {
	// Command to start the language server
	Command string `json:"command"`

	// Args for the command
	Args []string `json:"args,omitempty"`

	// FileTypes this server handles
	FileTypes []string `json:"file_types,omitempty"`
}

// DataConfig defines data directory configuration
type DataConfig struct {
	// Directory where application data is stored
	Directory string `json:"directory,omitempty"`
}

// ProviderConfig defines configuration for a model provider
type ProviderConfig struct {
	// APIKey for the provider
	APIKey string `json:"api_key,omitempty"`

	// BaseURL for the provider
	BaseURL string `json:"base_url,omitempty"`

	// Enabled indicates if the provider is enabled
	Enabled bool `json:"enabled"`
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	// Logging configuration
	Logging LoggingConfig `json:"logging,omitempty"`
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string `json:"level,omitempty"`

	// Format is the output format (text, json)
	Format string `json:"format,omitempty"`

	// Output destinations (console, file)
	Outputs []string `json:"outputs,omitempty"`

	// File output configuration
	File FileLoggingConfig `json:"file,omitempty"`
}

// FileLoggingConfig defines file logging configuration
type FileLoggingConfig struct {
	// Path to log file
	Path string `json:"path,omitempty"`

	// MaxSize in MB before rotation
	MaxSize int `json:"max_size,omitempty"`

	// MaxBackups to keep
	MaxBackups int `json:"max_backups,omitempty"`

	// MaxAge in days
	MaxAge int `json:"max_age,omitempty"`

	// Compress rotated files
	Compress bool `json:"compress"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	// Provider specifies the AI provider (e.g., "openrouter")
	Provider string `json:"provider" validate:"provider"`

	// BaseURL overrides the default API endpoint
	BaseURL string `json:"base_url,omitempty" validate:"omitempty,url"`

	// APIKey for authentication (can be omitted if using env vars)
	APIKey string `json:"api_key,omitempty"`

	// APIKeyEnvVar specifies the environment variable to read the API key from
	APIKeyEnvVar string `json:"api_key_env_var,omitempty"`

	// Headers for additional API headers
	Headers map[string]string `json:"headers,omitempty"`

	// Timeout for API requests
	Timeout time.Duration `json:"timeout,omitempty" validate:"min=0"`

	// RetryConfig for API request retries
	Retry RetryConfig `json:"retry,omitempty"`

	// RateLimit configuration
	RateLimit RateLimitConfig `json:"rate_limit,omitempty"`
}

// RetryConfig defines retry behavior for API requests
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	Multiplier      float64       `json:"multiplier"`
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	BurstSize         int `json:"burst_size"`
}

// PermissionsConfig defines tool and file system permissions
type PermissionsConfig struct {
	// DefaultMode sets the default permission mode ("allow", "deny", "prompt")
	DefaultMode string `json:"default_mode" validate:"permission_mode"`

	// Tools permissions
	Tools ToolPermissions `json:"tools"`

	// FileSystem permissions
	FileSystem FileSystemPermissions `json:"filesystem"`

	// Commands permissions
	Commands CommandPermissions `json:"commands"`

	// Network permissions
	Network NetworkPermissions `json:"network"`
}

// ToolPermissions defines which tools are allowed or denied
type ToolPermissions struct {
	// Allow lists specific allowed tool patterns
	Allow []string `json:"allow,omitempty"`

	// Deny lists specific denied tool patterns
	Deny []string `json:"deny,omitempty"`

	// RequireConfirmation lists tools that require user confirmation
	RequireConfirmation []string `json:"require_confirmation,omitempty"`

	// CustomRules for more complex permission logic
	CustomRules []PermissionRule `json:"custom_rules,omitempty"`
}

// FileSystemPermissions defines file system access permissions
type FileSystemPermissions struct {
	// ReadPaths lists allowed paths for read operations
	ReadPaths []string `json:"read_paths,omitempty"`

	// WritePaths lists allowed paths for write operations
	WritePaths []string `json:"write_paths,omitempty"`

	// DenyPaths lists explicitly denied paths
	DenyPaths []string `json:"deny_paths,omitempty"`

	// MaxFileSize limits the size of files that can be read/written
	MaxFileSize int64 `json:"max_file_size,omitempty"`

	// AllowedExtensions restricts file operations to specific extensions
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`

	// DeniedExtensions prevents operations on specific file types
	DeniedExtensions []string `json:"denied_extensions,omitempty"`

	// SandboxMode restricts all operations to project directory
	SandboxMode bool `json:"sandbox_mode"`
}

// CommandPermissions defines command execution permissions
type CommandPermissions struct {
	// AllowedCommands lists specific allowed commands
	AllowedCommands []string `json:"allowed_commands,omitempty"`

	// DeniedCommands lists specific denied commands
	DeniedCommands []string `json:"denied_commands,omitempty"`

	// AllowedPatterns lists allowed command patterns (regex)
	AllowedPatterns []string `json:"allowed_patterns,omitempty"`

	// DeniedPatterns lists denied command patterns (regex)
	DeniedPatterns []string `json:"denied_patterns,omitempty"`

	// RequireSudo controls whether sudo commands are allowed
	RequireSudo bool `json:"require_sudo"`

	// MaxTimeout limits command execution time
	MaxTimeout time.Duration `json:"max_timeout,omitempty"`

	// Environment variables to filter out
	FilterEnvVars []string `json:"filter_env_vars,omitempty"`
}

// NetworkPermissions defines network access permissions
type NetworkPermissions struct {
	// AllowedDomains lists allowed domains for web fetch
	AllowedDomains []string `json:"allowed_domains,omitempty"`

	// DeniedDomains lists denied domains
	DeniedDomains []string `json:"denied_domains,omitempty"`

	// AllowLocalhost controls access to localhost
	AllowLocalhost bool `json:"allow_localhost"`

	// AllowPrivateNetworks controls access to private IP ranges
	AllowPrivateNetworks bool `json:"allow_private_networks"`

	// MaxRequestSize limits the size of network requests
	MaxRequestSize int64 `json:"max_request_size,omitempty"`
}

// PermissionRule defines a custom permission rule
type PermissionRule struct {
	// Name of the rule
	Name string `json:"name"`

	// Pattern to match (regex)
	Pattern string `json:"pattern"`

	// Action to take ("allow", "deny", "prompt")
	Action string `json:"action"`

	// Conditions for when this rule applies
	Conditions map[string]interface{} `json:"conditions,omitempty"`

	// Message to show when prompting
	Message string `json:"message,omitempty"`
}

// SecurityConfig holds security-related settings
type SecurityConfig struct {
	// RequireConfirmation for all potentially destructive operations
	RequireConfirmation bool `json:"require_confirmation"`

	// ShowPreview before executing operations
	ShowPreview bool `json:"show_preview"`

	// LogOperations enables operation logging
	LogOperations bool `json:"log_operations"`

	// AuditLog configuration
	AuditLog AuditLogConfig `json:"audit_log,omitempty"`

	// Encryption settings
	Encryption EncryptionConfig `json:"encryption,omitempty"`

	// SessionTimeout for idle sessions
	SessionTimeout time.Duration `json:"session_timeout,omitempty"`
}

// AuditLogConfig defines audit logging settings
type AuditLogConfig struct {
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	MaxSize    int64  `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	Format     string `json:"format"` // "json" or "text"
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	// EncryptConfig controls whether to encrypt the config file
	EncryptConfig bool `json:"encrypt_config"`

	// EncryptLogs controls whether to encrypt log files
	EncryptLogs bool `json:"encrypt_logs"`

	// KeyDerivation method ("pbkdf2", "scrypt", "argon2")
	KeyDerivation string `json:"key_derivation,omitempty"`
}

// PreferencesConfig holds user preferences
type PreferencesConfig struct {
	// Editor settings
	Editor EditorConfig `json:"editor,omitempty"`

	// UI preferences
	UI UIConfig `json:"ui,omitempty"`

	// Output preferences
	Output OutputConfig `json:"output,omitempty"`

	// Behavior preferences
	Behavior BehaviorConfig `json:"behavior,omitempty"`
}

// EditorConfig defines editor preferences
type EditorConfig struct {
	// Command to launch external editor
	Command string `json:"command,omitempty"`

	// Args for editor command
	Args []string `json:"args,omitempty"`

	// TabSize for indentation
	TabSize int `json:"tab_size,omitempty"`

	// UseTabs vs spaces
	UseTabs bool `json:"use_tabs"`

	// LineEndings ("lf", "crlf", "auto")
	LineEndings string `json:"line_endings,omitempty"`
}

// UIConfig defines UI preferences
type UIConfig struct {
	// Theme ("light", "dark", "auto")
	Theme string `json:"theme,omitempty"`

	// Colors customization
	Colors map[string]string `json:"colors,omitempty"`

	// ShowLineNumbers in code display
	ShowLineNumbers bool `json:"show_line_numbers"`

	// SyntaxHighlighting enables syntax highlighting
	SyntaxHighlighting bool `json:"syntax_highlighting"`

	// CompactMode reduces UI verbosity
	CompactMode bool `json:"compact_mode"`
}

// OutputConfig defines output preferences
type OutputConfig struct {
	// Format for structured output ("json", "yaml", "table")
	Format string `json:"format,omitempty"`

	// Verbose enables detailed output
	Verbose bool `json:"verbose"`

	// Quiet suppresses non-essential output
	Quiet bool `json:"quiet"`

	// TimestampFormat for log entries
	TimestampFormat string `json:"timestamp_format,omitempty"`

	// ShowDiff enables diff display for file changes
	ShowDiff bool `json:"show_diff"`
}

// BehaviorConfig defines behavior preferences
type BehaviorConfig struct {
	// AutoSave enables automatic saving of changes
	AutoSave bool `json:"auto_save"`

	// ConfirmBeforeExit prompts before exiting
	ConfirmBeforeExit bool `json:"confirm_before_exit"`

	// HistorySize limits command history
	HistorySize int `json:"history_size,omitempty"`

	// CacheDuration for various caches
	CacheDuration time.Duration `json:"cache_duration,omitempty"`

	// DefaultTimeout for operations
	DefaultTimeout time.Duration `json:"default_timeout,omitempty"`
}

// ProjectConfig holds project-specific settings
type ProjectConfig struct {
	// Root directory of the project
	Root string `json:"root,omitempty"`

	// Name of the project
	Name string `json:"name,omitempty"`

	// Type of project (e.g., "go", "node", "python")
	Type string `json:"type,omitempty"`

	// IgnorePatterns for file operations
	IgnorePatterns []string `json:"ignore_patterns,omitempty"`

	// UseGitIgnore respects .gitignore files
	UseGitIgnore bool `json:"use_gitignore"`

	// CustomSettings for project-specific configuration
	CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
}

// ToolConfig holds tool-specific configuration
type ToolConfig struct {
	// Enabled controls whether the tool is available
	Enabled bool `json:"enabled"`

	// Config holds tool-specific settings
	Config map[string]interface{} `json:"config,omitempty"`

	// Permissions overrides for this tool
	Permissions *ToolPermissions `json:"permissions,omitempty"`
}

// ConfigPrecedence defines the order of configuration loading
type ConfigPrecedence struct {
	// SystemConfig path
	SystemConfig string

	// UserConfig path
	UserConfig string

	// ProjectConfig path
	ProjectConfig string

	// LocalConfig path
	LocalConfig string

	// EnvironmentPrefix for env var overrides
	EnvironmentPrefix string
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e ValidationError) Error() string {
	return e.Message
}

// ConfigSource indicates where a configuration value came from
type ConfigSource string

const (
	SourceDefault     ConfigSource = "default"
	SourceSystem      ConfigSource = "system"
	SourceUser        ConfigSource = "user"
	SourceProject     ConfigSource = "project"
	SourceLocal       ConfigSource = "local"
	SourceEnvironment ConfigSource = "environment"
	SourceCLI         ConfigSource = "cli"
)

// Agent types
const (
	AgentCoder = "coder"
)

// AgentConfig holds basic agent configuration
type AgentConfig struct {
	Model        string  `json:"model"`
	Temperature  float32 `json:"temperature" validate:"min=0,max=2"`
	MaxTokens    int     `json:"max_tokens" validate:"min=1"`
	SystemPrompt string  `json:"system_prompt"`
	MaxRetries   int     `json:"max_retries"`
	RetryDelay   int     `json:"retry_delay"`
}

// MCPServerConfig holds MCP server configuration
type MCPServerConfig struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	TransportType string            `json:"transport_type,omitempty"`
}
