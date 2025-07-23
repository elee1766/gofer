package tool_patchfile

import (
	"encoding/json"
	"testing"
)

func TestPatchToolSchemaGeneration(t *testing.T) {
	tool, err := Tool()
	if err != nil {
		t.Fatalf("Failed to create patch tool: %v", err)
	}
	
	// Get the parameters schema
	schema := tool.GetParameters()
	if schema == nil {
		t.Fatal("Schema should not be nil")
	}
	
	// Convert to JSON to inspect
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}
	
	schemaStr := string(schemaJSON)
	t.Logf("Generated schema:\n%s", schemaStr)
	
	// Verify required fields
	if !contains(schemaStr, "patch") {
		t.Error("Schema should contain 'patch' property")
	}
	
	if !contains(schemaStr, "file_path") {
		t.Error("Schema should contain 'file_path' property")
	}
	
	// The swaggest reflector generates basic schemas without descriptions
	// This is expected behavior - the GenericTool provides type safety
	// and the descriptions are available via tool.GetDescription()
	t.Logf("Tool description: %s", tool.GetDescription())
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}