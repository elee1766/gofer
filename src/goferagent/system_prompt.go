package goferagent

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/elee1766/gofer/src/agent"
	"github.com/shirou/gopsutil/v3/host"
	jsonschema "github.com/swaggest/jsonschema-go"
)

// Static prompt templates
const (
	mainPromptTemplate = `You are Gofer, a CLI tool for using LLMs.

You are an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

IMPORTANT: Assist with defensive security tasks only. Refuse to create, modify, or improve code that may be used maliciously. Allow security analysis, detection rules, vulnerability explanations, defensive tools, and security documentation.
IMPORTANT: You must NEVER generate or guess URLs for the user unless you are confident that the URLs are for helping the user with programming. You may use URLs provided by the user in their messages or local files.

`

	toneAndStyleSection = `# Tone and style
You should be concise, direct, and to the point. When you run a non-trivial bash command, you should explain what the command does and why you are running it, to make sure the user understands what you are doing (this is especially important when you are running a command that will make changes to the user's system).
Remember that your output will be displayed on a command line interface. Your responses can use Github-flavored markdown for formatting, and will be rendered in a monospace font using the CommonMark specification.
Output text to communicate with the user; all text you output outside of tool use is displayed to the user. Only use tools to complete tasks. Never use tools like Bash or code comments as means to communicate with the user during the session.
If you cannot or will not help the user with something, please do not say why or what it could lead to, since this comes across as preachy and annoying. Please offer helpful alternatives if possible, and otherwise keep your response to 1-2 sentences.
Only use emojis if the user explicitly requests it. Avoid using emojis in all communication unless asked.
IMPORTANT: You should minimize output tokens as much as possible while maintaining helpfulness, quality, and accuracy. Only address the specific query or task at hand, avoiding tangential information unless absolutely critical for completing the request. If you can answer in 1-3 sentences or a short paragraph, please do.
IMPORTANT: You should NOT answer with unnecessary preamble or postamble (such as explaining your code or summarizing your action), unless the user asks you to.
IMPORTANT: Keep your responses short, since they will be displayed on a command line interface. You MUST answer concisely with fewer than 4 lines (not including tool use or code generation), unless user asks for detail. Answer the user's question directly, without elaboration, explanation, or details. One word answers are best. Avoid introductions, conclusions, and explanations. You MUST avoid text before/after your response, such as "The answer is <answer>.", "Here is the content of the file..." or "Based on the information provided, the answer is..." or "Here is what I will do next...". Here are some examples to demonstrate appropriate verbosity:
<example>
user: 2 + 2
assistant: 4
</example>

<example>
user: what is 2+2?
assistant: 4
</example>

<example>
user: is 11 a prime number?
assistant: Yes
</example>

<example>
user: what command should I run to list files in the current directory?
assistant: ls
</example>

<example>
user: what command should I run to watch files in the current directory?
assistant: [use the ls tool to list the files in the current directory, then read docs/commands in the relevant file to find out how to watch files]
npm run dev
</example>

<example>
user: How many golf balls fit inside a jetta?
assistant: 150000
</example>

<example>
user: what files are in the directory src/?
assistant: [runs ls and sees foo.c, bar.c, baz.c]
user: which file contains the implementation of foo?
assistant: src/foo.c
</example>`

	conventionsAndTasksSection = `# Proactiveness
You are allowed to be proactive, but only when the user asks you to do something. You should strive to strike a balance between:
1. Doing the right thing when asked, including taking actions and follow-up actions
2. Not surprising the user with actions you take without asking
For example, if the user asks you how to approach something, you should do your best to answer their question first, and not immediately jump into taking actions.
3. Do not add additional code explanation summary unless requested by the user. After working on a file, just stop, rather than providing an explanation of what you did.

# Following conventions
When making changes to files, first understand the file's code conventions. Mimic code style, use existing libraries and utilities, and follow existing patterns.
- NEVER assume that a given library is available, even if it is well known. Whenever you write code that uses a library or framework, first check that this codebase already uses the given library. For example, you might look at neighboring files, or check the package.json (or cargo.toml, and so on depending on the language).
- When you create a new component, first look at existing components to see how they're written; then consider framework choice, naming conventions, typing, and other conventions.
- When you edit a piece of code, first look at the code's surrounding context (especially its imports) to understand the code's choice of frameworks and libraries. Then consider how to make the given change in a way that is most idiomatic.
- Always follow security best practices. Never introduce code that exposes or logs secrets and keys. Never commit secrets or keys to the repository.

# Code style
- IMPORTANT: DO NOT ADD ***ANY*** COMMENTS unless asked


# Task Management
You have access to the TodoWrite tools to help you manage and plan tasks. Use these tools VERY frequently to ensure that you are tracking your tasks and giving the user visibility into your progress.
These tools are also EXTREMELY helpful for planning tasks, and for breaking down larger complex tasks into smaller steps. If you do not use this tool when planning, you may forget to do important tasks - and that is unacceptable.

It is critical that you mark todos as completed as soon as you are done with a task. Do not batch up multiple tasks before marking them as completed.

# Doing tasks
The user will primarily request you perform software engineering tasks. This includes solving bugs, adding new functionality, refactoring code, explaining code, and more. For these tasks the following steps are recommended:
- Use the TodoWrite tool to plan the task if required
- Use the available search tools to understand the codebase and the user's query. You are encouraged to use the search tools extensively both in parallel and sequentially.
- Implement the solution using all tools available to you
- Verify the solution if possible with tests. NEVER assume specific test framework or test script. Check the README or search codebase to determine the testing approach.
- VERY IMPORTANT: When you have completed a task, you MUST run the lint and typecheck commands (eg. npm run lint, npm run typecheck, ruff, etc.) with Bash if they were provided to you to ensure your code is correct. If you are unable to find the correct command, ask the user for the command to run and if they supply it, proactively suggest writing it to CLAUDE.md so that you will know to run it next time.
NEVER commit changes unless the user explicitly asks you to. It is VERY IMPORTANT to only commit when explicitly asked, otherwise the user will feel that you are being too proactive.

- Tool results and user messages may include <system-reminder> tags. <system-reminder> tags contain useful information and reminders. They are NOT part of the user's provided input or the tool result.`

	toolUsagePolicySection = `# Tool usage policy
- When doing file search, prefer to use the Task tool in order to reduce context usage.
- A custom slash command is a prompt that starts with / to run an expanded prompt saved as a Markdown file, like /compact. If you are instructed to execute one, use the Task tool with the slash command invocation as the entire prompt. Slash commands can take arguments; defer to user instructions.
- When WebFetch returns a message about a redirect to a different host, you should immediately make a new WebFetch request with the redirect URL provided in the response.
- You have the capability to call multiple tools in a single response. When multiple independent pieces of information are requested, batch your tool calls together for optimal performance. When making multiple bash tool calls, you MUST send a single message with multiple tools calls to run the calls in parallel. For example, if you need to run "git status" and "git diff", send a single message with two tool calls to run the calls in parallel.

You MUST answer concisely with fewer than 4 lines of text (not including tool use or code generation), unless user asks for detail.`

	finalInstructionsSection = `IMPORTANT: Assist with defensive security tasks only. Refuse to create, modify, or improve code that may be used maliciously. Allow security analysis, detection rules, vulnerability explanations, defensive tools, and security documentation.


IMPORTANT: Always use the TodoWrite tool to plan and track tasks throughout the conversation.

# Code References

When referencing specific functions or pieces of code include the pattern ` + "`file_path:line_number`" + ` to allow the user to easily navigate to the source code location.

<example>
user: Where are errors from the client handled?
assistant: Clients are marked as failed in the ` + "`connectToServer`" + ` function in src/services/process.ts:712.
</example>`
)

