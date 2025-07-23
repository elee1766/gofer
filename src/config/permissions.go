package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// PermissionChecker checks if operations are allowed based on configuration
type PermissionChecker struct {
	config *PermissionsConfig
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(config *PermissionsConfig) *PermissionChecker {
	return &PermissionChecker{
		config: config,
	}
}

// CheckToolPermission checks if a tool operation is allowed
func (p *PermissionChecker) CheckToolPermission(toolName string, args map[string]interface{}) (PermissionResult, error) {
	// Build the full tool call string for pattern matching
	toolCall := p.buildToolCallString(toolName, args)

	// Check deny list first
	for _, pattern := range p.config.Tools.Deny {
		if matched, _ := p.matchPattern(toolCall, pattern); matched {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Tool call matches deny pattern: %s", pattern),
			}, nil
		}
	}

	// Check custom rules
	for _, rule := range p.config.Tools.CustomRules {
		if matched, _ := p.matchPattern(toolCall, rule.Pattern); matched {
			if p.evaluateConditions(rule.Conditions, args) {
				result := PermissionResult{
					Allowed: rule.Action == "allow",
					Reason:  rule.Message,
				}
				if rule.Action == "prompt" {
					result.RequiresConfirmation = true
					result.ConfirmationMessage = rule.Message
				}
				return result, nil
			}
		}
	}

	// Check allow list
	for _, pattern := range p.config.Tools.Allow {
		if matched, _ := p.matchPattern(toolCall, pattern); matched {
			// Check if confirmation required
			requiresConfirm := false
			for _, confirmPattern := range p.config.Tools.RequireConfirmation {
				if matched, _ := p.matchPattern(toolCall, confirmPattern); matched {
					requiresConfirm = true
					break
				}
			}
			
			return PermissionResult{
				Allowed:              true,
				RequiresConfirmation: requiresConfirm,
				ConfirmationMessage:  fmt.Sprintf("Confirm execution of: %s", toolCall),
			}, nil
		}
	}

	// Default based on mode
	switch p.config.DefaultMode {
	case "allow":
		return PermissionResult{Allowed: true}, nil
	case "deny":
		return PermissionResult{
			Allowed: false,
			Reason:  "Tool not in allow list and default mode is deny",
		}, nil
	case "prompt":
		return PermissionResult{
			Allowed:              true,
			RequiresConfirmation: true,
			ConfirmationMessage:  fmt.Sprintf("Allow tool call: %s?", toolCall),
		}, nil
	default:
		return PermissionResult{Allowed: true}, nil
	}
}

// CheckFileReadPermission checks if reading a file is allowed
func (p *PermissionChecker) CheckFileReadPermission(path string) (PermissionResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return PermissionResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Invalid path: %v", err),
		}, nil
	}

	// Check deny paths first
	for _, denyPath := range p.config.FileSystem.DenyPaths {
		if p.isPathUnder(absPath, denyPath) {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Path is in denied directory: %s", denyPath),
			}, nil
		}
	}

	// Check file extension
	ext := filepath.Ext(absPath)
	for _, deniedExt := range p.config.FileSystem.DeniedExtensions {
		if ext == deniedExt {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("File extension %s is denied", ext),
			}, nil
		}
	}

	// In sandbox mode, only allow paths under allowed directories
	if p.config.FileSystem.SandboxMode {
		allowed := false
		for _, readPath := range p.config.FileSystem.ReadPaths {
			if p.isPathUnder(absPath, readPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return PermissionResult{
				Allowed: false,
				Reason:  "Path is outside allowed directories in sandbox mode",
			}, nil
		}
	}

	// Check if path is in read paths
	for _, readPath := range p.config.FileSystem.ReadPaths {
		if p.isPathUnder(absPath, readPath) {
			return PermissionResult{Allowed: true}, nil
		}
	}

	// Default based on mode
	switch p.config.DefaultMode {
	case "allow":
		return PermissionResult{Allowed: true}, nil
	case "deny":
		return PermissionResult{
			Allowed: false,
			Reason:  "Path not in allowed read paths and default mode is deny",
		}, nil
	case "prompt":
		return PermissionResult{
			Allowed:              true,
			RequiresConfirmation: true,
			ConfirmationMessage:  fmt.Sprintf("Allow reading file: %s?", path),
		}, nil
	default:
		return PermissionResult{Allowed: true}, nil
	}
}

