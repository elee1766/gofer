package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Discovery handles finding configuration files in various locations
type Discovery struct {
	searchPaths []string
	configNames []string
}

// NewDiscovery creates a new configuration discovery instance
func NewDiscovery() *Discovery {
	return &Discovery{
		searchPaths: getDefaultSearchPaths(),
		configNames: getDefaultConfigNames(),
	}
}

// FindConfigs finds all configuration files in standard locations
func (d *Discovery) FindConfigs() ([]ConfigLocation, error) {
	var locations []ConfigLocation

	for _, searchPath := range d.searchPaths {
		// Expand path
		expandedPath := d.expandPath(searchPath)
		
		// Check each config name
		for _, configName := range d.configNames {
			fullPath := filepath.Join(expandedPath, configName)
			
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				source := d.determineSource(searchPath)
				locations = append(locations, ConfigLocation{
					Path:     fullPath,
					Source:   source,
					Priority: d.getPriority(source),
				})
			}
		}
	}

	// Sort by priority (lower number = higher priority)
	d.sortByPriority(locations)

	return locations, nil
}

// FindProjectConfig searches for project configuration starting from current directory
func (d *Discovery) FindProjectConfig(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Walk up the directory tree looking for config
	currentDir := startDir
	for {
		// Check for project config files
		projectConfigNames := []string{
			".gofer/config.json",
			".gofer/config.local.json",
			"gofer.json",
			".gocoderc.json",
		}

		for _, name := range projectConfigNames {
			configPath := filepath.Join(currentDir, name)
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}

		// Check if we've reached the root
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir

		// Stop at home directory
		if home, _ := os.UserHomeDir(); currentDir == home {
			break
		}
	}

	return "", fmt.Errorf("no project configuration found")
}

// GetSystemConfig returns the system-wide configuration path
func (d *Discovery) GetSystemConfig() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "gofer", "config.json")
	}
	return "/etc/gofer/config.json"
}

// GetUserConfig returns the user configuration path
func (d *Discovery) GetUserConfig() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "gofer", "config.json")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Roaming", "gofer", "config.json")
	}

	return filepath.Join(home, ".config", "gofer", "config.json")
}

// CreateDefaultConfig creates a default configuration file at the specified location
func (d *Discovery) CreateDefaultConfig(location string, provider string) error {
	// Generate default config
	config, err := GenerateDefaultConfig(provider)
	if err != nil {
		return fmt.Errorf("failed to generate default config: %w", err)
	}

	// Determine path based on location
	var configPath string
	switch location {
	case "user":
		configPath = d.GetUserConfig()
	case "project":
		configPath = ".gofer/config.json"
	case "local":
		configPath = ".gofer/config.local.json"
	default:
		configPath = location
	}

	// Create directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save config
	loader := NewLoader(ConfigPrecedence{})
	if err := loader.SaveFile(config, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// getDefaultSearchPaths returns the default paths to search for configs
func getDefaultSearchPaths() []string {
	paths := []string{
		// System paths
		"/etc/gofer",
		"$ProgramData/gofer", // Windows

		// User paths
		"$XDG_CONFIG_HOME/gofer",
		"$HOME/.config/gofer",
		"$HOME/.gofer",
		"$APPDATA/gofer", // Windows

		// Project paths
		".",
		".gofer",
	}

	return paths
}

// getDefaultConfigNames returns the default config file names to look for
func getDefaultConfigNames() []string {
	return []string{
		"config.json",
		"config.local.json",
		"gofer.json",
		".gocoderc.json",
		"settings.json",
	}
}

// expandPath expands environment variables and special paths
func (d *Discovery) expandPath(path string) string {
	// Expand environment variables
	path = os.ExpandEnv(path)

	// Handle home directory
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	return path
}

// determineSource determines the config source from the path
func (d *Discovery) determineSource(searchPath string) ConfigSource {
	switch {
	case strings.Contains(searchPath, "/etc") || strings.Contains(searchPath, "ProgramData"):
		return SourceSystem
	case strings.Contains(searchPath, "HOME") || strings.Contains(searchPath, "APPDATA") || strings.Contains(searchPath, "XDG_CONFIG"):
		return SourceUser
	case searchPath == "." || searchPath == ".gofer":
		if strings.Contains(searchPath, "local") {
			return SourceLocal
		}
		return SourceProject
	default:
		return SourceDefault
	}
}

// getPriority returns the priority for a config source
func (d *Discovery) getPriority(source ConfigSource) int {
	priorities := map[ConfigSource]int{
		SourceCLI:         1,
		SourceEnvironment: 2,
		SourceLocal:       3,
		SourceProject:     4,
		SourceUser:        5,
		SourceSystem:      6,
		SourceDefault:     7,
	}

	if priority, ok := priorities[source]; ok {
		return priority
	}
	return 999
}

// sortByPriority sorts config locations by priority
func (d *Discovery) sortByPriority(locations []ConfigLocation) {
	// Simple bubble sort for small lists
	n := len(locations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if locations[j].Priority > locations[j+1].Priority {
				locations[j], locations[j+1] = locations[j+1], locations[j]
			}
		}
	}
}

// ConfigLocation represents a found configuration file
type ConfigLocation struct {
	Path     string
	Source   ConfigSource
	Priority int
}

// ConfigInfo provides information about the configuration setup
type ConfigInfo struct {
	LoadedConfigs []ConfigLocation
	ActiveConfig  string
	Provider      string
	Model         string
	Errors        []string
	Warnings      []string
}

// GetConfigInfo returns information about the current configuration
func GetConfigInfo() (*ConfigInfo, error) {
	discovery := NewDiscovery()
	locations, err := discovery.FindConfigs()
	if err != nil {
		return nil, err
	}

	info := &ConfigInfo{
		LoadedConfigs: locations,
		Errors:        []string{},
		Warnings:      []string{},
	}

	// Try to load the highest priority config
	if len(locations) > 0 {
		loader := NewLoader(GetConfigPaths())
		config, err := loader.loadFile(locations[0].Path)
		if err != nil {
			info.Errors = append(info.Errors, fmt.Sprintf("Failed to load config from %s: %v", locations[0].Path, err))
		} else {
			info.ActiveConfig = locations[0].Path
			info.Provider = config.API.Provider
			info.Model = config.Agent.Model

			// Check for common issues
			if config.API.APIKey == "" && config.API.APIKeyEnvVar != "" {
				if os.Getenv(config.API.APIKeyEnvVar) == "" {
					info.Warnings = append(info.Warnings, fmt.Sprintf("API key environment variable %s is not set", config.API.APIKeyEnvVar))
				}
			}
		}
	} else {
		info.Warnings = append(info.Warnings, "No configuration files found")
	}

	return info, nil
}