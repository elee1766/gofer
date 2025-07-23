package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/elee1766/gofer/src/aisdk"
	"github.com/swaggest/jsonschema-go"
)

// GenericToolFixed is a version of GenericTool that works around the kaptinlin/jsonschema i18n bug
type GenericToolFixed[TInput any, TOutput any] struct {
	Type        string
	Name        string
	Description string
	InputType   reflect.Type
	OutputType  reflect.Type
	Schema      *jsonschema.Schema
	Handler     GenericToolHandler[TInput, TOutput]
}

// GetType returns the tool type (always "function" for now)
func (gt *GenericToolFixed[TInput, TOutput]) GetType() string {
	return gt.Type
}

// GetName returns the tool's name
func (gt *GenericToolFixed[TInput, TOutput]) GetName() string {
	return gt.Name
}

// GetDescription returns the tool's description
func (gt *GenericToolFixed[TInput, TOutput]) GetDescription() string {
	return gt.Description
}

// GetParameters returns the JSON schema for the tool's parameters
func (gt *GenericToolFixed[TInput, TOutput]) GetParameters() *jsonschema.Schema {
	return gt.Schema
}

// Execute runs the tool with the given parameters
func (gt *GenericToolFixed[TInput, TOutput]) Execute(ctx context.Context, call *aisdk.ToolCall) (*aisdk.ToolResponse, error) {
	// Skip kaptinlin validation and just use Go's type system
	
	// Parse and validate input
	var input TInput
	if err := json.Unmarshal(call.Function.Arguments, &input); err != nil {
		return &aisdk.ToolResponse{
			Type:    "error",
			Content: []byte(fmt.Sprintf("failed to parse input: %v", err)),
			IsError: true,
		}, nil // Return nil error to match LegacyTool behavior
	}

	// Validate required fields using reflection
	if err := gt.validateRequired(input); err != nil {
		return &aisdk.ToolResponse{
			Type:    "error",
			Content: []byte(fmt.Sprintf("validation failed: %v", err)),
			IsError: true,
		}, nil // Return nil error to match LegacyTool behavior
	}

	// Execute handler
	output, err := gt.Handler(ctx, input)
	if err != nil {
		return &aisdk.ToolResponse{
			Type:    "error",
			Content: []byte(err.Error()),
			IsError: true,
		}, nil // Return nil error to match LegacyTool behavior
	}

	// Marshal output
	content, err := json.Marshal(output)
	if err != nil {
		return &aisdk.ToolResponse{
			Type:    "error",
			Content: []byte(fmt.Sprintf("failed to marshal result: %v", err)),
			IsError: true,
		}, nil // Return nil error to match LegacyTool behavior
	}

	return &aisdk.ToolResponse{
		Type:    "success",
		Content: content,
		IsError: false,
	}, nil
}

// validateRequired checks that required fields are not empty
func (gt *GenericToolFixed[TInput, TOutput]) validateRequired(input TInput) error {
	if gt.Schema == nil || gt.Schema.Required == nil {
		return nil
	}

	val := reflect.ValueOf(input)
	typ := val.Type()

	// For each required field
	for _, requiredField := range gt.Schema.Required {
		// Find the struct field
		found := false
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			jsonTag := field.Tag.Get("json")
			fieldName := strings.Split(jsonTag, ",")[0]
			
			if fieldName == requiredField {
				found = true
				fieldValue := val.Field(i)
				
				// Check if the field is zero value
				if fieldValue.IsZero() {
					return fmt.Errorf("required field '%s' is missing", requiredField)
				}
				break
			}
		}
		
		if !found {
			return fmt.Errorf("required field '%s' not found in struct", requiredField)
		}
	}

	return nil
}

// NewGenericToolFixed creates a new generic tool without the kaptinlin validation bug
func NewGenericToolFixed[TInput any, TOutput any](name, description string, handler GenericToolHandler[TInput, TOutput]) (*GenericToolFixed[TInput, TOutput], error) {
	var input TInput
	inputType := reflect.TypeOf(input)

	// Ensure input type is a struct
	if inputType.Kind() == reflect.Ptr {
		if inputType.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("tool input type must be a struct, got %s", inputType.Elem().Kind())
		}
	} else if inputType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("tool input type must be a struct, got %s", inputType.Kind())
	}

	// Check output type is also a struct
	var output TOutput
	outputType := reflect.TypeOf(output)
	if outputType.Kind() == reflect.Ptr {
		if outputType.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("tool output type must be a struct, got %s", outputType.Elem().Kind())
		}
	} else if outputType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("tool output type must be a struct, got %s", outputType.Kind())
	}

	// Generate JSON Schema from the input type
	reflector := jsonschema.Reflector{}
	schema, err := reflector.Reflect(input)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	return &GenericToolFixed[TInput, TOutput]{
		Type:        "function",
		Name:        name,
		Description: description,
		InputType:   inputType,
		OutputType:  outputType,
		Schema:      &schema,
		Handler:     handler,
	}, nil
}

// Ensure GenericToolFixed implements the Tool interface
var _ Tool = (*GenericToolFixed[any, any])(nil)