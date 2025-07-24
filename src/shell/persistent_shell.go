package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// PersistentShell maintains a shell session with current directory tracking
type PersistentShell struct {
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	currentDir    string
	originalDir   string
	sessionID     string
	mu            sync.Mutex
	logger        *slog.Logger
	closed        bool
}

// ShellResult represents the result of a shell command
type ShellResult struct {
	Output      string
	Error       string
	ExitCode    int
	WorkingDir  string
	CommandLine string
}

// NewPersistentShell creates a new persistent shell session
func NewPersistentShell(logger *slog.Logger) (*PersistentShell, error) {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create shell command with bash in a more robust way
	cmd := exec.Command("bash", "--norc", "--noprofile", "-s")
	cmd.Dir = currentDir
	
	// Set up environment to minimize interference
	cmd.Env = append(os.Environ(),
		"PS1=", // Disable prompt to avoid interference
		"PS2=", // Disable secondary prompt
		"PS4=", // Disable xtrace prompt
		"PROMPT_COMMAND=", // Disable prompt command
		"TERM=dumb", // Simple terminal
		"BASH_ENV=", // Don't source any files
	)

	// Create pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the shell
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	sessionID := fmt.Sprintf("shell_%d", cmd.Process.Pid)
	
	shell := &PersistentShell{
		cmd:         cmd,
		stdin:       stdin,
		stdout:      stdout,
		stderr:      stderr,
		currentDir:  currentDir,
		originalDir: currentDir,
		sessionID:   sessionID,
		logger:      logger,
		closed:      false,
	}

	shell.logger.Info("starting persistent shell session", "session_id", sessionID, "working_dir", currentDir)
	
	// Initialize shell with basic settings
	initCommands := []string{
		"set -u", // Error on undefined variables
		"export LC_ALL=C", // Consistent locale
		"export LANG=C",
		"unset HISTFILE", // Don't save history
		"set +o history", // Disable history
	}
	
	for _, cmd := range initCommands {
		if _, err := shell.stdin.Write([]byte(cmd + "\n")); err != nil {
			shell.Close()
			return nil, fmt.Errorf("failed to initialize shell with %s: %w", cmd, err)
		}
	}
	
	return shell, nil
}

// ExecuteCommand runs a command in the persistent shell
func (ps *PersistentShell) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (*ShellResult, error) {
	// Use the simpler implementation
	return ps.ExecuteCommandSimple(ctx, command, timeout)
}

