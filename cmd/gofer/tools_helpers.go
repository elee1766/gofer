package main

import (
	"fmt"
	"log/slog"

	"github.com/elee1766/gofer/src/agent"
	"github.com/spf13/afero"
)

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Status      string `json:"status,omitempty"`
	Available   bool   `json:"available"`
	Installed   bool   `json:"installed"`
}

// GetAllTools returns information about all available tools
func GetAllTools() ([]ToolInfo, error) {
	// Create a temporary toolbox to get all tools
	toolbox, err := createToolbox(slog.Default(), afero.NewOsFs(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolbox: %w", err)
	}

	tools := toolbox.Tools()
	toolInfos := make([]ToolInfo, 0, len(tools))

	for _, tool := range tools {
		// Get tool name and description from the tool function
		var name, description string
		
		switch t := tool.(type) {
		case *agent.LegacyTool:
			name = t.Function.Name
			description = t.Function.Description
		default:
			// For GenericTool and other types, try to get name via interface
			if nameable, ok := tool.(interface{ GetName() string }); ok {
				name = nameable.GetName()
			}
			if describable, ok := tool.(interface{ GetDescription() string }); ok {
				description = describable.GetDescription()
			}
			// Skip if we couldn't get name
			if name == "" {
				continue
			}
		}

		toolInfo := ToolInfo{
			Name:        name,
			Description: description,
			Category:    categorizeToolByName(name),
			Status:      "enabled",
			Available:   true,
			Installed:   true,
		}
		toolInfos = append(toolInfos, toolInfo)
	}

	return toolInfos, nil
}

// categorizeToolByName categorizes a tool based on its name
func categorizeToolByName(name string) string {
	switch name {
	case "read_file", "write_file", "edit_file", "list_directory", 
	     "create_directory", "delete_file", "move_file", "copy_file", 
	     "get_file_info", "search_files", "grep_files":
		return "filesystem"
	case "run_command":
		return "system"
	case "web_fetch":
		return "network"
	case "patch":
		return "development"
	default:
		return "other"
	}
}