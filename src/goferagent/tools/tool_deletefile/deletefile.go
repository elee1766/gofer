package tool_deletefile

import (
	"context"
	"fmt"
	"os"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "delete_file"

const deleteFilePrompt = `Delete a file or directory. Use with caution as this operation cannot be undone.`

// DeleteFileInput represents the parameters for delete_file
type DeleteFileInput struct {
	Path  string `json:"path" required:"true" description:"The file path to delete"`
	Force bool   `json:"force,omitempty" description:"Force deletion without confirmation"`
}

// DeleteFileOutput represents the response from delete_file
type DeleteFileOutput struct {
	Path        string `json:"path" description:"The path that was processed"`
	Deleted     bool   `json:"deleted" description:"Whether the file was deleted"`
	WasDirectory bool   `json:"was_directory" description:"Whether the deleted item was a directory"`
	Size        int64  `json:"size,omitempty" description:"Size of the deleted file"`
	Reason      string `json:"reason,omitempty" description:"Reason if not deleted"`
}

// Tool returns the delete_file tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, deleteFilePrompt, makeDeleteFileHandler(fs))
}


// makeDeleteFileHandler creates a type-safe handler for the delete_file tool
func makeDeleteFileHandler(fs afero.Fs) func(ctx context.Context, input DeleteFileInput) (DeleteFileOutput, error) {
	return func(ctx context.Context, input DeleteFileInput) (DeleteFileOutput, error) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return DeleteFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			toolsutil.GetLogger().Error("unsafe path rejected", "path", input.Path)
			return DeleteFileOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		toolsutil.GetLogger().Info("deleting file", "path", input.Path, "force", input.Force)

		// Check for cancellation before I/O operations
		select {
		case <-ctx.Done():
			return DeleteFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Check if file exists and get info
		info, err := fs.Stat(input.Path)
		if err != nil {
			if os.IsNotExist(err) {
				toolsutil.GetLogger().Info("file does not exist", "path", input.Path)
				return DeleteFileOutput{
					Path:    input.Path,
					Deleted: false,
					Reason:  "file does not exist",
				}, nil
			}
			toolsutil.GetLogger().Error("failed to stat file", "path", input.Path, "error", err)
			return DeleteFileOutput{}, fmt.Errorf("failed to stat file: %v", err)
		}

		isDirectory := info.IsDir()
		fileSize := info.Size()

		// Check for cancellation before deletion
		select {
		case <-ctx.Done():
			return DeleteFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Delete the file or directory
		if err := fs.RemoveAll(input.Path); err != nil {
			toolsutil.GetLogger().Error("failed to delete file", "path", input.Path, "error", err)
			return DeleteFileOutput{}, fmt.Errorf("failed to delete file: %v", err)
		}

		toolsutil.GetLogger().Info("file deleted successfully", "path", input.Path, "was_directory", isDirectory)

		return DeleteFileOutput{
			Path:         input.Path,
			Deleted:      true,
			WasDirectory: isDirectory,
			Size:         fileSize,
		}, nil
	}
}