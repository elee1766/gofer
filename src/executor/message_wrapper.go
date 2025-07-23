package executor

import (
	"fmt"
	"strings"
)

// ConversationState represents the current state of a conversation for message wrapping
type ConversationState struct {
	// Whether this is the first message in the conversation
	IsFirstMessage bool
	
	// Number of turns remaining
	TurnsRemaining int
	
	// Whether tools are enabled
	ToolsEnabled bool
	
	// Whether the previous step involved tool calls
	PreviousStepHadToolCalls bool
	
	// Whether we're continuing after tool execution
	ContinuingAfterToolExecution bool
}

// WrapUserMessage wraps a user message with contextual information based on conversation state
func WrapUserMessage(originalMessage string, state ConversationState) string {
	// If no wrapping needed, return original
	if !needsWrapping(state) {
		return originalMessage
	}
	
	var contextSections []string
	
	// Add turn context
	if state.TurnsRemaining > 1 {
		var turnContext strings.Builder
		turnContext.WriteString(fmt.Sprintf("# Turn Information\n"))
		turnContext.WriteString(fmt.Sprintf("You have %d turns remaining to complete this task.\n", state.TurnsRemaining))
		
		if state.IsFirstMessage {
			turnContext.WriteString("\nUse multiple turns to complete complex tasks autonomously - ")
			turnContext.WriteString("for example, if asked to edit a file, first read it to understand the content, then make the edit. ")
			turnContext.WriteString("Do not ask for permission between these steps when the intent is clear.")
		} else if state.ContinuingAfterToolExecution {
			turnContext.WriteString("\nContinue with the next steps to complete the user's original request. ")
			turnContext.WriteString("If you need to perform additional operations (like reading more files or making more edits), ")
			turnContext.WriteString("proceed without asking for confirmation.")
		}
		
		contextSections = append(contextSections, turnContext.String())
	}
	
	// Add tool usage guidance
	if state.ToolsEnabled && state.IsFirstMessage {
		toolContext := `# Tool Usage Guidelines
When working with files:
- Always read a file before editing it (use read_file then edit_file)
- Use list_directory to explore unfamiliar directory structures
- Be proactive in gathering context you need to complete the task`
		contextSections = append(contextSections, toolContext)
	}
	
	// Add continuation context
	if state.ContinuingAfterToolExecution {
		var contContext strings.Builder
		contContext.WriteString("# Execution Status\n")
		contContext.WriteString("Previous tool execution completed. ")
		if state.TurnsRemaining > 1 {
			contContext.WriteString("Continue with next steps to fulfill the user's request.")
		} else {
			contContext.WriteString("This is your final turn - complete the task.")
		}
		contextSections = append(contextSections, contContext.String())
	}
	
	// Build the wrapped message
	var result strings.Builder
	
	// Add system reminder with context
	if len(contextSections) > 0 {
		result.WriteString("<system-reminder>\n")
		result.WriteString("As you answer the user's questions, you can use the following context:\n")
		
		for i, section := range contextSections {
			if i > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(section)
		}
		
		result.WriteString("\n\n")
		result.WriteString("IMPORTANT: this context may or may not be relevant to your tasks. ")
		result.WriteString("You should not respond to this context or otherwise consider it in your response ")
		result.WriteString("unless it is highly relevant to your task. Most of the time, it is not relevant.\n")
		result.WriteString("</system-reminder>\n")
	}
	
	// Add the original message
	result.WriteString(originalMessage)
	
	return result.String()
}

// needsWrapping determines if the message needs wrapping
func needsWrapping(state ConversationState) bool {
	// Don't wrap if we have no context to add
	if state.TurnsRemaining <= 1 && !state.IsFirstMessage && !state.ContinuingAfterToolExecution {
		return false
	}
	return true
}

// WrapFirstMessage wraps the very first user message in a conversation
func WrapFirstMessage(originalMessage string, turnsRemaining int, toolsEnabled bool) string {
	return WrapUserMessage(originalMessage, ConversationState{
		IsFirstMessage:               true,
		TurnsRemaining:               turnsRemaining,
		ToolsEnabled:                 toolsEnabled,
		PreviousStepHadToolCalls:     false,
		ContinuingAfterToolExecution: false,
	})
}

// WrapContinuationMessage wraps a message for continuing after tool execution
func WrapContinuationMessage(turnsRemaining int, toolsEnabled bool) string {
	// For continuation messages, we don't have new user content
	// Instead, we create a message that prompts the model to continue
	return WrapUserMessage("", ConversationState{
		IsFirstMessage:               false,
		TurnsRemaining:               turnsRemaining,
		ToolsEnabled:                 toolsEnabled,
		PreviousStepHadToolCalls:     true,
		ContinuingAfterToolExecution: true,
	})
}