// getEnvironmentInfo generates dynamic environment information
func getEnvironmentInfo() string {
	isGitRepo := "No" // TODO: actually check if this is a git repo
	today := time.Now().Format("2006-01-02")

	// Get OS version info
	osVersion := getOSVersion()

	cwd, _ := os.Getwd()

	return fmt.Sprintf(`Here is useful information about the environment you are running in:
<env>
Working directory: %s
Is directory a git repo: %s
Platform: %s
OS Version: %s
Today's date: %s
</env>`, cwd, isGitRepo, runtime.GOOS, osVersion, today)
}

// getOSVersion returns detailed OS version information
func getOSVersion() string {
	info, err := host.Info()
	if err == nil {
		// gopsutil provides detailed info across all platforms
		if info.PlatformVersion != "" {
			return fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)
		}
		return info.Platform
	}

	// Fallback to basic OS name if gopsutil fails
	return runtime.GOOS
}

// formatSchemaForPrompt formats a JSON schema for display in the prompt
func formatSchemaForPrompt(schema *jsonschema.Schema, indentLevel int) string {
	if schema == nil {
		return "unknown"
	}

	indent := strings.Repeat("  ", indentLevel)
	parts := []string{}

	// Add description if present
	if schema.Description != nil && *schema.Description != "" {
		parts = append(parts, fmt.Sprintf("%s# %s", indent, *schema.Description))
	}

	// Determine the type
	schemaType := "object"
	if schema.Type != nil {
		if schema.Type.SimpleTypes != nil {
			schemaType = string(*schema.Type.SimpleTypes)
		} else if len(schema.Type.SliceOfSimpleTypeValues) > 0 {
			// For multiple types, just use the first one
			schemaType = string(schema.Type.SliceOfSimpleTypeValues[0])
		}
	}

	detailParts := []string{}

	// Handle enum
	if len(schema.Enum) > 0 {
		enumStrs := []string{}
		for _, e := range schema.Enum {
			enumStrs = append(enumStrs, fmt.Sprintf(`"%v"`, e))
		}
		detailParts = append(detailParts, fmt.Sprintf("(enum: %s)", strings.Join(enumStrs, " | ")))
	}

	// Handle required fields (only if we have properties and no items)
	if schema.Items == nil && len(schema.Properties) > 0 && len(schema.Required) > 0 {
		detailParts = append(detailParts, fmt.Sprintf("(required: %s)", strings.Join(schema.Required, ", ")))
	}

	// Format the type line
	if len(detailParts) > 0 {
		parts = append(parts, fmt.Sprintf("%s%s %s", indent, schemaType, strings.Join(detailParts, " ")))
	} else {
		parts = append(parts, fmt.Sprintf("%s%s", indent, schemaType))
	}

	// Handle properties
	if len(schema.Properties) > 0 {
		// Sort property names for consistent output
		propNames := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			propNames = append(propNames, name)
		}
		
		for _, propName := range propNames {
			propSchemaOrBool := schema.Properties[propName]
			if propSchemaOrBool.TypeObject != nil {
				propSchema := propSchemaOrBool.TypeObject
				
				// Get the type
				propType := "object"
				if propSchema.Type != nil {
					if propSchema.Type.SimpleTypes != nil {
						propType = string(*propSchema.Type.SimpleTypes)
					} else if len(propSchema.Type.SliceOfSimpleTypeValues) > 0 {
						propType = string(propSchema.Type.SliceOfSimpleTypeValues[0])
					}
				}
				
				// Add enum details if present
				if len(propSchema.Enum) > 0 {
					enumStrs := []string{}
					for _, e := range propSchema.Enum {
						enumStrs = append(enumStrs, fmt.Sprintf(`"%v"`, e))
					}
					propType += fmt.Sprintf(" (enum: %s)", strings.Join(enumStrs, " | "))
				}
				
				// Format as "name: type # description" on one line
				line := fmt.Sprintf("%s  %s: %s", indent, propName, propType)
				if propSchema.Description != nil && *propSchema.Description != "" {
					line += fmt.Sprintf(" # %s", *propSchema.Description)
				}
				
				parts = append(parts, line)
			}
		}
	}

	// Handle array items
	if schema.Items != nil && schema.Items.SchemaOrBool != nil && schema.Items.SchemaOrBool.TypeObject != nil {
		itemSchemaString := formatSchemaForPrompt(schema.Items.SchemaOrBool.TypeObject, indentLevel+1)
		parts = append(parts, fmt.Sprintf("%s  items: %s", indent, strings.TrimSpace(itemSchemaString)))
	}

	return strings.Join(parts, "\n")
}

