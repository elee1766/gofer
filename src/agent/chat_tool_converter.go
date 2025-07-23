package agent

import (
	"github.com/elee1766/gofer/src/aisdk"
)

// ToChatTool converts a Tool interface to ChatTool for API requests
func ToChatTool(tool Tool) *aisdk.ChatTool {
	return &aisdk.ChatTool{
		Type: tool.GetType(),
		Function: aisdk.ChatToolFunction{
			Name:        tool.GetName(),
			Description: tool.GetDescription(),
			Parameters:  tool.GetParameters(),
		},
	}
}

// ToChatTools converts a slice of Tool interfaces to ChatTools
func ToChatTools(tools []Tool) []*aisdk.ChatTool {
	chatTools := make([]*aisdk.ChatTool, len(tools))
	for i, tool := range tools {
		chatTools[i] = ToChatTool(tool)
	}
	return chatTools
}