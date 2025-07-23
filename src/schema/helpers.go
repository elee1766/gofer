package schema

import (
	jsonschema "github.com/swaggest/jsonschema-go"
)

// Helper functions to create JSON schemas

// CreateStringSchema creates a JSON schema for a string field
func CreateStringSchema(description string) *jsonschema.Schema {
	strType := jsonschema.SimpleType("string")
	return &jsonschema.Schema{
		Type:        &jsonschema.Type{SimpleTypes: &strType},
		Description: &description,
	}
}

// CreateBoolSchema creates a JSON schema for a boolean field with default value
func CreateBoolSchema(description string, defaultValue bool) *jsonschema.Schema {
	boolType := jsonschema.SimpleType("boolean")
	defVal := interface{}(defaultValue)
	return &jsonschema.Schema{
		Type:        &jsonschema.Type{SimpleTypes: &boolType},
		Description: &description,
		Default:     &defVal,
	}
}

// CreateIntSchema creates a JSON schema for an integer field with default value
func CreateIntSchema(description string, defaultValue int) *jsonschema.Schema {
	intType := jsonschema.SimpleType("integer")
	defVal := interface{}(defaultValue)
	return &jsonschema.Schema{
		Type:        &jsonschema.Type{SimpleTypes: &intType},
		Description: &description,
		Default:     &defVal,
	}
}

// CreateObjectSchema creates a JSON schema for an object with properties and required fields
func CreateObjectSchema(properties map[string]*jsonschema.Schema, required []string) *jsonschema.Schema {
	schemaProps := make(map[string]jsonschema.SchemaOrBool)
	for name, prop := range properties {
		schemaProps[name] = jsonschema.SchemaOrBool{TypeObject: prop}
	}
	
	objType := jsonschema.SimpleType("object")
	return &jsonschema.Schema{
		Type:       &jsonschema.Type{SimpleTypes: &objType},
		Properties: schemaProps,
		Required:   required,
	}
}

// CreateStringSchemaEnum creates a JSON schema for a string field with enum values
func CreateStringSchemaEnum(description string, enumValues []string) *jsonschema.Schema {
	strType := jsonschema.SimpleType("string")
	enum := make([]interface{}, len(enumValues))
	for i, v := range enumValues {
		enum[i] = v
	}
	return &jsonschema.Schema{
		Type:        &jsonschema.Type{SimpleTypes: &strType},
		Description: &description,
		Enum:        enum,
	}
}

// CreateIntegerSchema creates a JSON schema for an integer field
func CreateIntegerSchema(description string) *jsonschema.Schema {
	intType := jsonschema.SimpleType("integer")
	return &jsonschema.Schema{
		Type:        &jsonschema.Type{SimpleTypes: &intType},
		Description: &description,
	}
}