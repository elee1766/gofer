package tool_createdir

import (
	"context"
	"fmt"
	"os"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "create_directory"

const createDirectoryPrompt = `Create a new directory. Creates parent directories as needed if they don't exist.`

// CreateDirectoryInput represents the input for creating a directory
type CreateDirectoryInput struct {
	Path        string `json:"path" jsonschema:"required,description=The directory path to create"`
	Permissions string `json:"permissions,omitempty" jsonschema:"description=Directory permissions (octal, e.g., '0755')"`
	Recursive   bool   `json:"recursive,omitempty" jsonschema:"description=Create parent directories if they don't exist"`
}

// CreateDirectoryOutput represents the output of creating a directory
type CreateDirectoryOutput struct {
	Path        string `json:"path" jsonschema:"description=The directory path that was created"`
	Created     bool   `json:"created" jsonschema:"description=Whether the directory was successfully created"`
	Permissions string `json:"permissions" jsonschema:"description=The actual permissions set on the directory"`
	Recursive   bool   `json:"recursive" jsonschema:"description=Whether parent directories were created as needed"`
}

// makeCreateDirectoryHandler creates a typed handler for the create directory tool
func makeCreateDirectoryHandler(fs afero.Fs) func(context.Context, CreateDirectoryInput) (CreateDirectoryOutput, error) {
	return func(ctx context.Context, input CreateDirectoryInput) (CreateDirectoryOutput, error) {
		logger := toolsutil.GetLogger()

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			logger.Error("unsafe path rejected", "path", input.Path)
			return CreateDirectoryOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		// Set default permissions if not specified
		perm := os.FileMode(0755)
		if input.Permissions != "" {
			// Parse octal permissions (e.g., "0755")
			var parsedPerm uint32
			if _, err := fmt.Sscanf(input.Permissions, "%o", &parsedPerm); err != nil {
				logger.Error("invalid permissions format", "permissions", input.Permissions, "error", err)
				return CreateDirectoryOutput{}, fmt.Errorf("invalid permissions format: %s", input.Permissions)
			}
			perm = os.FileMode(parsedPerm)
		}

		logger.Info("creating directory", "path", input.Path, "permissions", perm, "recursive", input.Recursive)

		var err error
		if input.Recursive {
			err = fs.MkdirAll(input.Path, perm)
		} else {
			err = fs.Mkdir(input.Path, perm)
		}

		if err != nil {
			logger.Error("failed to create directory", "path", input.Path, "error", err)
			return CreateDirectoryOutput{}, fmt.Errorf("failed to create directory: %v", err)
		}

		// Get directory info
		info, err := fs.Stat(input.Path)
		if err != nil {
			logger.Error("failed to stat created directory", "path", input.Path, "error", err)
			return CreateDirectoryOutput{}, fmt.Errorf("failed to stat created directory: %v", err)
		}

		result := CreateDirectoryOutput{
			Path:        input.Path,
			Created:     true,
			Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
			Recursive:   input.Recursive,
		}

		logger.Info("directory created successfully", "path", input.Path, "permissions", info.Mode().Perm())
		return result, nil
	}
}

// Tool returns the create_directory tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, createDirectoryPrompt, makeCreateDirectoryHandler(fs))
}


