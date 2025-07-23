package goferagent

import (
	"github.com/elee1766/gofer/src/agent"
)

// GetDefaultSystemPrompt returns the rendered system prompt with default values
func GetDefaultSystemPrompt(toolbox *agent.DefaultToolbox) string {
	return GenerateSystemPrompt(toolbox)
}
