package tool_readfile

import (
	"context"
	"fmt"
	"strings"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "read_file"

const readFilePrompt = `Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage:
- The file_path parameter can be an absolute path or a relative path (relative to the current working directory)
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify line_numbers: true to include line numbers in the output (format: "1: line content")
- Any lines longer than 2000 characters will be truncated
- This tool allows Claude Code to read images (eg PNG, JPG, etc). When reading an image file the contents are presented visually as Claude Code is a multimodal LLM.
- For Jupyter notebooks (.ipynb files), use the NotebookRead instead
- You have the capability to call multiple tools in a single response. It is always better to speculatively read multiple files as a batch that are potentially useful. 
- You will regularly be asked to read screenshots. If the user provides a path to a screenshot ALWAYS use this tool to view the file at the path. This tool will work with all temporary file paths like /var/folders/123/abc/T/TemporaryItems/NSIRD_screencaptureui_ZfB1tD/Screenshot.png
- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents.`

// ReadFileInput represents the parameters for read_file
type ReadFileInput struct {
	Path        string `json:"path" required:"true" description:"The file path to read (absolute or relative to current working directory)"`
	LineNumbers bool   `json:"line_numbers,omitempty" description:"Include line numbers in output (format: '1: line content')"`
}

// ReadFileOutput represents the response from read_file
type ReadFileOutput struct {
	Content  string `json:"content" description:"The file contents"`
	Path     string `json:"path" description:"The file path that was read"`
	Size     int64  `json:"size" description:"File size in bytes"`
	Language string `json:"language,omitempty" description:"Detected programming language"`
	IsText   bool   `json:"is_text" description:"Whether the file is a text file"`
}

// Tool returns the read_file tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, readFilePrompt, makeReadFileHandlerV2(fs))
}

// ToolMultimodal returns the read_file tool definition with multimodal support
func ToolMultimodal(fs afero.Fs) (agent.Tool, error) {
	return &agent.LegacyTool{
		Type: "function",
		Function: aisdk.ToolFunction{
			Name:        Name,
			Description: readFilePrompt,
			Parameters:  nil, // Will be set via reflection if needed
		},
		Executor: makeReadFileHandlerMultimodal(fs),
	}, nil
}


// makeReadFileHandler creates a type-safe handler for the read_file tool
func makeReadFileHandler(fs afero.Fs) func(ctx context.Context, input ReadFileInput) (ReadFileOutput, error) {
	return func(ctx context.Context, input ReadFileInput) (ReadFileOutput, error) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ReadFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			toolsutil.GetLogger().Error("unsafe path rejected", "path", input.Path)
			return ReadFileOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		toolsutil.GetLogger().Info("reading file", "path", input.Path)

		// Check for cancellation before I/O operations
		select {
		case <-ctx.Done():
			return ReadFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Check if file exists and get info
		info, err := fs.Stat(input.Path)
		if err != nil {
			toolsutil.GetLogger().Error("file not found", "path", input.Path, "error", err)
			return ReadFileOutput{}, fmt.Errorf("file not found: %s", input.Path)
		}

		// Check file size
		if err := toolsutil.ValidateFileSize(info.Size()); err != nil {
			toolsutil.GetLogger().Error("file too large", "path", input.Path, "size", info.Size())
			return ReadFileOutput{}, err
		}

		// Check for cancellation before reading file
		select {
		case <-ctx.Done():
			return ReadFileOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Read file content
		content, err := afero.ReadFile(fs, input.Path)
		if err != nil {
			toolsutil.GetLogger().Error("failed to read file", "path", input.Path, "error", err)
			return ReadFileOutput{}, fmt.Errorf("failed to read file: %v", err)
		}

		// Detect content type and language
		language := toolsutil.DetectLanguage(input.Path, content)
		isText := toolsutil.IsTextFile(content)

		// Format content with line numbers if requested
		var displayContent string
		if input.LineNumbers && isText {
			displayContent = addLineNumbers(string(content))
		} else {
			displayContent = string(content)
		}

		toolsutil.GetLogger().Info("file read successfully", "path", input.Path, "size", len(content), "language", language)

		return ReadFileOutput{
			Content:  displayContent,
			Path:     input.Path,
			Size:     int64(len(content)),
			Language: language,
			IsText:   isText,
		}, nil
	}
}


// addLineNumbers adds line numbers to the content
func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for i, line := range lines {
		result = append(result, fmt.Sprintf("%d: %s", i+1, line))
	}
	return strings.Join(result, "\n")
}
