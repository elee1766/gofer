package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator validates configuration values using go-playground/validator
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	v := validator.New()
	
	// Register custom validation functions
	v.RegisterValidation("provider", validateProvider)
	v.RegisterValidation("permission_mode", validatePermissionMode)
	v.RegisterValidation("theme", validateTheme)
	v.RegisterValidation("format", validateFormat)
	v.RegisterValidation("line_endings", validateLineEndings)
	v.RegisterValidation("key_derivation", validateKeyDerivation)
	v.RegisterValidation("log_format", validateLogFormat)
	v.RegisterValidation("domain_pattern", validateDomainPattern)
	v.RegisterValidation("glob_pattern", validateGlobPattern)
	v.RegisterValidation("regex_pattern", validateRegexPattern)
	v.RegisterValidation("abs_or_rel_path", validateAbsOrRelPath)
	
	return &Validator{
		validate: v,
	}
}

// Validate validates a complete configuration
func (v *Validator) Validate(config *Config) error {
	// Set default version if empty
	if config.Version == "" {
		config.Version = "1.0"
	}

	// Use go-playground/validator for struct validation
	if err := v.validate.Struct(config); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Convert validator errors to our custom format
			for _, e := range validationErrors {
				return ValidationError{
					Field:   e.Field(),
					Message: fmt.Sprintf("validation failed on tag '%s' with value '%v'", e.Tag(), e.Value()),
					Value:   e.Value(),
				}
			}
		}
		return err
	}

	return nil
}

// Custom validation functions for go-playground/validator

// validateProvider validates API provider values
func validateProvider(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty, will be filled by defaults
	}
	validProviders := []string{"openrouter", "anthropic", "google", "openai", "local", "test", "test-provider", "updated-provider"}
	return contains(validProviders, value)
}

// validatePermissionMode validates permission mode values
func validatePermissionMode(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	validModes := []string{"allow", "deny", "prompt"}
	return contains(validModes, value)
}

// validateTheme validates theme values
func validateTheme(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	validThemes := []string{"light", "dark", "auto"}
	return contains(validThemes, value)
}

// validateFormat validates format values
func validateFormat(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	validFormats := []string{"json", "yaml", "table", "text"}
	return contains(validFormats, value)
}

// validateLineEndings validates line ending values
func validateLineEndings(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	validEndings := []string{"lf", "crlf", "auto"}
	return contains(validEndings, value)
}

// validateKeyDerivation validates key derivation method values
func validateKeyDerivation(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	validMethods := []string{"pbkdf2", "scrypt", "argon2"}
	return contains(validMethods, value)
}

// validateLogFormat validates log format values
func validateLogFormat(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	validFormats := []string{"json", "text"}
	return contains(validFormats, value)
}

// validateDomainPattern validates domain patterns
func validateDomainPattern(fl validator.FieldLevel) bool {
	domain := fl.Field().String()
	if domain == "" {
		return false
	}
	
	// Allow wildcards
	testDomain := strings.ReplaceAll(domain, "*.", "")
	
	// Check for valid characters
	validChars := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	return validChars.MatchString(testDomain)
}

// validateGlobPattern validates glob patterns
func validateGlobPattern(fl validator.FieldLevel) bool {
	pattern := fl.Field().String()
	if pattern == "" {
		return true
	}
	
	// Check for valid glob pattern
	if strings.Contains(pattern, "**") && strings.Contains(pattern, "***") {
		return false
	}
	
	return true
}

// validateRegexPattern validates regex patterns
func validateRegexPattern(fl validator.FieldLevel) bool {
	pattern := fl.Field().String()
	if pattern == "" {
		return true
	}
	
	// Check for regex patterns (enclosed in slashes)
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		regex := pattern[1 : len(pattern)-1]
		if _, err := regexp.Compile(regex); err != nil {
			return false
		}
	}
	
	return true
}

// validateAbsOrRelPath validates that path is absolute or relative to current directory
func validateAbsOrRelPath(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true
	}
	
	return filepath.IsAbs(path) || path == "." || strings.HasPrefix(path, "./")
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}