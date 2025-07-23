package tool_searchfiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
	"github.com/spf13/afero"
)

// Tool name constant
const Name = "search_files"

const searchFilesPrompt = `- Fast file pattern matching tool that works with any codebase size
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns
- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Agent tool instead
- You have the capability to call multiple tools in a single response. It is always better to speculatively perform multiple searches as a batch that are potentially useful.`

// SearchFilesInput represents the parameters for search_files
type SearchFilesInput struct {
	Pattern     string `json:"pattern" required:"true" description:"The search pattern (regex or string)"`
	Path        string `json:"path,omitempty" description:"The directory to search in (defaults to current directory)"`
	FilePattern string `json:"file_pattern,omitempty" description:"File name pattern (glob) to filter files"`
}

// SearchMatch represents a single search match
type SearchMatch struct {
	File    string   `json:"file" description:"The file path containing the match"`
	Line    int      `json:"line" description:"The line number of the match"`
	Content string   `json:"content" description:"The content of the matching line"`
	Context []string `json:"context" description:"Lines around the match for context"`
}

// SearchFilesOutput represents the response from search_files
type SearchFilesOutput struct {
	Pattern string        `json:"pattern" description:"The search pattern used"`
	Path    string        `json:"path" description:"The directory searched"`
	Matches []SearchMatch `json:"matches" description:"All matches found"`
	Count   int           `json:"count" description:"Total number of matches"`
}

// Tool returns the search_files tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, searchFilesPrompt, makeSearchFilesHandler(fs))
}

// makeSearchFilesHandler creates a type-safe handler for the search_files tool
func makeSearchFilesHandler(fs afero.Fs) func(ctx context.Context, input SearchFilesInput) (SearchFilesOutput, error) {
	return func(ctx context.Context, input SearchFilesInput) (SearchFilesOutput, error) {
		logger := toolsutil.GetLogger()
		
		// Check for cancellation
		select {
		case <-ctx.Done():
			return SearchFilesOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Default path if not specified
		if input.Path == "" {
			input.Path = "."
		}

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			logger.Error("unsafe path rejected", "path", input.Path)
			return SearchFilesOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		logger.Info("searching files", "pattern", input.Pattern, "path", input.Path, "file_pattern", input.FilePattern)

		matches := make([]SearchMatch, 0)

		// Check for cancellation before walk
		select {
		case <-ctx.Done():
			return SearchFilesOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		err := afero.Walk(fs, input.Path, func(path string, info os.FileInfo, err error) error {
			// Check for cancellation during walk
			select {
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled")
			default:
			}

			if err != nil || info.IsDir() {
				return nil
			}

			// Check file pattern if specified
			if input.FilePattern != "" {
				matched, err := filepath.Match(input.FilePattern, info.Name())
				if err != nil || !matched {
					return nil
				}
			}

			// Skip non-text files to avoid binary content issues
			content, err := afero.ReadFile(fs, path)
			if err != nil {
				return nil
			}

			if !toolsutil.IsTextFile(content) {
				return nil
			}

			// Search for pattern in content
			lines := strings.Split(string(content), "\n")
			regex, err := regexp.Compile(input.Pattern)
			if err != nil {
				// Fall back to simple string search if regex is invalid
				for i, line := range lines {
					if strings.Contains(line, input.Pattern) {
						matches = append(matches, SearchMatch{
							File:    path,
							Line:    i + 1,
							Content: line,
							Context: getContext(lines, i, 2),
						})
					}
				}
			} else {
				for i, line := range lines {
					if regex.MatchString(line) {
						matches = append(matches, SearchMatch{
							File:    path,
							Line:    i + 1,
							Content: line,
							Context: getContext(lines, i, 2),
						})
					}
				}
			}

			return nil
		})

		if err != nil {
			logger.Error("search failed", "error", err)
			return SearchFilesOutput{}, fmt.Errorf("search failed: %v", err)
		}

		logger.Info("search completed", "pattern", input.Pattern, "matches", len(matches))
		
		return SearchFilesOutput{
			Pattern: input.Pattern,
			Path:    input.Path,
			Matches: matches,
			Count:   len(matches),
		}, nil
	}
}

// getContext returns lines around the match
func getContext(lines []string, index, contextSize int) []string {
	start := index - contextSize
	if start < 0 {
		start = 0
	}
	end := index + contextSize + 1
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end]
}
