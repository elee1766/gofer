package tool_readfile

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

const (
	maxLines         = 2000
	maxLineLength    = 2000
	maxFileSizeBytes = 10 * 1024 * 1024 // 10MB for binary files
)

// makeReadFileHandlerV2 creates a type-safe handler that matches the prompt description
func makeReadFileHandlerV2(fs afero.Fs) func(ctx context.Context, input ReadFileInput) (ReadFileOutput, error) {
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

		toolsutil.GetLogger().Info("reading file", "path", input.Path, "line_numbers", input.LineNumbers)

		// Check if file exists and get info
		info, err := fs.Stat(input.Path)
		if err != nil {
			toolsutil.GetLogger().Error("file not found", "path", input.Path, "error", err)
			return ReadFileOutput{}, fmt.Errorf("file not found: %s", input.Path)
		}

		// Check if it's a directory
		if info.IsDir() {
			return ReadFileOutput{}, fmt.Errorf("path is a directory, not a file: %s", input.Path)
		}

		// Detect file type
		ext := strings.ToLower(filepath.Ext(input.Path))
		isImage := isImageFile(ext)
		isNotebook := ext == ".ipynb"

		// Special handling for notebooks
		if isNotebook {
			return ReadFileOutput{}, fmt.Errorf("for Jupyter notebooks (.ipynb files), use the NotebookRead tool instead")
		}

		// For image files, we need different handling
		if isImage {
			// Check file size for images
			if info.Size() > maxFileSizeBytes {
				return ReadFileOutput{}, fmt.Errorf("image file too large: %s (max %s)", 
					toolsutil.FormatBytes(info.Size()), 
					toolsutil.FormatBytes(maxFileSizeBytes))
			}

			// Read the entire image file
			_, err = afero.ReadFile(fs, input.Path)
			if err != nil {
				toolsutil.GetLogger().Error("failed to read image file", "path", input.Path, "error", err)
				return ReadFileOutput{}, fmt.Errorf("failed to read image file: %v", err)
			}

			// For images, return a special format indicating it's an image
			// In a real implementation, this would be handled by the multimodal LLM
			imageInfo := fmt.Sprintf("[Image file: %s, Size: %s, Format: %s]", 
				filepath.Base(input.Path), 
				toolsutil.FormatBytes(info.Size()),
				strings.TrimPrefix(ext, "."))

			return ReadFileOutput{
				Content:  imageInfo,
				Path:     input.Path,
				Size:     info.Size(),
				Language: "image",
				IsText:   false,
			}, nil
		}

		// For text files, read with line limits
		file, err := fs.Open(input.Path)
		if err != nil {
			toolsutil.GetLogger().Error("failed to open file", "path", input.Path, "error", err)
			return ReadFileOutput{}, fmt.Errorf("failed to open file: %v", err)
		}
		defer file.Close()

		// Read file content with limits
		content, truncated, err := readFileWithLimits(file, input.LineNumbers)
		if err != nil {
			toolsutil.GetLogger().Error("failed to read file", "path", input.Path, "error", err)
			return ReadFileOutput{}, fmt.Errorf("failed to read file: %v", err)
		}

		// Check if file is empty
		if len(content) == 0 && info.Size() == 0 {
			// Return with a system reminder for empty files
			content = "<system-reminder>This file exists but has empty contents.</system-reminder>"
		}

		// Detect language for syntax highlighting
		language := toolsutil.DetectLanguage(input.Path, []byte(content))

		// Add truncation notice if needed
		if truncated {
			content += fmt.Sprintf("\n\n[File truncated at %d lines]", maxLines)
		}

		toolsutil.GetLogger().Info("file read successfully", 
			"path", input.Path, 
			"size", info.Size(), 
			"language", language,
			"truncated", truncated)

		return ReadFileOutput{
			Content:  content,
			Path:     input.Path,
			Size:     info.Size(),
			Language: language,
			IsText:   true,
		}, nil
	}
}

// readFileWithLimits reads a file respecting line and character limits
func readFileWithLimits(file afero.File, includeLineNumbers bool) (string, bool, error) {
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 0
	truncated := false

	// Custom split function to handle very long lines
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		// Truncate lines that are too long
		if len(token) > maxLineLength {
			token = token[:maxLineLength]
			truncated = true
		}
		return
	})

	// Set a reasonable buffer size
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lineNum++
		if lineNum > maxLines {
			truncated = true
			break
		}

		line := scanner.Text()
		
		// Add line numbers if requested
		if includeLineNumbers {
			line = fmt.Sprintf("%d: %s", lineNum, line)
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return "", false, err
	}

	return strings.Join(lines, "\n"), truncated, nil
}

// isImageFile checks if the file extension indicates an image
func isImageFile(ext string) bool {
	imageExts := []string{
		".png", ".jpg", ".jpeg", ".gif", ".bmp", 
		".svg", ".webp", ".ico", ".tiff", ".tif",
		".heic", ".heif", ".avif", ".jfif",
	}
	
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}