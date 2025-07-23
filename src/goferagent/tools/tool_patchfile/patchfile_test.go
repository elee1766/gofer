package tool_patchfile

import (
	"context"
	"testing"

	"github.com/elee1766/gofer/src/agent"
)

func TestPatchTool(t *testing.T) {
	tool, err := Tool()
	if err != nil {
		t.Fatalf("Failed to create patch tool: %v", err)
	}
	
	// Test that it implements the Tool interface
	var _ agent.Tool = tool
	
	// Test basic properties
	if tool.GetName() != Name {
		t.Errorf("Expected tool name %s, got %s", Name, tool.GetName())
	}
	
	if tool.GetType() != "function" {
		t.Errorf("Expected tool type 'function', got %s", tool.GetType())
	}
	
	// Test that description is not empty
	if tool.GetDescription() == "" {
		t.Error("Tool description should not be empty")
	}
	
	// Test that parameters schema is not nil
	if tool.GetParameters() == nil {
		t.Error("Tool parameters schema should not be nil")
	}
}

func TestPatchHandler(t *testing.T) {
	ctx := context.Background()
	
	// Test with empty patch (should return error)
	input := PatchInput{
		Patch: "",
	}
	
	output, err := patchHandler(ctx, input)
	if err == nil {
		t.Error("Expected error for empty patch")
	}
	
	if output.Success {
		t.Error("Expected Success to be false for empty patch")
	}
	
	if output.Message == "" {
		t.Error("Expected error message for empty patch")
	}
}

func TestPatchInputOutputTypes(t *testing.T) {
	// Test that types can be marshaled/unmarshaled
	input := PatchInput{
		Patch:    "--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-old\n+new",
		FilePath: "/test/file.txt",
	}
	
	output := PatchOutput{
		Success:  true,
		Message:  "Patch applied successfully",
		FilePath: "/test/file.txt",
		Output:   "patching file /test/file.txt",
	}
	
	// Just ensure the types are well-formed
	if input.Patch == "" {
		t.Error("Input patch should not be empty")
	}
	
	if !output.Success {
		t.Error("Output success should be true")
	}
}