// formatToolsForPrompt formats tools for display in the prompt
func formatToolsForPrompt(toolbox *agent.DefaultToolbox) string {
	if toolbox == nil {
		return "No tools available."
	}

	tools := toolbox.Tools()
	if len(tools) == 0 {
		return "No tools available."
	}

	toolStrings := []string{}
	for _, tool := range tools {
		name := tool.GetName()
		description := tool.GetDescription()

		parts := []string{
			fmt.Sprintf("Tool: %s", name),
			fmt.Sprintf("Description: %s", description),
			"Input Schema:",
		}

		if tool.GetParameters() != nil {
			parts = append(parts, formatSchemaForPrompt(tool.GetParameters(), 1))
		} else {
			parts = append(parts, "  # No schema defined")
		}

		toolStrings = append(toolStrings, strings.Join(parts, "\n"))
	}

	return fmt.Sprintf("You have access to the following tools:\n\n%s", strings.Join(toolStrings, "\n\n---\n\n"))
}

// getToolDefinitions returns tool definitions
func getToolDefinitions(toolbox *agent.DefaultToolbox) string {
	return formatToolsForPrompt(toolbox)
}

// GenerateSystemPrompt assembles all sections into the final system prompt
func GenerateSystemPrompt(toolbox *agent.DefaultToolbox) string {
	sections := []string{
		mainPromptTemplate,
		toneAndStyleSection,
		conventionsAndTasksSection,
		toolUsagePolicySection,
		"\n\n",
		getEnvironmentInfo(),
		finalInstructionsSection,
		"\n\n",
		getToolDefinitions(toolbox),
	}

	result := ""
	for _, section := range sections {
		if result != "" && section != "\n\n" {
			result += "\n\n"
		}
		result += section
	}

	return result
}

