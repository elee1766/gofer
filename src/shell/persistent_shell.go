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

	// Create shell command with bash
	cmd := exec.Command("bash", "--norc", "--noprofile")
	cmd.Dir = currentDir
	
	// Set up environment
	cmd.Env = append(os.Environ(),
		"PS1=", // Disable prompt to avoid interference
		"TERM=dumb", // Simple terminal
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

	// Initial setup - set markers for output detection
	shell.logger.Info("starting persistent shell session", "session_id", sessionID, "working_dir", currentDir)
	
	return shell, nil
}

// ExecuteCommand runs a command in the persistent shell
func (ps *PersistentShell) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (*ShellResult, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return nil, fmt.Errorf("shell session is closed")
	}

	ps.logger.Info("executing command in persistent shell", "command", command, "session_id", ps.sessionID)

	// Create unique markers for this command
	startMarker := fmt.Sprintf("__CMD_START_%d__", time.Now().UnixNano())
	endMarker := fmt.Sprintf("__CMD_END_%d__", time.Now().UnixNano())
	pwdMarker := fmt.Sprintf("__PWD_%d__", time.Now().UnixNano())

	// Construct the full command with markers and directory tracking
	fullCommand := fmt.Sprintf("echo '%s'; %s; EXIT_CODE=$?; echo '%s'; pwd; echo '%s'; exit $EXIT_CODE\n", 
		startMarker, command, endMarker, pwdMarker)

	// Set up timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel to collect results
	resultChan := make(chan *ShellResult, 1)
	errorChan := make(chan error, 1)

	// Execute command in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("panic in command execution: %v", r)
			}
		}()

		// Send command
		_, err := ps.stdin.Write([]byte(fullCommand))
		if err != nil {
			errorChan <- fmt.Errorf("failed to write command: %w", err)
			return
		}

		// Read output until we see our markers
		var outputLines []string
		var errorLines []string
		var newWorkingDir string
		foundStart := false
		foundEnd := false
		foundPwd := false
		exitCode := 0

		// Read stdout
		stdoutScanner := bufio.NewScanner(ps.stdout)
		stderrScanner := bufio.NewScanner(ps.stderr)

		// Read lines until we find our end marker
		for !foundEnd {
			select {
			case <-cmdCtx.Done():
				errorChan <- fmt.Errorf("command timed out after %v", timeout)
				return
			default:
			}

			// Try to read stdout
			if stdoutScanner.Scan() {
				line := stdoutScanner.Text()
				
				if line == startMarker {
					foundStart = true
					continue
				} else if line == endMarker {
					foundEnd = true
					continue
				} else if strings.HasPrefix(line, pwdMarker) {
					foundPwd = true
					continue
				} else if foundEnd && foundPwd && !foundStart {
					// This should be the pwd output
					newWorkingDir = strings.TrimSpace(line)
					break
				} else if foundStart && !foundEnd {
					// This is command output
					outputLines = append(outputLines, line)
				} else if foundPwd && !foundStart {
					// This is the pwd output after command
					newWorkingDir = strings.TrimSpace(line)
				}
			}

			// Try to read stderr (non-blocking)
			if stderrScanner.Scan() {
				errorLines = append(errorLines, stderrScanner.Text())
			}
		}

		// Try to get pwd if we haven't found it yet
		if newWorkingDir == "" {
			// Send a separate pwd command
			pwdCmd := fmt.Sprintf("pwd; echo '%s'\n", pwdMarker)
			ps.stdin.Write([]byte(pwdCmd))
			
			for stdoutScanner.Scan() {
				line := stdoutScanner.Text()
				if line == pwdMarker {
					break
				}
				if newWorkingDir == "" {
					newWorkingDir = strings.TrimSpace(line)
				}
			}
		}

		// Update current directory if we got a valid path
		if newWorkingDir != "" && newWorkingDir != ps.currentDir {
			ps.logger.Info("working directory changed", "old", ps.currentDir, "new", newWorkingDir)
			ps.currentDir = newWorkingDir
		}

		result := &ShellResult{
			Output:      strings.Join(outputLines, "\n"),
			Error:       strings.Join(errorLines, "\n"),
			ExitCode:    exitCode,
			WorkingDir:  ps.currentDir,
			CommandLine: command,
		}

		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		ps.logger.Info("command completed", "command", command, "exit_code", result.ExitCode, "working_dir", result.WorkingDir)
		return result, nil
	case err := <-errorChan:
		ps.logger.Error("command failed", "command", command, "error", err)
		return nil, err
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
	
	// Check for dangerous commands (same as before but adapted)
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

	commandLower := strings.ToLower(command)
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