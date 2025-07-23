package tool_editfile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "edit_file"

const editFilePrompt = `Performs exact string replacements in files.

Usage:
- You must use your 'read_file' tool at least once in the conversation before editing. This tool will error if you attempt an edit without reading the file.
- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. The line number prefix format is: spaces + line number + tab. Everything after that tab is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- The edit will FAIL if old_string is not unique in the file. Either provide a larger string with more surrounding context to make it unique or use replace_all to change every instance of old_string.
- Use replace_all for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.`

// EditFileInput represents the parameters for edit_file
type EditFileInput struct {
	Path         string `json:"path" required:"true" description:"The file path to edit"`
	OldContent   string `json:"old_content" required:"true" description:"The exact content to replace"`
	NewContent   string `json:"new_content" required:"true" description:"The new content to replace with"`
	LineNumber   int    `json:"line_number,omitempty" description:"Optional line number for context"`
	CreateBackup bool   `json:"create_backup,omitempty" description:"Create backup before editing"`
}

// EditFileOutput represents the response from edit_file
type EditFileOutput struct {
	Path          string `json:"path" description:"The file path that was edited"`
	OldSize       int    `json:"old_size" description:"Size of the file before edit"`
	NewSize       int    `json:"new_size" description:"Size of the file after edit"`
	ChangesMade   bool   `json:"changes_made" description:"Whether changes were made"`
	BackupCreated bool   `json:"backup_created" description:"Whether a backup was created"`
}

// Tool returns the edit_file tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, editFilePrompt, makeEditFileHandler(fs))
}

// makeEditFileHandler creates a type-safe handler for the edit_file tool
func makeEditFileHandler(fs afero.Fs) func(ctx context.Context, input EditFileInput) (EditFileOutput, error) {
	return func(ctx context.Context, input EditFileInput) (EditFileOutput, error) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return EditFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			toolsutil.GetLogger().Error("unsafe path rejected", "path", input.Path)
			return EditFileOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		// Validate content size
		if err := toolsutil.ValidateFileSize(int64(len(input.OldContent))); err != nil {
			return EditFileOutput{}, fmt.Errorf("old content too large: %v", err)
		}
		if err := toolsutil.ValidateFileSize(int64(len(input.NewContent))); err != nil {
			return EditFileOutput{}, fmt.Errorf("new content too large: %v", err)
		}

		toolsutil.GetLogger().Info("editing file", "path", input.Path, "create_backup", input.CreateBackup)

		// Check for cancellation before I/O operations
		select {
		case <-ctx.Done():
			return EditFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Read current file content
		content, err := afero.ReadFile(fs, input.Path)
		if err != nil {
			toolsutil.GetLogger().Error("failed to read file", "path", input.Path, "error", err)
			return EditFileOutput{}, fmt.Errorf("failed to read file: %v", err)
		}

		currentContent := string(content)

		// Create backup if requested
		if input.CreateBackup {
			backupPath := input.Path + ".backup." + time.Now().Format("20060102_150405")
			if err := afero.WriteFile(fs, backupPath, content, 0644); err != nil {
				toolsutil.GetLogger().Error("failed to create backup", "path", backupPath, "error", err)
				return EditFileOutput{}, fmt.Errorf("failed to create backup: %v", err)
			}
			toolsutil.GetLogger().Info("backup created", "path", backupPath)
		}

		// Find and replace content
		if !strings.Contains(currentContent, input.OldContent) {
			toolsutil.GetLogger().Error("old content not found in file", "path", input.Path)
			return EditFileOutput{}, fmt.Errorf("old content not found in file")
		}

		newContent := strings.Replace(currentContent, input.OldContent, input.NewContent, 1)

		// Check for cancellation before writing
		select {
		case <-ctx.Done():
			return EditFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Write updated content
		if err := afero.WriteFile(fs, input.Path, []byte(newContent), 0644); err != nil {
			toolsutil.GetLogger().Error("failed to write file", "path", input.Path, "error", err)
			return EditFileOutput{}, fmt.Errorf("failed to write file: %v", err)
		}

		toolsutil.GetLogger().Info("file edited successfully", "path", input.Path, "old_size", len(currentContent), "new_size", len(newContent))

		return EditFileOutput{
			Path:          input.Path,
			OldSize:       len(currentContent),
			NewSize:       len(newContent),
			ChangesMade:   true,
			BackupCreated: input.CreateBackup,
		}, nil
	}
}