// CheckFileWritePermission checks if writing a file is allowed
func (p *PermissionChecker) CheckFileWritePermission(path string) (PermissionResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return PermissionResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Invalid path: %v", err),
		}, nil
	}

	// Check deny paths first
	for _, denyPath := range p.config.FileSystem.DenyPaths {
		if p.isPathUnder(absPath, denyPath) {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Path is in denied directory: %s", denyPath),
			}, nil
		}
	}

	// Check file extension
	ext := filepath.Ext(absPath)
	for _, deniedExt := range p.config.FileSystem.DeniedExtensions {
		if ext == deniedExt {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("File extension %s is denied", ext),
			}, nil
		}
	}

	// In sandbox mode, only allow paths under allowed directories
	if p.config.FileSystem.SandboxMode {
		allowed := false
		for _, writePath := range p.config.FileSystem.WritePaths {
			if p.isPathUnder(absPath, writePath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return PermissionResult{
				Allowed: false,
				Reason:  "Path is outside allowed directories in sandbox mode",
			}, nil
		}
	}

	// Check if path is in write paths
	for _, writePath := range p.config.FileSystem.WritePaths {
		if p.isPathUnder(absPath, writePath) {
			return PermissionResult{
				Allowed:              true,
				RequiresConfirmation: true,
				ConfirmationMessage:  fmt.Sprintf("Confirm writing to file: %s", path),
			}, nil
		}
	}

	// Default based on mode
	switch p.config.DefaultMode {
	case "allow":
		return PermissionResult{
			Allowed:              true,
			RequiresConfirmation: true,
			ConfirmationMessage:  fmt.Sprintf("Confirm writing to file: %s", path),
		}, nil
	case "deny":
		return PermissionResult{
			Allowed: false,
			Reason:  "Path not in allowed write paths and default mode is deny",
		}, nil
	case "prompt":
		return PermissionResult{
			Allowed:              true,
			RequiresConfirmation: true,
			ConfirmationMessage:  fmt.Sprintf("Allow writing file: %s?", path),
		}, nil
	default:
		return PermissionResult{
			Allowed:              true,
			RequiresConfirmation: true,
			ConfirmationMessage:  fmt.Sprintf("Confirm writing to file: %s", path),
		}, nil
	}
}

// CheckCommandPermission checks if executing a command is allowed
func (p *PermissionChecker) CheckCommandPermission(command string, args []string) (PermissionResult, error) {
	fullCommand := command
	if len(args) > 0 {
		fullCommand = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}

	// Check denied commands first
	for _, denied := range p.config.Commands.DeniedCommands {
		if strings.Contains(fullCommand, denied) {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Command contains denied pattern: %s", denied),
			}, nil
		}
	}

	// Check denied patterns
	for _, pattern := range p.config.Commands.DeniedPatterns {
		if matched, _ := regexp.MatchString(pattern, fullCommand); matched {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Command matches denied pattern: %s", pattern),
			}, nil
		}
	}

	// Check if sudo is required but not allowed
	if strings.HasPrefix(fullCommand, "sudo") && !p.config.Commands.RequireSudo {
		return PermissionResult{
			Allowed: false,
			Reason:  "Sudo commands are not allowed",
		}, nil
	}

	// Check allowed commands
	if len(p.config.Commands.AllowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range p.config.Commands.AllowedCommands {
			if command == allowedCmd || strings.HasPrefix(fullCommand, allowedCmd) {
				allowed = true
				break
			}
		}
		if !allowed {
			return PermissionResult{
				Allowed: false,
				Reason:  "Command not in allowed list",
			}, nil
		}
	}

	// Check allowed patterns
	if len(p.config.Commands.AllowedPatterns) > 0 {
		allowed := false
		for _, pattern := range p.config.Commands.AllowedPatterns {
			if matched, _ := regexp.MatchString(pattern, fullCommand); matched {
				allowed = true
				break
			}
		}
		if !allowed {
			return PermissionResult{
				Allowed: false,
				Reason:  "Command does not match any allowed pattern",
			}, nil
		}
	}

	// Default behavior - always require confirmation for commands
	return PermissionResult{
		Allowed:              true,
		RequiresConfirmation: true,
		ConfirmationMessage:  fmt.Sprintf("Execute command: %s?", fullCommand),
	}, nil
}

