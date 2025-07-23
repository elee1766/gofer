package main

import (
	"fmt"
	"reflect"
	"strings"
	
	"github.com/elee1766/gofer/src/config"
)

// getConfigValue retrieves a configuration value by key using reflection
func getConfigValue(cfg *config.Config, key string) (interface{}, error) {
	v := reflect.ValueOf(cfg).Elem()
	return getNestedValue(v, key)
}

// setConfigValue sets a configuration value by key using reflection
func setConfigValue(cfg *config.Config, key string, value interface{}) error {
	v := reflect.ValueOf(cfg).Elem()
	return setNestedValue(v, key, value)
}

// getNestedValue retrieves a nested value from a struct using dot notation
func getNestedValue(v reflect.Value, key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	
	for _, part := range parts {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		
		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cannot access field %s: not a struct", part)
		}
		
		field := v.FieldByName(part)
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found", part)
		}
		
		v = field
	}
	
	return v.Interface(), nil
}

// setNestedValue sets a nested value in a struct using dot notation
func setNestedValue(v reflect.Value, key string, value interface{}) error {
	parts := strings.Split(key, ".")
	
	for i, part := range parts {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		
		if v.Kind() != reflect.Struct {
			return fmt.Errorf("cannot access field %s: not a struct", part)
		}
		
		field := v.FieldByName(part)
		if !field.IsValid() {
			return fmt.Errorf("field %s not found", part)
		}
		
		if i == len(parts)-1 {
			// Last part, set the value
			if !field.CanSet() {
				return fmt.Errorf("field %s cannot be set", part)
			}
			
			valueV := reflect.ValueOf(value)
			if !valueV.Type().ConvertibleTo(field.Type()) {
				return fmt.Errorf("cannot convert %v to %s", value, field.Type())
			}
			
			field.Set(valueV.Convert(field.Type()))
			return nil
		}
		
		v = field
	}
	
	return nil
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}