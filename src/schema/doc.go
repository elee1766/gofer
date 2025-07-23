// Package schema provides helper functions for creating JSON Schema definitions.
//
// This package contains utilities for creating JSON Schema objects used in
// tool parameter definitions and API specifications. It provides type-safe
// convenience functions for common schema patterns.
//
// Example usage:
//
//	import "github.com/elee1766/gofer/src/schema"
//
//	// Create a simple string schema
//	nameSchema := schema.CreateStringSchema("The user's name")
//
//	// Create an object schema with properties
//	userSchema := schema.CreateObjectSchema(map[string]*jsonschema.Schema{
//		"name": schema.CreateStringSchema("The user's name"),
//		"age":  schema.CreateIntSchema("The user's age", 0),
//		"active": schema.CreateBoolSchema("Whether user is active", true),
//	}, []string{"name"}) // name is required
package schema