package tool_listdir

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "list_directory"

const listDirectoryPrompt = `Lists files and directories in a given path. The path parameter can be an absolute path or a relative path (relative to the current working directory). You can optionally provide an array of glob patterns to ignore with the ignore parameter. You should generally prefer the Glob and Grep tools, if you know which directories to search.`

// ListDirectoryInput represents the input for listing a directory
type ListDirectoryInput struct {
	Path      string `json:"path" jsonschema:"required,description=The directory path to list"`
	Recursive bool   `json:"recursive,omitempty" jsonschema:"description=Whether to list recursively"`
}

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name     string `json:"name" jsonschema:"description=The name of the file or directory"`
	Path     string `json:"path" jsonschema:"description=The path to the file or directory"`
	IsDir    bool   `json:"is_dir" jsonschema:"description=Whether this is a directory"`
	Size     int64  `json:"size" jsonschema:"description=File size in bytes"`
	ModTime  string `json:"mod_time" jsonschema:"description=Last modification time in RFC3339 format"`
	Language string `json:"language,omitempty" jsonschema:"description=Detected programming language (for files only)"`
}

// ListDirectoryOutput represents the output of listing a directory
type ListDirectoryOutput struct {
	Path  string     `json:"path" jsonschema:"description=The directory path that was listed"`
	Files []FileInfo `json:"files" jsonschema:"description=List of files and directories"`
	Count int        `json:"count" jsonschema:"description=Total number of items found"`
}

// makeListDirectoryHandler creates a typed handler for the list directory tool
func makeListDirectoryHandler(fs afero.Fs) func(context.Context, ListDirectoryInput) (ListDirectoryOutput, error) {
	return func(ctx context.Context, input ListDirectoryInput) (ListDirectoryOutput, error) {
		logger := toolsutil.GetLogger()

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			logger.Error("unsafe path rejected", "path", input.Path)
			return ListDirectoryOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		logger.Info("listing directory", "path", input.Path, "recursive", input.Recursive)

		var files []FileInfo

		if input.Recursive {
			err := afero.Walk(fs, input.Path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip errors and continue
				}

				fileInfo := FileInfo{
					Name:    info.Name(),
					Path:    path,
					IsDir:   info.IsDir(),
					Size:    info.Size(),
					ModTime: info.ModTime().Format(time.RFC3339),
				}

				if !info.IsDir() {
					fileInfo.Language = toolsutil.DetectLanguage(path, nil)
				}

				files = append(files, fileInfo)
				return nil
			})
			if err != nil {
				logger.Error("failed to walk directory", "path", input.Path, "error", err)
				return ListDirectoryOutput{}, fmt.Errorf("failed to walk directory: %v", err)
			}
		} else {
			entries, err := afero.ReadDir(fs, input.Path)
			if err != nil {
				logger.Error("failed to read directory", "path", input.Path, "error", err)
				return ListDirectoryOutput{}, fmt.Errorf("failed to read directory: %v", err)
			}

			for _, info := range entries {
				filePath := filepath.Join(input.Path, info.Name())
				fileInfo := FileInfo{
					Name:    info.Name(),
					Path:    filePath,
					IsDir:   info.IsDir(),
					Size:    info.Size(),
					ModTime: info.ModTime().Format(time.RFC3339),
				}

				if !info.IsDir() {
					fileInfo.Language = toolsutil.DetectLanguage(filePath, nil)
				}

				files = append(files, fileInfo)
			}
		}

		result := ListDirectoryOutput{
			Path:  input.Path,
			Files: files,
			Count: len(files),
		}

		logger.Info("directory listed successfully", "path", input.Path, "count", len(files))
		return result, nil
	}
}

// Tool returns the list_directory tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, listDirectoryPrompt, makeListDirectoryHandler(fs))
}


