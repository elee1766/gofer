package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// ConfigDefaults provides common configuration default patterns
type ConfigDefaults struct{}

func NewConfigDefaults() *ConfigDefaults {
	return &ConfigDefaults{}
}

func (cd *ConfigDefaults) GetStringDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (cd *ConfigDefaults) GetIntDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && parsed == 1 {
			return defaultValue
		}
	}
	return defaultValue
}

func (cd *ConfigDefaults) GetBoolDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

func (cd *ConfigDefaults) GetDurationDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// ConfigValidator provides common configuration validation patterns
type ConfigValidator struct{}

func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

func (cv *ConfigValidator) ValidateRequired(config interface{}, requiredFields []string) []ValidationResult {
	var results []ValidationResult
	val := reflect.ValueOf(config)
	
	// Handle pointer to struct
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		results = append(results, ValidationResult{
			Field:   "config",
			Valid:   false,
			Message: "configuration must be a struct",
		})
		return results
	}
	
	for _, fieldName := range requiredFields {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			results = append(results, ValidationResult{
				Field:   fieldName,
				Valid:   false,
				Message: "field does not exist",
			})
			continue
		}
		
		if cv.isZeroValue(field) {
			results = append(results, ValidationResult{
				Field:   fieldName,
				Valid:   false,
				Message: "required field is empty",
			})
		} else {
			results = append(results, ValidationResult{
				Field: fieldName,
				Valid: true,
			})
		}
	}
	
	return results
}

func (cv *ConfigValidator) ValidateStringLength(value string, minLength, maxLength int, fieldName string) ValidationResult {
	length := len(value)
	
	if minLength > 0 && length < minLength {
		return ValidationResult{
			Field:   fieldName,
			Valid:   false,
			Message: fmt.Sprintf("must be at least %d characters long", minLength),
		}
	}
	
	if maxLength > 0 && length > maxLength {
		return ValidationResult{
			Field:   fieldName,
			Valid:   false,
			Message: fmt.Sprintf("must not exceed %d characters", maxLength),
		}
	}
	
	return ValidationResult{
		Field: fieldName,
		Valid: true,
	}
}

func (cv *ConfigValidator) ValidateRange(value, min, max int, fieldName string) ValidationResult {
	if value < min {
		return ValidationResult{
			Field:   fieldName,
			Valid:   false,
			Message: fmt.Sprintf("must be at least %d", min),
		}
	}
	
	if value > max {
		return ValidationResult{
			Field:   fieldName,
			Valid:   false,
			Message: fmt.Sprintf("must not exceed %d", max),
		}
	}
	
	return ValidationResult{
		Field: fieldName,
		Valid: true,
	}
}

func (cv *ConfigValidator) ValidateURL(url string, fieldName string) ValidationResult {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ValidationResult{
			Field:   fieldName,
			Valid:   false,
			Message: "must be a valid URL starting with http:// or https://",
		}
	}
	
	return ValidationResult{
		Field: fieldName,
		Valid: true,
	}
}

func (cv *ConfigValidator) ValidateFilePath(path string, fieldName string, mustExist bool) ValidationResult {
	if path == "" {
		return ValidationResult{
			Field:   fieldName,
			Valid:   false,
			Message: "file path cannot be empty",
		}
	}
	
	if mustExist {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return ValidationResult{
				Field:   fieldName,
				Valid:   false,
				Message: "file does not exist",
			}
		}
	}
	
	return ValidationResult{
		Field: fieldName,
		Valid: true,
	}
}

func (cv *ConfigValidator) isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// ValidationResult represents a configuration validation result
type ValidationResult struct {
	Field   string
	Valid   bool
	Message string
}

// ConfigLoader provides common configuration loading patterns
type ConfigLoader struct {
	searchPaths []string
}

func NewConfigLoader(searchPaths []string) *ConfigLoader {
	return &ConfigLoader{
		searchPaths: searchPaths,
	}
}

func (cl *ConfigLoader) FindConfigFile(filename string) (string, error) {
	for _, searchPath := range cl.searchPaths {
		fullPath := filepath.Join(searchPath, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	
	return "", fmt.Errorf("config file %s not found in search paths: %v", filename, cl.searchPaths)
}

func (cl *ConfigLoader) LoadJSONConfig(filename string, config interface{}) error {
	configPath, err := cl.FindConfigFile(filename)
	if err != nil {
		return err
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	
	return nil
}

func (cl *ConfigLoader) SaveJSONConfig(filename string, config interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	configPath := filepath.Join(cl.searchPaths[0], filename)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}
	
	return nil
}

// ConfigMerger provides common configuration merging patterns
type ConfigMerger struct{}

func NewConfigMerger() *ConfigMerger {
	return &ConfigMerger{}
}

func (cm *ConfigMerger) MergeConfigs(base, override interface{}) interface{} {
	baseVal := reflect.ValueOf(base)
	overrideVal := reflect.ValueOf(override)
	
	// Handle pointers
	if baseVal.Kind() == reflect.Ptr {
		baseVal = baseVal.Elem()
	}
	if overrideVal.Kind() == reflect.Ptr {
		overrideVal = overrideVal.Elem()
	}
	
	if baseVal.Kind() != reflect.Struct || overrideVal.Kind() != reflect.Struct {
		return override
	}
	
	result := reflect.New(baseVal.Type()).Elem()
	
	// Copy base values
	for i := 0; i < baseVal.NumField(); i++ {
		field := result.Field(i)
		if field.CanSet() {
			field.Set(baseVal.Field(i))
		}
	}
	
	// Override with non-zero values from override
	for i := 0; i < overrideVal.NumField(); i++ {
		overrideField := overrideVal.Field(i)
		if !cm.isZeroValue(overrideField) {
			resultField := result.Field(i)
			if resultField.CanSet() {
				resultField.Set(overrideField)
			}
		}
	}
	
	return result.Interface()
}

func (cm *ConfigMerger) isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// ConfigEnvironment provides environment variable handling patterns
type ConfigEnvironment struct {
	prefix string
}

func NewConfigEnvironment(prefix string) *ConfigEnvironment {
	return &ConfigEnvironment{
		prefix: prefix,
	}
}

func (ce *ConfigEnvironment) GetEnvKey(configKey string) string {
	return fmt.Sprintf("%s_%s", ce.prefix, strings.ToUpper(configKey))
}

func (ce *ConfigEnvironment) LoadFromEnv(config interface{}) error {
	val := reflect.ValueOf(config)
	
	// Handle pointer to struct
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct or pointer to struct")
	}
	
	typ := val.Type()
	
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		if !field.CanSet() {
			continue
		}
		
		envKey := ce.GetEnvKey(fieldType.Name)
		envValue := os.Getenv(envKey)
		
		if envValue == "" {
			continue
		}
		
		if err := ce.setFieldFromString(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", fieldType.Name, envKey, err)
		}
	}
	
	return nil
}

func (ce *ConfigEnvironment) setFieldFromString(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intValue int64
		if _, err := fmt.Sscanf(value, "%d", &intValue); err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintValue uint64
		if _, err := fmt.Sscanf(value, "%d", &uintValue); err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		var floatValue float64
		if _, err := fmt.Sscanf(value, "%f", &floatValue); err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue := strings.ToLower(value) == "true" || value == "1"
		field.SetBool(boolValue)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	
	return nil
}