// ExecuteCommandOld runs a command in the persistent shell (old complex implementation)
func (ps *PersistentShell) ExecuteCommandOld(ctx context.Context, command string, timeout time.Duration) (*ShellResult, error) {
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

	ps.logger.Info("executing command in persistent shell", "command", command, "session_id", ps.sessionID)

	// Create a unique delimiter for this command
	delimiter := fmt.Sprintf("__GOFER_%d__", time.Now().UnixNano())
	
	// Build command wrapper that ensures we get output and exit code
	wrappedCommand := fmt.Sprintf(`
# Execute command and capture result
__gofer_output=$(mktemp)
__gofer_error=$(mktemp)
__gofer_exit=0

# Run the actual command
{ %s; } > "$__gofer_output" 2> "$__gofer_error"
__gofer_exit=$?

# Output results with delimiters
echo "%s:START"
cat "$__gofer_output"
echo "%s:STDOUT_END"
cat "$__gofer_error" >&2
echo "%s:STDERR_END"
echo "%s:EXIT:$__gofer_exit"
echo "%s:PWD:$(pwd)"
echo "%s:DONE"

# Cleanup
rm -f "$__gofer_output" "$__gofer_error"
`, command, delimiter, delimiter, delimiter, delimiter, delimiter, delimiter)

	// Set up timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Send command
	if _, err := ps.stdin.Write([]byte(wrappedCommand)); err != nil {
		ps.logger.Error("failed to write command", "error", err)
		ps.closed = true
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Read response
	type readResult struct {
		output     string
		errorOut   string
		exitCode   int
		workingDir string
		err        error
	}

	resultChan := make(chan readResult, 1)

	go func() {
		var output strings.Builder
		var errorOut strings.Builder
		var exitCode int
		var workingDir string
		
		stdoutReader := bufio.NewReader(ps.stdout)
		stderrReader := bufio.NewReader(ps.stderr)
		
		readingOutput := false
		readingError := false
		done := false
		
		for !done {
			select {
			case <-cmdCtx.Done():
				resultChan <- readResult{err: fmt.Errorf("timeout reading output")}
				return
			default:
			}

			// Read stdout line
			line, err := stdoutReader.ReadString('\n')
			if err != nil && err != io.EOF {
				resultChan <- readResult{err: fmt.Errorf("error reading stdout: %w", err)}
				return
			}
			
			line = strings.TrimRight(line, "\n\r")
			
			switch {
			case line == delimiter+":START":
				readingOutput = true
			case line == delimiter+":STDOUT_END":
				readingOutput = false
			case line == delimiter+":STDERR_END":
				readingError = false
			case strings.HasPrefix(line, delimiter+":EXIT:"):
				parts := strings.Split(line, ":")
				if len(parts) >= 3 {
					fmt.Sscanf(parts[2], "%d", &exitCode)
				}
			case strings.HasPrefix(line, delimiter+":PWD:"):
				parts := strings.SplitN(line, ":", 3)
				if len(parts) >= 3 {
					workingDir = parts[2]
				}
			case line == delimiter+":DONE":
				done = true
			default:
				if readingOutput {
					if output.Len() > 0 {
						output.WriteString("\n")
					}
					output.WriteString(line)
				}
			}

			// Try to read any stderr
			if stderrReader.Buffered() > 0 {
				errLine, _ := stderrReader.ReadString('\n')
				errLine = strings.TrimRight(errLine, "\n\r")
				if errLine == delimiter+":STDERR_END" {
					readingError = false
				} else if readingError || errLine != "" {
					if errorOut.Len() > 0 {
						errorOut.WriteString("\n")
					}
					errorOut.WriteString(errLine)
					readingError = true
				}
			}
		}

		resultChan <- readResult{
			output:     output.String(),
			errorOut:   errorOut.String(),
			exitCode:   exitCode,
			workingDir: workingDir,
		}
	}()

	// Wait for result
	select {
	case result := <-resultChan:
		if result.err != nil {
			ps.logger.Error("command execution error", "command", command, "error", result.err)
			return nil, result.err
		}

		// Update working directory if changed
		if result.workingDir != "" && result.workingDir != ps.currentDir {
			ps.logger.Info("working directory changed", "old", ps.currentDir, "new", result.workingDir)
			ps.currentDir = result.workingDir
		}

		shellResult := &ShellResult{
			Output:      result.output,
			Error:       result.errorOut,
			ExitCode:    result.exitCode,
			WorkingDir:  ps.currentDir,
			CommandLine: command,
		}

		ps.logger.Info("command completed", "command", command, "exit_code", result.exitCode, "working_dir", ps.currentDir)
		return shellResult, nil

	case <-cmdCtx.Done():
		ps.logger.Error("command timed out", "command", command, "timeout", timeout)
		return nil, fmt.Errorf("command timed out after %v", timeout)
	}
}

// GetCurrentDirectory returns the current working directory
func (ps *PersistentShell) GetCurrentDirectory() string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.currentDir
}

// GetWorkingDirectory queries the shell for its actual current directory
func (ps *PersistentShell) GetWorkingDirectory(ctx context.Context) (string, error) {
	if err := ps.UpdateWorkingDirectory(ctx); err != nil {
		return "", err
	}
	return ps.GetCurrentDirectory(), nil
}

// UpdateWorkingDirectory queries the shell for its current directory and updates the cached value
func (ps *PersistentShell) UpdateWorkingDirectory(ctx context.Context) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return fmt.Errorf("shell session is closed")
	}

	// Use a simple pwd command with a unique marker
	marker := fmt.Sprintf("__PWD_%d__", time.Now().UnixNano())
	pwdCommand := fmt.Sprintf("pwd && echo '%s'\n", marker)

	// Write command
	if _, err := ps.stdin.Write([]byte(pwdCommand)); err != nil {
		ps.logger.Error("failed to write pwd command", "error", err)
		ps.closed = true
		return fmt.Errorf("failed to write pwd command: %w", err)
	}

	// Read response with timeout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(ps.stdout)
		var pwd string
		
		for {
			select {
			case <-ctx.Done():
				errorChan <- fmt.Errorf("timeout reading pwd")
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				errorChan <- fmt.Errorf("error reading pwd: %w", err)
				return
			}

			line = strings.TrimSpace(line)
			
			// If we see our marker, the previous line was the pwd
			if line == marker {
				if pwd != "" {
					resultChan <- pwd
					return
				}
			} else if pwd == "" && line != "" && !strings.Contains(line, marker) {
				// This might be the pwd output
				pwd = line
			}
		}
	}()

	select {
	case pwd := <-resultChan:
		if pwd != ps.currentDir {
			ps.logger.Info("working directory updated", "old", ps.currentDir, "new", pwd)
			ps.currentDir = pwd
		}
		return nil
	case err := <-errorChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("timeout getting working directory")
	}
}

