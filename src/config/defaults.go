package config

import (
	"time"
)

// DefaultConfig returns a default configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		API: APIConfig{
			Provider:     "openrouter",
			APIKeyEnvVar: "OPENROUTER_API_KEY",
			Timeout:      30 * time.Second,
			Retry: RetryConfig{
				MaxRetries:   3,
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				RetryableErrors: []string{
					"rate_limit_exceeded",
					"timeout",
					"connection_error",
				},
			},
			RateLimit: RateLimitConfig{
				RequestsPerMinute: 60,
				TokensPerMinute:   100000,
				BurstSize:         10,
			},
		},

		Agent: AgentConfig{
			Model:      "google/gemini-2.5-flash",
			MaxTokens:  4096,
			MaxRetries: 3,
			RetryDelay: 1000,
		},

		Agents: map[string]AgentConfig{
			AgentCoder: {
				Model:      "google/gemini-2.5-flash",
				MaxTokens:  4096,
				MaxRetries: 3,
				RetryDelay: 1000,
			},
		},

		Permissions: PermissionsConfig{
			DefaultMode: "prompt",
			Tools: ToolPermissions{
				Allow: []string{
					"read_file*",
					"write_file*",
					"list_directory*",
					"execute_command*",
					"web_search*",
					"web_fetch*",
				},
				Deny: []string{
					"system_*",
					"admin_*",
				},
				RequireConfirmation: []string{
					"execute_command*",
					"write_file*",
					"delete_*",
				},
			},
			FileSystem: FileSystemPermissions{
				ReadPaths: []string{
					".",
					"~/.config/gofer",
				},
				WritePaths: []string{
					".",
				},
				DenyPaths: []string{
					"/etc",
					"/sys",
					"/proc",
					"/dev",
					"/boot",
					"/root",
				},
				MaxFileSize: 10 * 1024 * 1024, // 10MB
				DeniedExtensions: []string{
					".exe", ".dll", ".so", ".dylib",
					".app", ".dmg", ".pkg", ".deb", ".rpm",
				},
				SandboxMode: false,
			},
			Commands: CommandPermissions{
				DeniedCommands: []string{
					"rm -rf /",
					"format",
					"fdisk",
					"dd",
					"mkfs",
					"shutdown",
					"reboot",
					"halt",
					"poweroff",
					"systemctl",
					"service",
				},
				DeniedPatterns: []string{
					`.*\brm\s+-rf\s+/.*`,
					`.*\bsudo\s+rm.*`,
					`.*\b(curl|wget).*\|\s*(bash|sh).*`,
				},
				MaxTimeout: 5 * time.Minute,
				FilterEnvVars: []string{
					"AWS_SECRET_ACCESS_KEY",
					"AWS_SESSION_TOKEN",
					"GITHUB_TOKEN",
					"GITLAB_TOKEN",
					"NPM_TOKEN",
					"PYPI_PASSWORD",
				},
			},
			Network: NetworkPermissions{
				DeniedDomains: []string{
					"localhost",
					"127.0.0.1",
					"0.0.0.0",
					"*.local",
				},
				AllowLocalhost:       false,
				AllowPrivateNetworks: false,
				MaxRequestSize:       50 * 1024 * 1024, // 50MB
			},
		},

		Security: SecurityConfig{
			RequireConfirmation: true,
			ShowPreview:         true,
			LogOperations:       true,
			SessionTimeout:      30 * time.Minute,
			AuditLog: AuditLogConfig{
				Enabled:    false,
				Path:       "~/.local/share/gofer/audit.log",
				MaxSize:    100 * 1024 * 1024, // 100MB
				MaxBackups: 5,
				Format:     "json",
			},
			Encryption: EncryptionConfig{
				EncryptConfig: false,
				EncryptLogs:   false,
				KeyDerivation: "argon2",
			},
		},

		Preferences: PreferencesConfig{
			Editor: EditorConfig{
				TabSize:     4,
				UseTabs:     false,
				LineEndings: "auto",
			},
			UI: UIConfig{
				Theme:              "auto",
				ShowLineNumbers:    true,
				SyntaxHighlighting: true,
				CompactMode:        false,
			},
			Output: OutputConfig{
				Format:          "text",
				Verbose:         false,
				Quiet:           false,
				TimestampFormat: "2006-01-02 15:04:05",
				ShowDiff:        true,
			},
			Behavior: BehaviorConfig{
				AutoSave:          true,
				ConfirmBeforeExit: true,
				HistorySize:       1000,
				CacheDuration:     24 * time.Hour,
				DefaultTimeout:    30 * time.Second,
			},
		},

		Project: ProjectConfig{
			Root:         ".",
			UseGitIgnore: true,
			IgnorePatterns: []string{
				"node_modules/",
				"vendor/",
				".git/",
				"dist/",
				"build/",
				"*.log",
				"*.tmp",
				".DS_Store",
			},
		},

		Tools: map[string]ToolConfig{
			"bash": {
				Enabled: true,
				Config: map[string]interface{}{
					"shell":          "/bin/bash",
					"timeout":        300,
					"working_dir":    ".",
					"inherit_env":    true,
					"capture_output": true,
				},
			},
			"file": {
				Enabled: true,
				Config: map[string]interface{}{
					"encoding":        "utf-8",
					"line_endings":    "auto",
					"create_backup":   false,
					"validate_syntax": true,
				},
			},
			"web": {
				Enabled: true,
				Config: map[string]interface{}{
					"user_agent":      "gofer/1.0",
					"follow_redirect": true,
					"max_redirects":   5,
					"timeout":         30,
					"verify_ssl":      true,
				},
			},
		},
	}
}

// DefaultOpenRouterConfig returns default configuration for OpenRouter
func DefaultOpenRouterConfig() *Config {
	config := DefaultConfig()
	config.API.Provider = "openrouter"
	config.API.BaseURL = "https://openrouter.ai/api/v1"
	config.API.APIKeyEnvVar = "OPENROUTER_API_KEY"

	// OpenRouter-specific headers
	config.API.Headers = map[string]string{
		"HTTP-Referer": "https://github.com/elee1766/gofer",
		"X-Title":      "gofer",
	}

	return config
}

// DefaultAnthropicConfig returns default configuration for Anthropic API
func DefaultAnthropicConfig() *Config {
	config := DefaultConfig()
	config.API.Provider = "anthropic"
	config.API.BaseURL = "https://api.anthropic.com/v1"
	config.API.APIKeyEnvVar = "ANTHROPIC_API_KEY"
	config.Agent.Model = "claude-3-5-sonnet-20241022"

	return config
}

// DefaultGoogleConfig returns default configuration for Google/Gemini
func DefaultGoogleConfig() *Config {
	config := DefaultConfig()
	config.API.Provider = "google"
	config.API.APIKeyEnvVar = "GOOGLE_API_KEY"
	config.Agent.Model = "gemini-1.5-pro"

	return config
}

// GenerateDefaultConfig generates a default configuration file
func GenerateDefaultConfig(provider string) (*Config, error) {
	var config *Config

	switch provider {
	case "openrouter":
		config = DefaultOpenRouterConfig()
	case "anthropic":
		config = DefaultAnthropicConfig()
	case "google":
		config = DefaultGoogleConfig()
	default:
		config = DefaultConfig()
	}

	return config, nil
}

// MergeWithDefaults merges a partial configuration with defaults
func MergeWithDefaults(partial *Config) *Config {
	defaults := DefaultConfig()
	loader := &Loader{}
	return loader.mergeConfigs(defaults, partial)
}

