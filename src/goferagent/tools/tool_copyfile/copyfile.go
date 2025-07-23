package tool_copyfile

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "copy_file"

const copyFilePrompt = `Copy a file or directory from one location to another. Creates the destination directory if it doesn't exist.`

// CopyFileInput represents the parameters for copy_file
type CopyFileInput struct {
	Source      string `json:"source" required:"true" description:"The source file or directory path"`
	Destination string `json:"destination" required:"true" description:"The destination file or directory path"`
	Overwrite   bool   `json:"overwrite,omitempty" description:"Overwrite destination if it exists"`
	Recursive   bool   `json:"recursive,omitempty" description:"Copy directories recursively"`
	CreateDirs  bool   `json:"create_dirs,omitempty" default:"true" description:"Create destination directories if they don't exist"`
}

// CopyFileOutput represents the response from copy_file
type CopyFileOutput struct {
	Source       string `json:"source" description:"The source path that was copied"`
	Destination  string `json:"destination" description:"The destination path"`
	Copied       bool   `json:"copied" description:"Whether the file was copied"`
	Overwritten  bool   `json:"overwritten" description:"Whether an existing file was overwritten"`
	BytesCopied  int64  `json:"bytes_copied" description:"Number of bytes copied"`
	Size         int64  `json:"size" description:"Size of the source file"`
	FilesCopied  int    `json:"files_copied,omitempty" description:"Number of files copied (for directories)"`
	IsDirectory  bool   `json:"is_directory,omitempty" description:"Whether the copied item was a directory"`
}

// Tool returns the copy_file tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, copyFilePrompt, makeCopyFileHandler(fs))
}



