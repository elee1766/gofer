package main

import (
	"testing"
	"github.com/spf13/afero"
)

func TestCreateDefaultToolbox(t *testing.T) {
	// Test creating toolbox without logger
	toolbox, err := createToolbox(nil, afero.NewOsFs())
	if err != nil {
		t.Fatalf("Failed to create default toolbox: %v", err)
	}

	if toolbox == nil {
		t.Fatal("Toolbox should not be nil")
	}

	// Check that tools are registered
	tools := toolbox.Tools()
	if len(tools) == 0 {
		t.Error("Toolbox should have tools registered")
	}

	// Check for specific tools
	expectedTools := []string{
		"read_file",
		"write_file",
		"list_directory",
		"run_command",
		"search_files",
		"edit_file",
		"create_directory",
		"delete_file",
		"move_file",
		"copy_file",
		"get_file_info",
		"grep_files",
		"web_fetch",
		"patch",
	}

	for _, toolName := range expectedTools {
		if !toolbox.HasTool(toolName) {
			t.Errorf("Expected tool %s to be registered", toolName)
		}
	}
}

func TestGetAllTools(t *testing.T) {
	tools, err := GetAllTools()
	if err != nil {
		t.Fatalf("Failed to get all tools: %v", err)
	}

	if len(tools) == 0 {
		t.Error("Should return at least one tool")
	}

	// Check that patch tool is included
	foundPatch := false
	for _, tool := range tools {
		t.Logf("Found tool: %s", tool.Name)
		if tool.Name == "patch" {
			foundPatch = true
			break
		}
	}

	if !foundPatch {
		t.Error("Patch tool should be included in all tools")
	}
}