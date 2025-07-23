package tool_movefile

import (
	"context"
	"fmt"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "move_file"

const moveFilePrompt = `Move or rename a file from one location to another. Creates the destination directory if it doesn't exist.`

// MoveFileInput represents the parameters for move_file
type MoveFileInput struct {
	Source      string `json:"source" required:"true" description:"The source file path"`
	Destination string `json:"destination" required:"true" description:"The destination file path"`
	Overwrite   bool   `json:"overwrite,omitempty" description:"Overwrite destination if it exists"`
}

// MoveFileOutput represents the response from move_file
type MoveFileOutput struct {
	Source       string `json:"source" description:"The source path that was moved"`
	Destination  string `json:"destination" description:"The destination path"`
	Moved        bool   `json:"moved" description:"Whether the file was moved"`
	Overwritten  bool   `json:"overwritten" description:"Whether an existing file was overwritten"`
	WasDirectory bool   `json:"was_directory" description:"Whether the moved item was a directory"`
	Size         int64  `json:"size" description:"Size of the moved file"`
}

// Tool returns the move_file tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, moveFilePrompt, makeMoveFileHandler(fs))
}


// makeMoveFileHandler creates a type-safe handler for the move_file tool
func makeMoveFileHandler(fs afero.Fs) func(ctx context.Context, input MoveFileInput) (MoveFileOutput, error) {
	return func(ctx context.Context, input MoveFileInput) (MoveFileOutput, error) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return MoveFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate paths
		if !toolsutil.IsPathSafe(input.Source) || !toolsutil.IsPathSafe(input.Destination) {
			toolsutil.GetLogger().Error("unsafe path rejected", "source", input.Source, "destination", input.Destination)
			return MoveFileOutput{}, fmt.Errorf("unsafe path")
		}

		toolsutil.GetLogger().Info("moving file", "source", input.Source, "destination", input.Destination, "overwrite", input.Overwrite)

		// Check for cancellation before I/O operations
		select {
		case <-ctx.Done():
			return MoveFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Check if source exists
		sourceInfo, err := fs.Stat(input.Source)
		if err != nil {
			toolsutil.GetLogger().Error("source file not found", "source", input.Source, "error", err)
			return MoveFileOutput{}, fmt.Errorf("source file not found: %s", input.Source)
		}

		// Check if destination exists
		_, err = fs.Stat(input.Destination)
		destinationExists := err == nil

		if destinationExists && !input.Overwrite {
			toolsutil.GetLogger().Error("destination exists and overwrite not allowed", "destination", input.Destination)
			return MoveFileOutput{}, fmt.Errorf("destination exists and overwrite not allowed: %s", input.Destination)
		}

		// Check for cancellation before move
		select {
		case <-ctx.Done():
			return MoveFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Perform the move
		if err := fs.Rename(input.Source, input.Destination); err != nil {
			toolsutil.GetLogger().Error("failed to move file", "source", input.Source, "destination", input.Destination, "error", err)
			return MoveFileOutput{}, fmt.Errorf("failed to move file: %v", err)
		}

		toolsutil.GetLogger().Info("file moved successfully", "source", input.Source, "destination", input.Destination)

		return MoveFileOutput{
			Source:       input.Source,
			Destination:  input.Destination,
			Moved:        true,
			Overwritten:  destinationExists,
			WasDirectory: sourceInfo.IsDir(),
			Size:         sourceInfo.Size(),
		}, nil
	}
}