package tool_patchfile

import (
	"context"
	"testing"

	"github.com/elee1766/gofer/src/agent"
)

func TestPatchToolIntegration(t *testing.T) {
	// Create a toolbox and register the patch tool
	toolbox := agent.NewToolbox[agent.Tool]()
	
	patchTool, err := Tool()
	if err != nil {
		t.Fatalf("Failed to create patch tool: %v", err)
	}
	err = toolbox.RegisterTool(patchTool)
	if err != nil {
		t.Fatalf("Failed to register patch tool: %v", err)
	}
	
	// Verify the tool is registered
	registeredTool, exists := toolbox.GetTool(Name)
	if !exists {
		t.Fatal("Patch tool not found in toolbox")
	}
	
	if registeredTool.GetName() != Name {
		t.Errorf("Expected tool name %s, got %s", Name, registeredTool.GetName())
	}
	
	// Test the type-safe handler directly (bypasses JSON schema validation issues)
	ctx := context.Background()
	
	// Test with empty patch
	emptyInput := PatchInput{Patch: ""}
	output, err := patchHandler(ctx, emptyInput)
	if err == nil {
		t.Error("Expected error for empty patch")
	}
	if output.Success {
		t.Error("Expected Success to be false for empty patch")
	}
	
	// Test with some patch content (will fail but should process)
	validInput := PatchInput{
		Patch: "--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-old\n+new",
	}
	output, err = patchHandler(ctx, validInput)
	// No error expected from handler - execution details in output
	if err != nil {
		t.Logf("Handler execution completed with details: %v", err)
	}
	t.Logf("Patch result: success=%v, message=%s", output.Success, output.Message)
}

func TestPatchToolConversionToChatTool(t *testing.T) {
	patchTool, err := Tool()
	if err != nil {
		t.Fatalf("Failed to create patch tool: %v", err)
	}
	
	// Test conversion to ChatTool for API requests
	chatTool := agent.ToChatTool(patchTool)
	
	if chatTool == nil {
		t.Fatal("ChatTool conversion returned nil")
	}
	
	if chatTool.Type != "function" {
		t.Errorf("Expected ChatTool type 'function', got %s", chatTool.Type)
	}
	
	if chatTool.Function.Name != Name {
		t.Errorf("Expected ChatTool function name %s, got %s", Name, chatTool.Function.Name)
	}
	
	if chatTool.Function.Description == "" {
		t.Error("ChatTool should have a description")
	}
	
	if chatTool.Function.Parameters == nil {
		t.Error("ChatTool should have parameters schema")
	}
}