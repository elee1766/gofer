package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"syscall"
	"time"
)

// ExecuteCommandSimple runs a command in the persistent shell with simplified output handling
func (ps *PersistentShell) ExecuteCommandSimple(ctx context.Context, command string, timeout time.Duration) (*ShellResult, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return nil, fmt.Errorf("shell session is closed")
	}

	// Check if shell process is still alive
	if ps.cmd.Process != nil {
		if err := ps.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			ps.logger.Error("shell process is dead", "error", err)
			ps.closed = true
			return nil, fmt.Errorf("shell process has died")
		}
	}

	ps.logger.Info("executing command", "command", command, "session_id", ps.sessionID)

	// Set up timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create a simple command wrapper that captures exit code
	endMarker := fmt.Sprintf("__END_%d__", time.Now().UnixNano())
	wrappedCommand := fmt.Sprintf("%s\necho \"EXIT_CODE:$?:%s\"\n", command, endMarker)

	// Write command
	if _, err := ps.stdin.Write([]byte(wrappedCommand)); err != nil {
		ps.logger.Error("failed to write command", "error", err)
		ps.closed = true
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Read output
	outputChan := make(chan *ShellResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(ps.stdout)
		errReader := bufio.NewReader(ps.stderr)
		
		var output strings.Builder
		var errorOutput strings.Builder
		exitCode := 0
		
		// Read until we see our end marker
		for {
			select {
			case <-ctx.Done():
				errorChan <- fmt.Errorf("timeout reading output")
				return
			default:
			}

			// Read stdout line
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				errorChan <- fmt.Errorf("error reading stdout: %w", err)
				return
			}

			// Check for our exit code marker
			if strings.Contains(line, "EXIT_CODE:") && strings.Contains(line, endMarker) {
				// Parse exit code
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					fmt.Sscanf(parts[1], "%d", &exitCode)
				}
				break
			}

			// Regular output
			output.WriteString(line)

			// Non-blocking read from stderr
			if errReader.Buffered() > 0 {
				errLine, _ := errReader.ReadString('\n')
				if errLine != "" {
					errorOutput.WriteString(errLine)
				}
			}
		}

		// Trim the final newline if present
		outStr := strings.TrimSuffix(output.String(), "\n")
		errStr := strings.TrimSuffix(errorOutput.String(), "\n")

		outputChan <- &ShellResult{
			Output:      outStr,
			Error:       errStr,
			ExitCode:    exitCode,
			CommandLine: command,
		}
	}()

	// Wait for result
	select {
	case result := <-outputChan:
		// After command execution, update the working directory
		ps.mu.Unlock() // Unlock before calling UpdateWorkingDirectory
		if err := ps.updateWorkingDirectoryInternal(ctx); err != nil {
			ps.logger.Warn("failed to update working directory", "error", err)
		}
		ps.mu.Lock() // Re-lock

		result.WorkingDir = ps.currentDir
		ps.logger.Info("command completed", "command", command, "exit_code", result.ExitCode, "working_dir", ps.currentDir)
		return result, nil

	case err := <-errorChan:
		ps.logger.Error("command execution error", "command", command, "error", err)
		return nil, err

	case <-ctx.Done():
		ps.logger.Error("command timed out", "command", command, "timeout", timeout)
		return nil, fmt.Errorf("command timed out after %v", timeout)
	}
}

// updateWorkingDirectoryInternal is an internal version that doesn't lock the mutex
func (ps *PersistentShell) updateWorkingDirectoryInternal(ctx context.Context) error {
	if ps.closed {
		return fmt.Errorf("shell session is closed")
	}

	// Use a simple pwd command with a unique marker
	marker := fmt.Sprintf("__PWD_%d__", time.Now().UnixNano())
	pwdCommand := fmt.Sprintf("pwd\necho '%s'\n", marker)

	// Write command
	if _, err := ps.stdin.Write([]byte(pwdCommand)); err != nil {
		ps.logger.Error("failed to write pwd command", "error", err)
		return fmt.Errorf("failed to write pwd command: %w", err)
	}

	// Read response with short timeout
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	reader := bufio.NewReader(ps.stdout)
	var pwd string

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout getting working directory")
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading pwd: %w", err)
		}

		line = strings.TrimSpace(line)
		
		// If we see our marker, we're done
		if line == marker {
			if pwd != "" {
				if pwd != ps.currentDir {
					ps.logger.Info("working directory updated", "old", ps.currentDir, "new", pwd)
					ps.currentDir = pwd
				}
				return nil
			}
		} else if line != "" && !strings.Contains(line, marker) {
			// This should be the pwd output
			pwd = line
		}
	}
}