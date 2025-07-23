package tool_getfileinfo

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "get_file_info"

const getFileInfoPrompt = `Get detailed information about a file or directory including size, permissions, modification time, and file type. This tool provides metadata about files without reading their contents.`

// GetFileInfoInput represents the parameters for get_file_info
type GetFileInfoInput struct {
	Path string `json:"path" required:"true" description:"The file or directory path"`
}

// GetFileInfoOutput represents the response from get_file_info
type GetFileInfoOutput struct {
	Path        string    `json:"path" description:"The file or directory path"`
	Name        string    `json:"name" description:"The name of the file or directory"`
	Size        int64     `json:"size" description:"Size in bytes"`
	SizeHuman   string    `json:"size_human" description:"Human-readable size"`
	IsDir       bool      `json:"is_dir" description:"Whether this is a directory"`
	Mode        string    `json:"mode" description:"File mode string"`
	Permissions string    `json:"permissions" description:"File permissions in octal"`
	ModTime     time.Time `json:"mod_time" description:"Last modification time"`
	Exists      bool      `json:"exists" description:"Whether the file exists"`
	
	// Directory-specific fields (only present when IsDir is true)
	EntryCount *int `json:"entry_count,omitempty" description:"Number of entries in directory"`
	FileCount  *int `json:"file_count,omitempty" description:"Number of files in directory"`
	DirCount   *int `json:"dir_count,omitempty" description:"Number of subdirectories"`
	
	// File-specific fields
	Extension string `json:"extension,omitempty" description:"File extension"`
	Language  string `json:"language,omitempty" description:"Detected programming language"`
	MimeType  string `json:"mime_type,omitempty" description:"MIME type based on extension"`
}

// Tool returns the get_file_info tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, getFileInfoPrompt, makeGetFileInfoHandler(fs))
}

// makeGetFileInfoHandler creates a type-safe handler for the get_file_info tool
func makeGetFileInfoHandler(fs afero.Fs) func(ctx context.Context, input GetFileInfoInput) (GetFileInfoOutput, error) {
	return func(ctx context.Context, input GetFileInfoInput) (GetFileInfoOutput, error) {
		logger := toolsutil.GetLogger()
		
		// Check for cancellation
		select {
		case <-ctx.Done():
			return GetFileInfoOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			logger.Error("unsafe path rejected", "path", input.Path)
			return GetFileInfoOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		logger.Info("getting file info", "path", input.Path)

		// Check for cancellation before I/O operations
		select {
		case <-ctx.Done():
			return GetFileInfoOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Get file info
		info, err := fs.Stat(input.Path)
		if err != nil {
			logger.Error("failed to stat file", "path", input.Path, "error", err)
			return GetFileInfoOutput{}, fmt.Errorf("failed to get file info: %v", err)
		}

		result := GetFileInfoOutput{
			Path:        input.Path,
			Name:        info.Name(),
			Size:        info.Size(),
			SizeHuman:   toolsutil.FormatBytes(info.Size()),
			IsDir:       info.IsDir(),
			Mode:        info.Mode().String(),
			Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
			ModTime:     info.ModTime(),
			Exists:      true,
		}

		// Add directory-specific info
		if info.IsDir() {
			entries, err := afero.ReadDir(fs, input.Path)
			if err == nil {
				entryCount := len(entries)
				result.EntryCount = &entryCount
				
				fileCount := 0
				dirCount := 0
				for _, entry := range entries {
					if entry.IsDir() {
						dirCount++
					} else {
						fileCount++
					}
				}
				result.FileCount = &fileCount
				result.DirCount = &dirCount
			}
		} else {
			// Add file-specific info
			result.Extension = filepath.Ext(input.Path)
			result.Language = toolsutil.DetectLanguage(input.Path, nil)
			
			// Add MIME type hint based on extension
			ext := filepath.Ext(input.Path)
			if ext != "" {
				switch ext {
				case ".go":
					result.MimeType = "text/x-go"
				case ".js":
					result.MimeType = "application/javascript"
				case ".ts":
					result.MimeType = "application/typescript"
				case ".py":
					result.MimeType = "text/x-python"
				case ".md":
					result.MimeType = "text/markdown"
				case ".json":
					result.MimeType = "application/json"
				case ".yaml", ".yml":
					result.MimeType = "application/x-yaml"
				case ".xml":
					result.MimeType = "application/xml"
				case ".html":
					result.MimeType = "text/html"
				case ".css":
					result.MimeType = "text/css"
				default:
					result.MimeType = "application/octet-stream"
				}
			}
		}

		logger.Info("file info retrieved", "path", input.Path, "size", info.Size(), "is_dir", info.IsDir())
		
		return result, nil
	}
}