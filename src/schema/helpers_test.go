package schema

import (
	"testing"

	jsonschema "github.com/swaggest/jsonschema-go"
)

func TestCreateStringSchema(t *testing.T) {
	schema := CreateStringSchema("test description")
	
	if schema == nil {
		t.Fatal("Expected schema to be non-nil")
	}
	
	if schema.Description == nil || *schema.Description != "test description" {
		t.Errorf("Expected description 'test description', got %v", schema.Description)
	}
	
	if schema.Type == nil || schema.Type.SimpleTypes == nil {
		t.Fatal("Expected type to be set")
	}
	
	expectedType := jsonschema.SimpleType("string")
	if *schema.Type.SimpleTypes != expectedType {
		t.Errorf("Expected type 'string', got %v", *schema.Type.SimpleTypes)
	}
}

func TestCreateBoolSchema(t *testing.T) {
	schema := CreateBoolSchema("test bool", true)
	
	if schema == nil {
		t.Fatal("Expected schema to be non-nil")
	}
	
	if schema.Description == nil || *schema.Description != "test bool" {
		t.Errorf("Expected description 'test bool', got %v", schema.Description)
	}
	
	if schema.Type == nil || schema.Type.SimpleTypes == nil {
		t.Fatal("Expected type to be set")
	}
	
	expectedType := jsonschema.SimpleType("boolean")
	if *schema.Type.SimpleTypes != expectedType {
		t.Errorf("Expected type 'boolean', got %v", *schema.Type.SimpleTypes)
	}
	
	if schema.Default == nil || *schema.Default != true {
		t.Errorf("Expected default true, got %v", schema.Default)
	}
}

func TestCreateIntSchema(t *testing.T) {
	schema := CreateIntSchema("test int", 42)
	
	if schema == nil {
		t.Fatal("Expected schema to be non-nil")
	}
	
	if schema.Description == nil || *schema.Description != "test int" {
		t.Errorf("Expected description 'test int', got %v", schema.Description)
	}
	
	if schema.Type == nil || schema.Type.SimpleTypes == nil {
		t.Fatal("Expected type to be set")
	}
	
	expectedType := jsonschema.SimpleType("integer")
	if *schema.Type.SimpleTypes != expectedType {
		t.Errorf("Expected type 'integer', got %v", *schema.Type.SimpleTypes)
	}
	
	if schema.Default == nil || *schema.Default != 42 {
		t.Errorf("Expected default 42, got %v", schema.Default)
	}
}

func TestCreateObjectSchema(t *testing.T) {
	properties := map[string]*jsonschema.Schema{
		"name": CreateStringSchema("The name"),
		"age":  CreateIntSchema("The age", 0),
	}
	required := []string{"name"}
	
	schema := CreateObjectSchema(properties, required)
	
	if schema == nil {
		t.Fatal("Expected schema to be non-nil")
	}
	
	if schema.Type == nil || schema.Type.SimpleTypes == nil {
		t.Fatal("Expected type to be set")
	}
	
	expectedType := jsonschema.SimpleType("object")
	if *schema.Type.SimpleTypes != expectedType {
		t.Errorf("Expected type 'object', got %v", *schema.Type.SimpleTypes)
	}
	
	if len(schema.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
	}
	
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("Expected required field 'name', got %v", schema.Required)
	}
}