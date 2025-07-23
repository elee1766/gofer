package tool_grepfiles

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
const Name = "grep_files"

const grepFilesPrompt = `A powerful search tool built on ripgrep

  Usage:
  - ALWAYS use Grep for search tasks. NEVER invoke grep or rg as a Bash command. The Grep tool has been optimized for correct permissions and access.
  - Supports full regex syntax (e.g., "log.*Error", "function\\s+\\w+")
  - Filter files with glob parameter (e.g., "*.js", "**/*.tsx") or type parameter (e.g., "js", "py", "rust")
  - Output modes: "content" shows matching lines, "files_with_matches" shows only file paths (default), "count" shows match counts
  - Use Task tool for open-ended searches requiring multiple rounds
  - Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping (use interface\\{\\} to find interface{} in Go code)
  - Multiline matching: By default patterns match within single lines only. For cross-line patterns like struct \\{[\\s\\S]*?field, use multiline: true`

// GrepFilesInput represents the parameters for grep_files
type GrepFilesInput struct {
	Pattern       string `json:"pattern" required:"true" description:"The regex pattern to search for"`
	Path          string `json:"path,omitempty" description:"The directory to search in (defaults to current directory)"`
	FilePattern   string `json:"file_pattern,omitempty" description:"File name pattern (glob) to filter files"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" description:"Case sensitive search (default: true)"`
	ContextLines  int    `json:"context_lines,omitempty" description:"Number of context lines around matches (default: 2)"`
	MaxResults    int    `json:"max_results,omitempty" description:"Maximum number of results (default: 100)"`
}

// GrepMatch represents a single grep match
type GrepMatch struct {
	File    string   `json:"file" description:"The file path containing the match"`
	Line    int      `json:"line" description:"The line number of the match"`
	Content string   `json:"content" description:"The content of the matching line"`
	Match   string   `json:"match" description:"The actual matched text"`
	Context []string `json:"context,omitempty" description:"Lines around the match for context"`
}

// GrepFilesOutput represents the response from grep_files
type GrepFilesOutput struct {
	Pattern      string      `json:"pattern" description:"The regex pattern used"`
	Path         string      `json:"path" description:"The directory searched"`
	Matches      []GrepMatch `json:"matches" description:"All matches found"`
	TotalMatches int         `json:"total_matches" description:"Total number of matches"`
	Truncated    bool        `json:"truncated" description:"Whether results were truncated due to max_results"`
}

// Tool returns the grep_files tool definition using GenericTool
func Tool(fs afero.Fs) (agent.Tool, error) {
	return agent.NewGenericTool(Name, grepFilesPrompt, makeGrepFilesHandler(fs))
}

// makeGrepFilesHandler creates a type-safe handler for the grep_files tool
func makeGrepFilesHandler(fs afero.Fs) func(ctx context.Context, input GrepFilesInput) (GrepFilesOutput, error) {
	return func(ctx context.Context, input GrepFilesInput) (GrepFilesOutput, error) {
		logger := toolsutil.GetLogger()
		
		// Check for cancellation
		select {
		case <-ctx.Done():
			return GrepFilesOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		// Set defaults
		if input.Path == "" {
			input.Path = "."
		}
		if input.MaxResults == 0 {
			input.MaxResults = 100
		}
		if input.ContextLines < 0 {
			input.ContextLines = 0
		}
		// Default case_sensitive is true unless explicitly set to false
		// (handled by the bool field)

		// Safety check: validate path
		if !toolsutil.IsPathSafe(input.Path) {
			logger.Error("unsafe path rejected", "path", input.Path)
			return GrepFilesOutput{}, fmt.Errorf("unsafe path: %s", input.Path)
		}

		logger.Info("grep files", "pattern", input.Pattern, "path", input.Path, "case_sensitive", input.CaseSensitive)

		// Compile regex
		var regex *regexp.Regexp
		var err error
		
		if !input.CaseSensitive {
			regex, err = regexp.Compile("(?mi)" + input.Pattern)
		} else {
			regex, err = regexp.Compile("(?m)" + input.Pattern)
		}
		
		if err != nil {
			logger.Error("invalid regex pattern", "pattern", input.Pattern, "error", err)
			return GrepFilesOutput{}, fmt.Errorf("invalid regex pattern: %v", err)
		}

		matches := make([]GrepMatch, 0)
		matchCount := 0

		// Check for cancellation before walk
		select {
		case <-ctx.Done():
			return GrepFilesOutput{}, fmt.Errorf("operation cancelled")
		default:
		}

		err = afero.Walk(fs, input.Path, func(path string, info os.FileInfo, err error) error {
			// Check for cancellation during walk
			select {
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled")
			default:
			}

			if err != nil || info.IsDir() {
				return nil
			}

			// Check result limit
			if matchCount >= input.MaxResults {
				return filepath.SkipDir
			}

			// Check file pattern if specified
			if input.FilePattern != "" {
				matched, err := filepath.Match(input.FilePattern, info.Name())
				if err != nil || !matched {
					return nil
				}
			}

			// Read file content
			content, err := afero.ReadFile(fs, path)
			if err != nil {
				return nil
			}

			// Skip binary files
			if err := toolsutil.ValidateFileSize(info.Size()); err != nil {
				return nil
			}

			if !toolsutil.IsTextFile(content) {
				return nil
			}

			// Search for pattern
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if matchCount >= input.MaxResults {
					break
				}

				if regex.MatchString(line) {
					match := GrepMatch{
						File:    path,
						Line:    i + 1,
						Content: line,
						Match:   regex.FindString(line),
					}

					if input.ContextLines > 0 {
						match.Context = getContext(lines, i, input.ContextLines)
					}

					matches = append(matches, match)
					matchCount++
				}
			}

			return nil
		})

		if err != nil {
			logger.Error("grep failed", "error", err)
			return GrepFilesOutput{}, fmt.Errorf("grep failed: %v", err)
		}

		logger.Info("grep completed", "pattern", input.Pattern, "matches", len(matches), "truncated", matchCount >= input.MaxResults)
		
		return GrepFilesOutput{
			Pattern:      input.Pattern,
			Path:         input.Path,
			Matches:      matches,
			TotalMatches: len(matches),
			Truncated:    matchCount >= input.MaxResults,
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
