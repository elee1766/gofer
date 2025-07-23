package agent

import (
	"context"
	"fmt"
	"reflect"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/swaggest/jsonschema-go"
)

// GenericTool represents a type-safe tool with schema validation that implements the Tool interface
// Note: Currently delegates to GenericToolFixed due to kaptinlin/jsonschema i18n bug
type GenericTool[TInput any, TOutput any] struct {
	Type        string
	Name        string
	Description string
	InputType   reflect.Type
	OutputType  reflect.Type
	Schema      *jsonschema.Schema
	Handler     GenericToolHandler[TInput, TOutput]
}

// GenericToolHandler is a type-safe handler function
type GenericToolHandler[TInput any, TOutput any] func(ctx context.Context, input TInput) (TOutput, error)

// GetType returns the tool type (always "function" for now)
func (gt *GenericTool[TInput, TOutput]) GetType() string {
	return gt.Type
}

// GetName returns the tool's name
func (gt *GenericTool[TInput, TOutput]) GetName() string {
	return gt.Name
}

// GetDescription returns the tool's description
func (gt *GenericTool[TInput, TOutput]) GetDescription() string {
	return gt.Description
}

// GetParameters returns the JSON schema for the tool's parameters
func (gt *GenericTool[TInput, TOutput]) GetParameters() *jsonschema.Schema {
	return gt.Schema
}

// Execute runs the tool with the given parameters
// Note: This is kept for interface compatibility but NewGenericTool returns GenericToolFixed
func (gt *GenericTool[TInput, TOutput]) Execute(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
	// This method should not be called since NewGenericTool returns GenericToolFixed
	return nil, fmt.Errorf("GenericTool.Execute should not be called - using GenericToolFixed instead")
}

// NewGenericTool creates a new generic tool with automatic schema generation
// Note: This currently uses GenericToolFixed to work around a kaptinlin/jsonschema i18n bug
func NewGenericTool[TInput any, TOutput any](name, description string, handler GenericToolHandler[TInput, TOutput]) (Tool, error) {
	// Use GenericToolFixed to work around kaptinlin/jsonschema i18n bug
	return NewGenericToolFixed(name, description, handler)
}

// MustNewGenericTool creates a new generic tool and panics on error
func MustNewGenericTool[TInput any, TOutput any](name, description string, handler GenericToolHandler[TInput, TOutput]) Tool {
	tool, err := NewGenericTool(name, description, handler)
	if err != nil {
		panic(fmt.Sprintf("failed to create generic tool: %v", err))
	}
	return tool
}


// Ensure GenericTool implements the Tool interface
var _ Tool = (*GenericTool[any, any])(nil)

// Example tool input/output types with JSON schema tags
type FileReadInput struct {
	Path string `json:"path" jsonschema:"required,description=The file path to read"`
}

type FileReadOutput struct {
	Content string `json:"content" jsonschema:"description=The file contents"`
	Size    int64  `json:"size" jsonschema:"description=File size in bytes"`
}

type SearchInput struct {
	Query      string `json:"query" jsonschema:"required,description=Search query"`
	Path       string `json:"path,omitempty" jsonschema:"description=Directory to search in"`
	FileType   string `json:"file_type,omitempty" jsonschema:"description=File extension filter"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"default=10,minimum=1,maximum=100,description=Maximum number of results"`
}

type SearchOutput struct {
	Results []SearchResult `json:"results" jsonschema:"description=Search results"`
	Total   int            `json:"total" jsonschema:"description=Total number of matches"`
}

type SearchResult struct {
	Path    string `json:"path" jsonschema:"description=File path"`
	Line    int    `json:"line" jsonschema:"description=Line number"`
	Content string `json:"content" jsonschema:"description=Matching line content"`
}

// Example of creating a type-safe file read tool
func CreateFileReadTool() (Tool, error) {
	return NewGenericTool("read_file", "Read the contents of a file",
		func(ctx context.Context, input FileReadInput) (FileReadOutput, error) {
			// Implementation would read the file
			// This is just an example
			return FileReadOutput{
				Content: "File contents here",
				Size:    1024,
			}, nil
		})
}

// Example of creating a type-safe search tool
func CreateSearchTool() (Tool, error) {
	return NewGenericTool("search_files", "Search for files containing a pattern",
		func(ctx context.Context, input SearchInput) (SearchOutput, error) {
			// Implementation would perform the search
			// This is just an example
			return SearchOutput{
				Results: []SearchResult{
					{
						Path:    "/example/file.go",
						Line:    42,
						Content: "matching line content",
					},
				},
				Total: 1,
			}, nil
		})
}
