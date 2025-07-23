package tool_runcommand

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/elee1766/gofer/src/shell"
)

// Tool name constant
const Name = "run_command"

const runCommandPrompt = `Executes a given bash command in a persistent shell session with optional timeout, ensuring proper handling and security measures.

Before executing the command, please follow these steps:

1. Directory Verification:
   - If the command will create new directories or files, first use the LS tool to verify the parent directory exists and is the correct location
   - For example, before running "mkdir foo/bar", first use LS to check that "foo" exists and is the intended parent directory

2. Command Execution:
   - Always quote file paths that contain spaces with double quotes (e.g., cd "path with spaces/file.txt")
   - Examples of proper quoting:
     - cd "/Users/name/My Documents" (correct)
     - cd /Users/name/My Documents (incorrect - will fail)
     - python "/path/with spaces/script.py" (correct)
     - python /path/with spaces/script.py (incorrect - will fail)
   - After ensuring proper quoting, execute the command.
   - Capture the output of the command.

Usage notes:
  - The command argument is required.
  - You can specify an optional timeout in milliseconds (up to 600000ms / 10 minutes). If not specified, commands will timeout after 120000ms (2 minutes).
  - It is very helpful if you write a clear, concise description of what this command does in 5-10 words.
  - If the output exceeds 30000 characters, output will be truncated before being returned to you.
  - VERY IMPORTANT: You MUST avoid using search commands like ` + "`find`" + ` and ` + "`grep`" + `. Instead use Grep, Glob, or Task to search. You MUST avoid read tools like ` + "`cat`" + `, ` + "`head`" + `, ` + "`tail`" + `, and ` + "`ls`" + `, and use Read and LS to read files.
 - If you _still_ need to run ` + "`grep`" + `, STOP. ALWAYS USE ripgrep at ` + "`rg`" + ` first, which all ${PRODUCT_NAME} users have pre-installed.
  - When issuing multiple commands, use the ';' or '&&' operator to separate them. DO NOT use newlines (newlines are ok in quoted strings).
  - Try to maintain your current working directory throughout the session by using absolute paths and avoiding usage of ` + "`cd`" + `. You may use ` + "`cd`" + ` if the User explicitly requests it.
    <good-example>
    pytest /foo/bar/tests
    </good-example>
    <bad-example>
    cd /foo/bar && pytest tests
    </bad-example>


# Committing changes with git

When the user asks you to create a new git commit, follow these steps carefully:

1. You have the capability to call multiple tools in a single response. When multiple independent pieces of information are requested, batch your tool calls together for optimal performance. ALWAYS run the following bash commands in parallel, each using the Bash tool:
  - Run a git status command to see all untracked files.
  - Run a git diff command to see both staged and unstaged changes that will be committed.
  - Run a git log command to see recent commit messages, so that you can follow this repository's commit message style.
2. Analyze all staged changes (both previously staged and newly added) and draft a commit message:
  - Summarize the nature of the changes (eg. new feature, enhancement to an existing feature, bug fix, refactoring, test, docs, etc.). Ensure the message accurately reflects the changes and their purpose (i.e. "add" means a wholly new feature, "update" means an enhancement to an existing feature, "fix" means a bug fix, etc.).
  - Check for any sensitive information that shouldn't be committed
  - Draft a concise (1-2 sentences) commit message that focuses on the "why" rather than the "what"
  - Ensure it accurately reflects the changes and their purpose
3. You have the capability to call multiple tools in a single response. When multiple independent pieces of information are requested, batch your tool calls together for optimal performance. ALWAYS run the following commands in parallel:
   - Add relevant untracked files to the staging area.
   - Create the commit with a message ending with:
   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   - Run git status to make sure the commit succeeded.
4. If the commit fails due to pre-commit hook changes, retry the commit ONCE to include these automated changes. If it fails again, it usually means a pre-commit hook is preventing the commit. If the commit succeeds but you notice that files were modified by the pre-commit hook, you MUST amend your commit to include them.

Important notes:
- NEVER update the git config
- NEVER run additional commands to read or explore code, besides git bash commands
- NEVER use the TodoWrite or Task tools
- DO NOT push to the remote repository unless the user explicitly asks you to do so
- IMPORTANT: Never use git commands with the -i flag (like git rebase -i or git add -i) since they require interactive input which is not supported.
- If there are no changes to commit (i.e., no untracked files and no modifications), do not create an empty commit
- In order to ensure good formatting, ALWAYS pass the commit message via a HEREDOC, a la this example:
<example>
git commit -m "$(cat <<'EOF'
   Commit message here.

   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   EOF
   )"
</example>

# Creating pull requests
Use the gh command via the Bash tool for ALL GitHub-related tasks including working with issues, pull requests, checks, and releases. If given a Github URL use the gh command to get the information needed.

IMPORTANT: When the user asks you to create a pull request, follow these steps carefully:

1. You have the capability to call multiple tools in a single response. When multiple independent pieces of information are requested, batch your tool calls together for optimal performance. ALWAYS run the following bash commands in parallel using the Bash tool, in order to understand the current state of the branch since it diverged from the main branch:
   - Run a git status command to see all untracked files
   - Run a git diff command to see both staged and unstaged changes that will be committed
   - Check if the current branch tracks a remote branch and is up to date with the remote, so you know if you need to push to the remote
   - Run a git log command and ` + "`git diff [base-branch]...HEAD`" + ` to understand the full commit history for the current branch (from the time it diverged from the base branch)
2. Analyze all changes that will be included in the pull request, making sure to look at all relevant commits (NOT just the latest commit, but ALL commits that will be included in the pull request!!!), and draft a pull request summary
3. You have the capability to call multiple tools in a single response. When multiple independent pieces of information are requested, batch your tool calls together for optimal performance. ALWAYS run the following commands in parallel:
   - Create new branch if needed
   - Push to remote with -u flag if needed
   - Create PR using gh pr create with the format below. Use a HEREDOC to pass the body to ensure correct formatting.
<example>
gh pr create --title "the pr title" --body "$(cat <<'EOF'
## Summary
<1-3 bullet points>

## Test plan
[Checklist of TODOs for testing the pull request...]

🤖 Generated with [Claude Code](https://claude.ai/code)
EOF
)"
</example>

Important:
- NEVER update the git config
- DO NOT use the TodoWrite or Task tools
- Return the PR URL when you're done, so the user can see it

# Other common operations
- View comments on a Github PR: gh api repos/foo/bar/pulls/123/comments`