// GetOriginalDirectory returns the original working directory when shell was created
func (ps *PersistentShell) GetOriginalDirectory() string {
	return ps.originalDir
}

// IsPathWithinOriginal checks if the current path is within the original directory
func (ps *PersistentShell) IsPathWithinOriginal() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	abs, err := filepath.Abs(ps.currentDir)
	if err != nil {
		return false
	}
	
	origAbs, err := filepath.Abs(ps.originalDir)
	if err != nil {
		return false
	}
	
	rel, err := filepath.Rel(origAbs, abs)
	if err != nil {
		return false
	}
	
	// Check if the relative path doesn't start with ".." (would indicate going outside)
	return !strings.HasPrefix(rel, "..")
}

// ResetToOriginalDirectory changes back to the original directory
func (ps *PersistentShell) ResetToOriginalDirectory(ctx context.Context) error {
	ps.logger.Warn("resetting to original directory", "current", ps.currentDir, "original", ps.originalDir)
	
	result, err := ps.ExecuteCommand(ctx, fmt.Sprintf("cd %s", ps.originalDir), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to reset to original directory: %w", err)
	}
	
	if result.ExitCode != 0 {
		return fmt.Errorf("cd command failed: %s", result.Error)
	}
	
	return nil
}

// ValidateCommand checks if a command is safe to execute
func (ps *PersistentShell) ValidateCommand(command string) error {
	// Check for empty command
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command not allowed")
	}
	
	// Check for attempts to navigate outside using absolute paths
	commandLower := strings.ToLower(command)
	
	// Check for cd to absolute paths outside the project
	if strings.Contains(commandLower, "cd ") {
		// Extract the path after cd
		parts := strings.Fields(command)
		for i, part := range parts {
			if part == "cd" && i+1 < len(parts) {
				targetPath := parts[i+1]
				// Check if it's an absolute path
				if strings.HasPrefix(targetPath, "/") && targetPath != "/" {
					// Allow navigation within the original directory
					if !strings.HasPrefix(targetPath, ps.originalDir) {
						return fmt.Errorf("cannot navigate to absolute path outside project directory: %s", targetPath)
					}
				} else if targetPath == "/" {
					return fmt.Errorf("cannot navigate to root directory")
				}
			}
		}
	}
	
	// Check for dangerous commands
	dangerousCommands := []string{
		"rm -rf", "rm -r", "sudo", "su ", "chmod 777", "chown",
		"mkfs", "fdisk", "dd if=", ">/dev/", "curl", "wget",
		"nc ", "netcat", "ssh", "scp", "rsync", "ftp",
		"telnet", "rsh", "rcp", "mount", "umount", "kill -9",
		"killall", "pkill", "systemctl", "service", "init",
		"reboot", "shutdown", "halt", "poweroff", "passwd",
		"adduser", "deluser", "userdel", "usermod", "groupadd",
		"groupdel", "visudo", "crontab", "at ", "batch",
	}

	for _, dangerous := range dangerousCommands {
		if strings.Contains(commandLower, dangerous) {
			return fmt.Errorf("dangerous command not allowed: %s", command)
		}
	}
	
	return nil
}

// Close terminates the shell session
func (ps *PersistentShell) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return nil
	}

	ps.logger.Info("closing persistent shell session", "session_id", ps.sessionID)
	ps.closed = true

	// Send exit command
	if ps.stdin != nil {
		ps.stdin.Write([]byte("exit\n"))
		ps.stdin.Close()
	}

	// Close pipes
	if ps.stdout != nil {
		ps.stdout.Close()
	}
	if ps.stderr != nil {
		ps.stderr.Close()
	}

	// Wait for process to exit
	if ps.cmd != nil && ps.cmd.Process != nil {
		ps.cmd.Wait()
	}

	return nil
}

// GetSessionID returns the unique session identifier
func (ps *PersistentShell) GetSessionID() string {
	return ps.sessionID
}