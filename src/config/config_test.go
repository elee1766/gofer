package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check version
	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}

	// Check API defaults
	if config.API.Provider != "openrouter" {
		t.Errorf("Expected provider openrouter, got %s", config.API.Provider)
	}

	// Check agent defaults
	if config.Agent.Model == "" {
		t.Error("Expected model to be set")
	}

	// Check permissions defaults
	if config.Permissions.DefaultMode != "prompt" {
		t.Errorf("Expected default mode prompt, got %s", config.Permissions.DefaultMode)
	}
}

func TestConfigValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid temperature",
			config: func() *Config {
				c := DefaultConfig()
				c.Agent.Temperature = 3.0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "negative max tokens",
			config: func() *Config {
				c := DefaultConfig()
				c.Agent.MaxTokens = -1
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid permission mode",
			config: func() *Config {
				c := DefaultConfig()
				c.Permissions.DefaultMode = "invalid"
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPermissionChecker(t *testing.T) {
	config := DefaultConfig()
	checker := NewPermissionChecker(&config.Permissions)

	tests := []struct {
		name       string
		checkFunc  func() (PermissionResult, error)
		wantAllowed bool
		wantConfirm bool
	}{
		{
			name: "allowed tool",
			checkFunc: func() (PermissionResult, error) {
				return checker.CheckToolPermission("read_file", map[string]interface{}{
					"path": "test.txt",
				})
			},
			wantAllowed: true,
			wantConfirm: false,
		},
		{
			name: "denied tool pattern",
			checkFunc: func() (PermissionResult, error) {
				return checker.CheckToolPermission("system_shutdown", nil)
			},
			wantAllowed: false,
			wantConfirm: false,
		},
		{
			name: "command requires confirmation",
			checkFunc: func() (PermissionResult, error) {
				return checker.CheckCommandPermission("ls", []string{"-la"})
			},
			wantAllowed: true,
			wantConfirm: true,
		},
		{
			name: "denied command",
			checkFunc: func() (PermissionResult, error) {
				return checker.CheckCommandPermission("rm", []string{"-rf", "/"})
			},
			wantAllowed: false,
			wantConfirm: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.checkFunc()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Allowed != tt.wantAllowed {
				t.Errorf("Expected allowed=%v, got %v (reason: %s)", 
					tt.wantAllowed, result.Allowed, result.Reason)
			}

			if result.RequiresConfirmation != tt.wantConfirm {
				t.Errorf("Expected confirmation=%v, got %v", 
					tt.wantConfirm, result.RequiresConfirmation)
			}
		})
	}
}

func TestConfigLoader(t *testing.T) {
	// Create temporary directory for test configs
	tempDir := t.TempDir()

	// Create test config file
	testConfig := &Config{
		Version: "1.0",
		API: APIConfig{
			Provider: "test",
			Timeout:  10 * time.Second,
		},
		Agent: DefaultConfig().Agent,
		Permissions: DefaultConfig().Permissions,
	}

	configPath := filepath.Join(tempDir, "config.json")
	
	loader := NewLoader(ConfigPrecedence{
		UserConfig: configPath,
	})

	// Test saving
	if err := loader.SaveFile(testConfig, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test loading
	loaded, err := loader.loadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.API.Provider != "test" {
		t.Errorf("Expected provider 'test', got %s", loaded.API.Provider)
	}
}

func TestConfigMerging(t *testing.T) {
	loader := &Loader{}

	base := DefaultConfig()
	override := &Config{
		API: APIConfig{
			Provider: "anthropic",
			BaseURL:  "https://api.anthropic.com",
		},
		Agent: base.Agent,
		Permissions: PermissionsConfig{
			DefaultMode: "deny",
		},
	}

	merged := loader.mergeConfigs(base, override)

	// Check overridden values
	if merged.API.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %s", merged.API.Provider)
	}

	if merged.Permissions.DefaultMode != "deny" {
		t.Errorf("Expected permission mode 'deny', got %s", merged.Permissions.DefaultMode)
	}

	// Check preserved values
	if merged.Agent.Temperature != base.Agent.Temperature {
		t.Error("Expected temperature to be preserved")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_API_KEY", "test-key-123")
	os.Setenv("TEST_MODEL", "test-model")
	os.Setenv("TEST_SANDBOX", "true")
	defer func() {
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("TEST_MODEL")
		os.Unsetenv("TEST_SANDBOX")
	}()

	loader := NewLoader(ConfigPrecedence{
		EnvironmentPrefix: "TEST",
	})

	config := DefaultConfig()
	loader.applyEnvironmentOverrides(config)

	if config.API.APIKey != "test-key-123" {
		t.Errorf("Expected API key from environment, got %s", config.API.APIKey)
	}

	if config.Agent.Model != "test-model" {
		t.Errorf("Expected model from environment, got %s", config.Agent.Model)
	}

	if !config.Permissions.FileSystem.SandboxMode {
		t.Error("Expected sandbox mode to be enabled from environment")
	}
}

func TestConfigDiscovery(t *testing.T) {
	discovery := NewDiscovery()

	// Test getting standard paths
	systemPath := discovery.GetSystemConfig()
	if systemPath == "" {
		t.Error("Expected system config path")
	}

	userPath := discovery.GetUserConfig()
	if userPath == "" {
		t.Error("Expected user config path")
	}

	// Test path expansion
	expanded := discovery.expandPath("$HOME/test")
	if !filepath.IsAbs(expanded) {
		t.Errorf("Expected absolute path, got %s", expanded)
	}
}

func TestConfigManager(t *testing.T) {
	// Create a test config
	testConfig := DefaultConfig()
	testConfig.API.Provider = "test-provider"

	manager, err := NewManagerWithConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test getting config
	config := manager.GetConfig()
	if config.API.Provider != "test-provider" {
		t.Errorf("Expected provider 'test-provider', got %s", config.API.Provider)
	}

	// Test updating config
	err = manager.Update(map[string]interface{}{
		"api": map[string]interface{}{
			"provider": "updated-provider",
		},
	})
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	config = manager.GetConfig()
	if config.API.Provider != "updated-provider" {
		t.Errorf("Expected updated provider, got %s", config.API.Provider)
	}

	// Test permission checking
	result, err := manager.CheckToolPermission("read_file", map[string]interface{}{
		"path": "test.txt",
	})
	if err != nil {
		t.Fatalf("Permission check failed: %v", err)
	}
	if !result.Allowed {
		t.Error("Expected read_file to be allowed")
	}
}