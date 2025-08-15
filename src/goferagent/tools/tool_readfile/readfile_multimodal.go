package tool_readfile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// makeReadFileHandlerMultimodal creates a handler that returns multimodal content
func makeReadFileHandlerMultimodal(fs afero.Fs) func(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
	return func(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
		// Parse input
		var input ReadFileInput
		if err := parseToolInput(call, &input); err != nil {
			return aisdk.NewErrorToolResponse(fmt.Sprintf("failed to parse input: %v", err)), nil
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return aisdk.NewErrorToolResponse("operation cancelled"), nil
		default:
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			toolsutil.GetLogger().Error("unsafe path rejected", "path", input.Path)
			return aisdk.NewErrorToolResponse(fmt.Sprintf("unsafe path: %s", input.Path)), nil
		}

		toolsutil.GetLogger().Info("reading file", "path", input.Path, "line_numbers", input.LineNumbers)

		// Check if file exists and get info
		info, err := fs.Stat(input.Path)
		if err != nil {
			toolsutil.GetLogger().Error("file not found", "path", input.Path, "error", err)
			return aisdk.NewErrorToolResponse(fmt.Sprintf("file not found: %s", input.Path)), nil
		}

		// Check if it's a directory
		if info.IsDir() {
			return aisdk.NewErrorToolResponse(fmt.Sprintf("path is a directory, not a file: %s", input.Path)), nil
		}

		// Detect file type
		ext := strings.ToLower(filepath.Ext(input.Path))
		isImage := isImageFile(ext)
		isNotebook := ext == ".ipynb"

		// Special handling for notebooks
		if isNotebook {
			return aisdk.NewErrorToolResponse("for Jupyter notebooks (.ipynb files), use the NotebookRead tool instead"), nil
		}

		// For image files, return as multimodal content
		if isImage {
			return handleImageFile(fs, input.Path, info, ext)
		}

		// For text files, handle with line limits and potentially line numbers
		return handleTextFile(fs, input.Path, info, input.LineNumbers)
	}
}

// handleImageFile processes image files and returns multimodal content
func handleImageFile(fs afero.Fs, filePath string, info os.FileInfo, ext string) (*aisdk.ToolResponse, error) {
	// Check file size for images
	if info.Size() > maxFileSizeBytes {
		return aisdk.NewErrorToolResponse(fmt.Sprintf("image file too large: %s (max %s)", 
			toolsutil.FormatBytes(info.Size()), 
			toolsutil.FormatBytes(maxFileSizeBytes))), nil
	}

	// Read the image file
	imageBytes, err := afero.ReadFile(fs, filePath)
	if err != nil {
		toolsutil.GetLogger().Error("failed to read image file", "path", filePath, "error", err)
		return aisdk.NewErrorToolResponse(fmt.Sprintf("failed to read image file: %v", err)), nil
	}

	// Create multimodal response with image
	response := aisdk.CreateMixedToolResponse("")
	
	// Add the image as base64-encoded content
	format := strings.TrimPrefix(ext, ".")
	base64Data := base64.StdEncoding.EncodeToString(imageBytes)
	filename := filepath.Base(filePath)
	
	response.AddImage(format, base64Data, filename, info.Size())

	// Also add JSON metadata for compatibility
	metadata := ReadFileOutput{
		Content:  fmt.Sprintf("[Image file: %s]", filename),
		Path:     filePath,
		Size:     info.Size(),
		Language: "image",
		IsText:   false,
	}
	
	if err := response.AddJSON(metadata); err != nil {
		toolsutil.GetLogger().Warn("failed to add JSON metadata", "error", err)
	}

	toolsutil.GetLogger().Info("image file read successfully", 
		"path", filePath, 
		"size", info.Size(), 
		"format", format)

	return response, nil
}

// handleTextFile processes text files with line limits and optional line numbers
func handleTextFile(fs afero.Fs, filePath string, info os.FileInfo, includeLineNumbers bool) (*aisdk.ToolResponse, error) {
	// Open file
	file, err := fs.Open(filePath)
	if err != nil {
		toolsutil.GetLogger().Error("failed to open file", "path", filePath, "error", err)
		return aisdk.NewErrorToolResponse(fmt.Sprintf("failed to open file: %v", err)), nil
	}
	defer file.Close()

	// Read file content with limits
	content, truncated, err := readFileWithLimits(file, includeLineNumbers)
	if err != nil {
		toolsutil.GetLogger().Error("failed to read file", "path", filePath, "error", err)
		return aisdk.NewErrorToolResponse(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	// Check if file is empty
	if len(content) == 0 && info.Size() == 0 {
		content = "<system-reminder>This file exists but has empty contents.</system-reminder>"
	}

	// Add truncation notice if needed
	if truncated {
		content += fmt.Sprintf("\n\n[File truncated at %d lines]", maxLines)
	}

	// Detect language for syntax highlighting
	language := toolsutil.DetectLanguage(filePath, []byte(content))

	// Create multimodal response
	response := aisdk.CreateMixedToolResponse(content)

	// Add JSON metadata
	metadata := ReadFileOutput{
		Content:  content,
		Path:     filePath,
		Size:     info.Size(),
		Language: language,
		IsText:   true,
	}
	
	if err := response.AddJSON(metadata); err != nil {
		toolsutil.GetLogger().Warn("failed to add JSON metadata", "error", err)
	}

	toolsutil.GetLogger().Info("text file read successfully", 
		"path", filePath, 
		"size", info.Size(), 
		"language", language,
		"truncated", truncated)

	return response, nil
}

// parseToolInput is a helper to parse tool call input
func parseToolInput(call *aisdk.ToolCall, input interface{}) error {
	return json.Unmarshal(call.Function.Arguments, input)
}