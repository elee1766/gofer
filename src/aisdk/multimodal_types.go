package aisdk

import (
	"encoding/json"
	"fmt"
)

// ContentType represents the type of content in a multimodal message
type ContentType string

const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
	ContentTypeFile  ContentType = "file"
	ContentTypeJSON  ContentType = "json"
)

// ContentItem represents a single piece of content in a multimodal message
type ContentItem struct {
	Type ContentType `json:"type"`
	Data interface{} `json:"data"`
}

// TextContent represents text content
type TextContent struct {
	Text string `json:"text"`
}

// ImageContent represents image content
type ImageContent struct {
	Format   string `json:"format"`          // "png", "jpeg", "gif", etc.
	Data     string `json:"data"`            // base64 encoded image data
	Filename string `json:"filename,omitempty"` // original filename if available
	Size     int64  `json:"size,omitempty"`     // file size in bytes
}

// FileContent represents file content (for non-image files)
type FileContent struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Data     string `json:"data"` // base64 encoded for binary, plain text for text files
}

// JSONContent represents structured JSON data
type JSONContent struct {
	Data json.RawMessage `json:"data"`
}

// MultimodalContent represents content that can contain multiple types of data
type MultimodalContent struct {
	Items []ContentItem `json:"items"`
}

// NewTextContent creates a new text content item
func NewTextContent(text string) ContentItem {
	return ContentItem{
		Type: ContentTypeText,
		Data: TextContent{Text: text},
	}
}

// NewImageContent creates a new image content item
func NewImageContent(format, data, filename string, size int64) ContentItem {
	return ContentItem{
		Type: ContentTypeImage,
		Data: ImageContent{
			Format:   format,
			Data:     data,
			Filename: filename,
			Size:     size,
		},
	}
}

// NewFileContent creates a new file content item
func NewFileContent(filename, mimeType, data string, size int64) ContentItem {
	return ContentItem{
		Type: ContentTypeFile,
		Data: FileContent{
			Filename: filename,
			MimeType: mimeType,
			Data:     data,
			Size:     size,
		},
	}
}

// NewJSONContent creates a new JSON content item
func NewJSONContent(data interface{}) (ContentItem, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ContentItem{}, fmt.Errorf("failed to marshal JSON data: %w", err)
	}
	
	return ContentItem{
		Type: ContentTypeJSON,
		Data: JSONContent{Data: jsonData},
	}, nil
}

// AddText adds a text item to the multimodal content
func (mc *MultimodalContent) AddText(text string) {
	mc.Items = append(mc.Items, NewTextContent(text))
}

// AddImage adds an image item to the multimodal content
func (mc *MultimodalContent) AddImage(format, data, filename string, size int64) {
	mc.Items = append(mc.Items, NewImageContent(format, data, filename, size))
}

// AddFile adds a file item to the multimodal content
func (mc *MultimodalContent) AddFile(filename, mimeType, data string, size int64) {
	mc.Items = append(mc.Items, NewFileContent(filename, mimeType, data, size))
}

// AddJSON adds a JSON item to the multimodal content
func (mc *MultimodalContent) AddJSON(data interface{}) error {
	item, err := NewJSONContent(data)
	if err != nil {
		return err
	}
	mc.Items = append(mc.Items, item)
	return nil
}

// HasImages returns true if the content contains any images
func (mc *MultimodalContent) HasImages() bool {
	for _, item := range mc.Items {
		if item.Type == ContentTypeImage {
			return true
		}
	}
	return false
}

// GetTextOnly returns all text content concatenated together
func (mc *MultimodalContent) GetTextOnly() string {
	var result string
	for _, item := range mc.Items {
		if item.Type == ContentTypeText {
			if textData, ok := item.Data.(TextContent); ok {
				if result != "" {
					result += "\n"
				}
				result += textData.Text
			}
		}
	}
	return result
}

// ToLegacyString converts multimodal content to a single string for backward compatibility
func (mc *MultimodalContent) ToLegacyString() string {
	if len(mc.Items) == 0 {
		return ""
	}
	
	// If only one text item, return it directly
	if len(mc.Items) == 1 && mc.Items[0].Type == ContentTypeText {
		if textData, ok := mc.Items[0].Data.(TextContent); ok {
			return textData.Text
		}
	}
	
	// For multiple items or non-text items, create a summary
	var result string
	for i, item := range mc.Items {
		if i > 0 {
			result += "\n"
		}
		
		switch item.Type {
		case ContentTypeText:
			if textData, ok := item.Data.(TextContent); ok {
				result += textData.Text
			}
		case ContentTypeImage:
			if imgData, ok := item.Data.(ImageContent); ok {
				result += fmt.Sprintf("[Image: %s, Format: %s, Size: %d bytes]", 
					imgData.Filename, imgData.Format, imgData.Size)
			}
		case ContentTypeFile:
			if fileData, ok := item.Data.(FileContent); ok {
				result += fmt.Sprintf("[File: %s, Type: %s, Size: %d bytes]", 
					fileData.Filename, fileData.MimeType, fileData.Size)
			}
		case ContentTypeJSON:
			result += "[JSON Data]"
		}
	}
	
	return result
}