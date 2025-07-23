package tool_writefile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "write_file"

const writeFilePrompt = `Writes a file to the local filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path.
- If this is an existing file, you MUST use the Read tool first to read the file's contents. This tool will fail if you did not read the file first.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
- Only use emojis if the user explicitly requests it. Avoid writing emojis to files unless asked.`

// WriteFileInput represents the parameters for write_file
type WriteFileInput struct {
	Path       string `json:"path" jsonschema:"required,description=The file path"`
	Content    string `json:"content" jsonschema:"required,description=The content"`
	CreateDirs bool   `json:"create_dirs,omitempty" jsonschema:"description=Create parent directories if they don't exist"`
	Mode       int    `json:"mode,omitempty" jsonschema:"description=File permissions (octal, e.g., 644),default=644"`
}

// WriteFileOutput represents the response from write_file
type WriteFileOutput struct {
	Path    string `json:"path" jsonschema:"description=The file path that was written"`
	Size    int    `json:"size" jsonschema:"description=Size of content written in bytes"`
	Success bool   `json:"success" jsonschema:"description=Whether the file was written successfully"`
}

// Tool returns the write_file tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, writeFilePrompt, makeWriteFileHandler(fs))
}


// makeWriteFileHandler creates a type-safe handler for the write_file tool
func makeWriteFileHandler(fs afero.Fs) func(ctx context.Context, input WriteFileInput) (WriteFileOutput, error) {
	return func(ctx context.Context, input WriteFileInput) (WriteFileOutput, error) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return WriteFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			toolsutil.GetLogger().Error("unsafe path rejected", "path", input.Path)
			return WriteFileOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		// Check content size
		if err := toolsutil.ValidateFileSize(int64(len(input.Content))); err != nil {
			toolsutil.GetLogger().Error("content too large", "path", input.Path, "size", len(input.Content))
			return WriteFileOutput{}, err
		}

		// Set default values
		mode := os.FileMode(input.Mode)
		if input.Mode == 0 {
			mode = 0644
		}

		toolsutil.GetLogger().Info("writing file", "path", input.Path, "content_size", len(input.Content), "create_dirs", input.CreateDirs, "mode", input.Mode)

		// Handle directory creation
		dir := filepath.Dir(input.Path)
		if input.CreateDirs {
			if err := fs.MkdirAll(dir, 0755); err != nil {
				toolsutil.GetLogger().Error("failed to create directory", "dir", dir, "error", err)
				return WriteFileOutput{}, fmt.Errorf("failed to create directory: %v", err)
			}
		} else {
			// Check if directory exists when create_dirs is false
			if _, err := fs.Stat(dir); err != nil {
				toolsutil.GetLogger().Error("directory does not exist and create_dirs is false", "dir", dir)
				return WriteFileOutput{}, fmt.Errorf("directory does not exist: %s", dir)
			}
		}

		// Check for cancellation before writing file
		select {
		case <-ctx.Done():
			return WriteFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Write file
		if err := afero.WriteFile(fs, input.Path, []byte(input.Content), mode); err != nil {
			toolsutil.GetLogger().Error("failed to write file", "path", input.Path, "error", err)
			return WriteFileOutput{}, fmt.Errorf("failed to write file: %v", err)
		}

		toolsutil.GetLogger().Info("file written successfully", "path", input.Path, "size", len(input.Content))

		return WriteFileOutput{
			Path:    input.Path,
			Size:    len(input.Content),
			Success: true,
		}, nil
	}
}

