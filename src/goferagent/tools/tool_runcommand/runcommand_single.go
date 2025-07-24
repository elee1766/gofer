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

// ToolWithSingleShell returns the run_command tool definition using a SingleShellManager
func ToolWithSingleShell(shellManager *shell.SingleShellManager) agent.Tool {
	tool, err := agent.NewGenericTool(Name, runCommandPrompt, makeRunCommandHandlerSingle(shellManager))
	if err != nil {
		// This should never happen with a well-formed handler, but we need to handle it
		panic(fmt.Sprintf("failed to create run_command tool: %v", err))
	}
	return tool
}

// makeRunCommandHandlerSingle creates a type-safe handler for the run_command tool using SingleShellManager
func makeRunCommandHandlerSingle(shellManager *shell.SingleShellManager) func(ctx context.Context, input RunCommandInput) (RunCommandOutput, error) {
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

		logger.Info("running command in persistent shell", "command", input.Command, "working_dir", input.WorkingDir, "timeout", input.Timeout)

		// Create context with timeout (use the provided context as parent)
		start := time.Now()
		ctx, cancel := context.WithTimeout(ctx, time.Duration(input.Timeout)*time.Second)
		defer cancel()

		// Execute command using persistent shell
		shellResult, err := shellManager.ExecuteCommand(ctx, command, time.Duration(input.Timeout)*time.Second)
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