// makeCopyFileHandler creates a type-safe handler for the copy_file tool
func makeCopyFileHandler(fs afero.Fs) func(ctx context.Context, input CopyFileInput) (CopyFileOutput, error) {
	return func(ctx context.Context, input CopyFileInput) (CopyFileOutput, error) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return CopyFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate paths
		if !toolsutil.IsPathSafe(input.Source) || !toolsutil.IsPathSafe(input.Destination) {
			toolsutil.GetLogger().Error("unsafe path rejected", "source", input.Source, "destination", input.Destination)
			return CopyFileOutput{}, fmt.Errorf("unsafe path")
		}

		toolsutil.GetLogger().Info("copying file", "source", input.Source, "destination", input.Destination, "overwrite", input.Overwrite)

		// Check for cancellation before I/O operations
		select {
		case <-ctx.Done():
			return CopyFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Check if source exists and get info
		sourceInfo, err := fs.Stat(input.Source)
		if err != nil {
			toolsutil.GetLogger().Error("source file not found", "source", input.Source, "error", err)
			return CopyFileOutput{}, fmt.Errorf("source file not found: %s", input.Source)
		}

		// Handle directory copying
		if sourceInfo.IsDir() {
			if !input.Recursive {
				toolsutil.GetLogger().Error("source is a directory but recursive not enabled", "source", input.Source)
				return CopyFileOutput{}, fmt.Errorf("source is a directory, enable recursive copying")
			}
			return copyDirectoryRecursivelyGeneric(ctx, fs, input, sourceInfo)
		}

		// Validate file size for regular files
		if err := toolsutil.ValidateFileSize(sourceInfo.Size()); err != nil {
			toolsutil.GetLogger().Error("source file too large", "source", input.Source, "size", sourceInfo.Size())
			return CopyFileOutput{}, err
		}

		// Check if destination exists
		_, err = fs.Stat(input.Destination)
		destinationExists := err == nil

		if destinationExists && !input.Overwrite {
			toolsutil.GetLogger().Error("destination exists and overwrite not allowed", "destination", input.Destination)
			return CopyFileOutput{}, fmt.Errorf("destination exists and overwrite not allowed: %s", input.Destination)
		}

		// Create destination directory if requested and it doesn't exist
		if input.CreateDirs {
			destDir := filepath.Dir(input.Destination)
			if err := fs.MkdirAll(destDir, 0755); err != nil {
				toolsutil.GetLogger().Error("failed to create destination directory", "dir", destDir, "error", err)
				return CopyFileOutput{}, fmt.Errorf("failed to create destination directory: %v", err)
			}
		}

		// Check for cancellation before copying
		select {
		case <-ctx.Done():
			return CopyFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Open source file
		sourceFile, err := fs.Open(input.Source)
		if err != nil {
			toolsutil.GetLogger().Error("failed to open source file", "source", input.Source, "error", err)
			return CopyFileOutput{}, fmt.Errorf("failed to open source file: %v", err)
		}
		defer sourceFile.Close()

		// Create destination file
		destFile, err := fs.Create(input.Destination)
		if err != nil {
			toolsutil.GetLogger().Error("failed to create destination file", "destination", input.Destination, "error", err)
			return CopyFileOutput{}, fmt.Errorf("failed to create destination file: %v", err)
		}
		defer destFile.Close()

		// Copy the content
		bytesCopied, err := io.Copy(destFile, sourceFile)
		if err != nil {
			toolsutil.GetLogger().Error("failed to copy file content", "source", input.Source, "destination", input.Destination, "error", err)
			return CopyFileOutput{}, fmt.Errorf("failed to copy file content: %v", err)
		}

		// Preserve file permissions
		if err := fs.Chmod(input.Destination, sourceInfo.Mode()); err != nil {
			toolsutil.GetLogger().Warn("failed to set file permissions", "destination", input.Destination, "error", err)
		}

		toolsutil.GetLogger().Info("file copied successfully", "source", input.Source, "destination", input.Destination, "bytes", bytesCopied)

		return CopyFileOutput{
			Source:       input.Source,
			Destination:  input.Destination,
			Copied:       true,
			Overwritten:  destinationExists,
			BytesCopied:  bytesCopied,
			Size:         sourceInfo.Size(),
		}, nil
	}
}

// copyDirectoryRecursivelyGeneric copies a directory and all its contents for GenericTool
func copyDirectoryRecursivelyGeneric(ctx context.Context, fs afero.Fs, input CopyFileInput, sourceInfo os.FileInfo) (CopyFileOutput, error) {
	// Check if destination exists
	_, err := fs.Stat(input.Destination)
	destinationExists := err == nil

	if destinationExists && !input.Overwrite {
		toolsutil.GetLogger().Error("destination exists and overwrite not allowed", "destination", input.Destination)
		return CopyFileOutput{}, fmt.Errorf("destination exists and overwrite not allowed: %s", input.Destination)
	}

	// Create destination directory
	if err := fs.MkdirAll(input.Destination, sourceInfo.Mode()); err != nil {
		toolsutil.GetLogger().Error("failed to create destination directory", "destination", input.Destination, "error", err)
		return CopyFileOutput{}, fmt.Errorf("failed to create destination directory: %v", err)
	}

	var totalBytesCopied int64
	var filesCopied int

	// Walk the source directory
	err = afero.Walk(fs, input.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check for cancellation during walk
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled")
		default:
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(input.Source, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(input.Destination, relPath)

		if info.IsDir() {
			// Create directory
			if err := fs.MkdirAll(destPath, info.Mode()); err != nil {
				return err
			}
		} else {
			// Copy file
			srcFile, err := fs.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			// Create destination directory if needed
			destDir := filepath.Dir(destPath)
			if err := fs.MkdirAll(destDir, 0755); err != nil {
				return err
			}

			destFile, err := fs.Create(destPath)
			if err != nil {
				return err
			}
			defer destFile.Close()

			// Copy content
			copied, err := io.Copy(destFile, srcFile)
			if err != nil {
				return err
			}

			totalBytesCopied += copied
			filesCopied++

			// Set permissions
			if err := fs.Chmod(destPath, info.Mode()); err != nil {
				toolsutil.GetLogger().Warn("failed to set file permissions", "file", destPath, "error", err)
			}
		}

		return nil
	})

	if err != nil {
		toolsutil.GetLogger().Error("failed to copy directory", "source", input.Source, "destination", input.Destination, "error", err)
		return CopyFileOutput{}, fmt.Errorf("failed to copy directory: %v", err)
	}

	toolsutil.GetLogger().Info("directory copied successfully", "source", input.Source, "destination", input.Destination, "files", filesCopied, "bytes", totalBytesCopied)

	return CopyFileOutput{
		Source:       input.Source,
		Destination:  input.Destination,
		Copied:       true,
		Overwritten:  destinationExists,
		BytesCopied:  totalBytesCopied,
		FilesCopied:  filesCopied,
		IsDirectory:  true,
	}, nil
}