// CheckNetworkPermission checks if a network request is allowed
func (p *PermissionChecker) CheckNetworkPermission(url string) (PermissionResult, error) {
	// Parse domain from URL
	domain := p.extractDomain(url)

	// Check localhost
	if p.isLocalhost(domain) && !p.config.Network.AllowLocalhost {
		return PermissionResult{
			Allowed: false,
			Reason:  "Localhost access is not allowed",
		}, nil
	}

	// Check private networks
	if p.isPrivateNetwork(domain) && !p.config.Network.AllowPrivateNetworks {
		return PermissionResult{
			Allowed: false,
			Reason:  "Private network access is not allowed",
		}, nil
	}

	// Check denied domains
	for _, denied := range p.config.Network.DeniedDomains {
		if p.matchDomain(domain, denied) {
			return PermissionResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Domain is denied: %s", denied),
			}, nil
		}
	}

	// Check allowed domains if specified
	if len(p.config.Network.AllowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range p.config.Network.AllowedDomains {
			if p.matchDomain(domain, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return PermissionResult{
				Allowed: false,
				Reason:  "Domain not in allowed list",
			}, nil
		}
	}

	return PermissionResult{Allowed: true}, nil
}

// buildToolCallString builds a string representation of a tool call
func (p *PermissionChecker) buildToolCallString(toolName string, args map[string]interface{}) string {
	if len(args) == 0 {
		return toolName
	}

	// For common patterns, build a more readable string
	if cmd, ok := args["command"].(string); ok {
		return fmt.Sprintf("%s(%s)", toolName, cmd)
	}
	if path, ok := args["path"].(string); ok {
		return fmt.Sprintf("%s(%s)", toolName, path)
	}
	if url, ok := args["url"].(string); ok {
		return fmt.Sprintf("%s(%s)", toolName, url)
	}

	return toolName
}

// matchPattern matches a string against a pattern (glob or regex)
func (p *PermissionChecker) matchPattern(str, pattern string) (bool, error) {
	// Check if it's a regex pattern (enclosed in slashes)
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		regex := pattern[1 : len(pattern)-1]
		return regexp.MatchString(regex, str)
	}

	// Otherwise treat as glob pattern
	return filepath.Match(pattern, str)
}

// evaluateConditions evaluates custom rule conditions
func (p *PermissionChecker) evaluateConditions(conditions map[string]interface{}, args map[string]interface{}) bool {
	if len(conditions) == 0 {
		return true
	}

	// Simple condition evaluation - can be extended
	for key, expectedValue := range conditions {
		actualValue, exists := args[key]
		if !exists {
			return false
		}
		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}

	return true
}

// isPathUnder checks if a path is under a parent directory
func (p *PermissionChecker) isPathUnder(path, parent string) bool {
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return false
	}

	relPath, err := filepath.Rel(absParent, path)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(relPath, "..")
}

// extractDomain extracts domain from URL
func (p *PermissionChecker) extractDomain(urlStr string) string {
	// Simple domain extraction - in production use proper URL parsing
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")
	parts := strings.Split(urlStr, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return urlStr
}

// isLocalhost checks if domain is localhost
func (p *PermissionChecker) isLocalhost(domain string) bool {
	return domain == "localhost" || 
		domain == "127.0.0.1" || 
		strings.HasPrefix(domain, "127.") ||
		domain == "::1" ||
		strings.HasSuffix(domain, ".local")
}

// isPrivateNetwork checks if domain is in private network range
func (p *PermissionChecker) isPrivateNetwork(domain string) bool {
	privatePatterns := []string{
		"10.*",
		"172.16.*", "172.17.*", "172.18.*", "172.19.*",
		"172.20.*", "172.21.*", "172.22.*", "172.23.*",
		"172.24.*", "172.25.*", "172.26.*", "172.27.*",
		"172.28.*", "172.29.*", "172.30.*", "172.31.*",
		"192.168.*",
	}

	for _, pattern := range privatePatterns {
		if matched, _ := filepath.Match(pattern, domain); matched {
			return true
		}
	}

	return false
}

// matchDomain matches a domain against a pattern
func (p *PermissionChecker) matchDomain(domain, pattern string) bool {
	// Handle wildcard subdomains
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(domain, suffix)
	}

	return domain == pattern
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Allowed              bool
	RequiresConfirmation bool
	ConfirmationMessage  string
	Reason               string
}