// Tool returns the run_command tool definition with a shell manager
// RunCommandInput represents the parameters for run_command
type RunCommandInput struct {
	Command    string `json:"command" required:"true" description:"The command to execute"`
	WorkingDir string `json:"working_dir,omitempty" description:"Working directory for the command"`
	Timeout    int    `json:"timeout,omitempty" description:"Timeout in seconds (default 30)"`
}

// RunCommandOutput represents the response from run_command
type RunCommandOutput struct {
	Command    string `json:"command" description:"The command that was executed"`
	ExitCode   int    `json:"exit_code" description:"Exit code of the command"`
	Output     string `json:"output" description:"Combined stdout and stderr output"`
	WorkingDir string `json:"working_dir" description:"Working directory where command was executed"`
	Timeout    bool   `json:"timeout" description:"Whether the command timed out"`
	Duration   string `json:"duration" description:"Time taken to execute the command"`
}

// Tool returns the run_command tool definition using GenericTool
func Tool(shellManager *shell.ShellManager) agent.Tool {
	tool, err := agent.NewGenericTool(Name, runCommandPrompt, makeRunCommandHandler(shellManager))
	if err != nil {
		// This should never happen with a well-formed handler, but we need to handle it
		panic(fmt.Sprintf("failed to create run_command tool: %v", err))
	}
	return tool
}

// makeRunCommandHandler creates a type-safe handler for the run_command tool
func makeRunCommandHandler(shellManager *shell.ShellManager) func(ctx context.Context, input RunCommandInput) (RunCommandOutput, error) {
	return func(ctx context.Context, input RunCommandInput) (RunCommandOutput, error) {
		logger := toolsutil.GetLogger()
		
		// Check for cancellation
		select {
		case <-ctx.Done():
			return RunCommandOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate working directory
		if input.WorkingDir != "" && !toolsutil.IsPathSafe(input.WorkingDir) {
			logger.Error("unsafe working directory rejected", "working_dir", input.WorkingDir)
			return RunCommandOutput{}, fmt.Errorf("unsafe working directory: %s", input.WorkingDir)
		}

		// Check if shell manager is provided
		if shellManager == nil {
			logger.Error("shell manager not provided")
			return RunCommandOutput{}, fmt.Errorf("shell manager not provided")
		}

		// Get current conversation ID
		conversationID := shell.GetConversationContext()
		if conversationID == "" {
			logger.Warn("no conversation context set, using default")
			conversationID = "default"
		}

		// If working directory is specified, change to it first
		command := input.Command
		if input.WorkingDir != "" {
			command = fmt.Sprintf("cd %s && %s", input.WorkingDir, input.Command)
		}

		if input.Timeout == 0 {
			input.Timeout = 30
		}
		
		// Limit timeout to maximum of 5 minutes
		if input.Timeout > 300 {
			input.Timeout = 300
		}

		logger.Info("running command in persistent shell", "command", input.Command, "working_dir", input.WorkingDir, "timeout", input.Timeout, "conversation_id", conversationID)

		// Create context with timeout (use the provided context as parent)
		start := time.Now()
		ctx, cancel := context.WithTimeout(ctx, time.Duration(input.Timeout)*time.Second)
		defer cancel()

		// Execute command using persistent shell
		shellResult, err := shellManager.ExecuteCommand(ctx, conversationID, command, time.Duration(input.Timeout)*time.Second)
		duration := time.Since(start)
		
		result := RunCommandOutput{
			Command:    input.Command,
			WorkingDir: "",
			Duration:   duration.String(),
		}

		if shellResult != nil {
			// Combine stdout and stderr for output
			var output strings.Builder
			if shellResult.Output != "" {
				output.WriteString(shellResult.Output)
			}
			if shellResult.Error != "" {
				if output.Len() > 0 {
					output.WriteString("\n")
				}
				output.WriteString(shellResult.Error)
			}
			
			result.Output = output.String()
			result.ExitCode = shellResult.ExitCode
			result.WorkingDir = shellResult.WorkingDir
		}

		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				logger.Error("command timed out", "command", input.Command, "timeout", input.Timeout)
				result.Timeout = true
				result.ExitCode = 124 // Standard timeout exit code
			} else {
				logger.Error("command failed", "command", input.Command, "error", err)
			}
			
			// Check if the command validation failed (no shellResult means validation error)
			if shellResult == nil {
				// This is a validation error, return as an error
				return RunCommandOutput{}, fmt.Errorf("command validation failed: %v", err)
			}
		} else {
			result.Timeout = false
			logger.Info("command completed successfully", "command", input.Command, "output_size", len(result.Output))
		}

		return result, nil
	}
}
