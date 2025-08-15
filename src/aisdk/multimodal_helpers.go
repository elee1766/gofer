package aisdk

import (
	"encoding/base64"
	"fmt"
)

// NewSuccessToolResponse creates a successful tool response with multimodal content
func NewSuccessToolResponse() *ToolResponse {
	return &ToolResponse{
		Type:              "success",
		MultimodalContent: &MultimodalContent{Items: []ContentItem{}},
		IsError:           false,
	}
}

// NewErrorToolResponse creates an error tool response with a text message
func NewErrorToolResponse(message string) *ToolResponse {
	response := &ToolResponse{
		Type:              "error",
		MultimodalContent: &MultimodalContent{Items: []ContentItem{}},
		IsError:           true,
	}
	response.MultimodalContent.AddText(message)
	response.SetMultimodalContent(response.MultimodalContent) // Update legacy content
	return response
}

// NewTextToolResponse creates a tool response with just text content
func NewTextToolResponse(text string) *ToolResponse {
	response := NewSuccessToolResponse()
	response.MultimodalContent.AddText(text)
	response.SetMultimodalContent(response.MultimodalContent)
	return response
}

// NewImageToolResponse creates a tool response with image content
func NewImageToolResponse(format, data, filename string, size int64) *ToolResponse {
	response := NewSuccessToolResponse()
	response.MultimodalContent.AddImage(format, data, filename, size)
	response.SetMultimodalContent(response.MultimodalContent)
	return response
}

// NewJSONToolResponse creates a tool response with JSON content
func NewJSONToolResponse(data interface{}) (*ToolResponse, error) {
	response := NewSuccessToolResponse()
	if err := response.MultimodalContent.AddJSON(data); err != nil {
		return nil, fmt.Errorf("failed to create JSON tool response: %w", err)
	}
	response.SetMultimodalContent(response.MultimodalContent)
	return response, nil
}

// CreateImageToolResponse creates a tool response with base64-encoded image
func CreateImageToolResponse(imageBytes []byte, format, filename string) *ToolResponse {
	base64Data := base64.StdEncoding.EncodeToString(imageBytes)
	return NewImageToolResponse(format, base64Data, filename, int64(len(imageBytes)))
}

// CreateMixedToolResponse creates a tool response with both text and other content
func CreateMixedToolResponse(text string) *ToolResponse {
	response := NewSuccessToolResponse()
	if text != "" {
		response.MultimodalContent.AddText(text)
	}
	return response
}

// AddImageToResponse adds an image to an existing tool response
func (tr *ToolResponse) AddImage(format, data, filename string, size int64) {
	if tr.MultimodalContent == nil {
		tr.MultimodalContent = &MultimodalContent{Items: []ContentItem{}}
	}
	tr.MultimodalContent.AddImage(format, data, filename, size)
	tr.SetMultimodalContent(tr.MultimodalContent)
}

// AddTextToResponse adds text to an existing tool response
func (tr *ToolResponse) AddText(text string) {
	if tr.MultimodalContent == nil {
		tr.MultimodalContent = &MultimodalContent{Items: []ContentItem{}}
	}
	tr.MultimodalContent.AddText(text)
	tr.SetMultimodalContent(tr.MultimodalContent)
}

// AddJSONToResponse adds JSON data to an existing tool response
func (tr *ToolResponse) AddJSON(data interface{}) error {
	if tr.MultimodalContent == nil {
		tr.MultimodalContent = &MultimodalContent{Items: []ContentItem{}}
	}
	if err := tr.MultimodalContent.AddJSON(data); err != nil {
		return err
	}
	tr.SetMultimodalContent(tr.MultimodalContent)
	return nil
}