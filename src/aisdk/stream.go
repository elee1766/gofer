package aisdk

import (
	"errors"
	"io"
	"strings"
)

// StreamCallback is a function called for each chunk in a stream.
type StreamCallback func(chunk *StreamChunk) error

// StreamToCallback reads a stream and calls the callback for each chunk.
func StreamToCallback(stream StreamInterface, callback StreamCallback) error {
	defer stream.Close()
	
	for {
		chunk, err := stream.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil // End of stream
			}
			return err
		}
		
		if chunk == nil {
			return nil // End of stream
		}
		
		if err := callback(chunk); err != nil {
			return err
		}
	}
}

// CollectStreamContent reads a stream and collects all content into a single string.
func CollectStreamContent(stream StreamInterface) (string, error) {
	var content strings.Builder
	
	err := StreamToCallback(stream, func(chunk *StreamChunk) error {
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
		return nil
	})
	
	return content.String(), err
}

// StreamToChannel converts a StreamInterface to a Go channel.
func StreamToChannel(stream StreamInterface) <-chan StreamResult {
	ch := make(chan StreamResult, 1)
	
	go func() {
		defer close(ch)
		defer stream.Close()
		
		for {
			chunk, err := stream.Read()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					ch <- StreamResult{Error: err}
				}
				return
			}
			
			if chunk == nil {
				return // End of stream
			}
			
			ch <- StreamResult{Chunk: chunk}
		}
	}()
	
	return ch
}

// StreamResult represents a result from a streaming operation.
type StreamResult struct {
	Chunk *StreamChunk
	Error error
}

// IsError returns true if this result contains an error.
func (r StreamResult) IsError() bool {
	return r.Error != nil
}

// IsChunk returns true if this result contains a chunk.
func (r StreamResult) IsChunk() bool {
	return r.Chunk != nil
}

// StreamAggregator helps aggregate streaming responses into a final response.
type StreamAggregator struct {
	ID       string
	Object   string
	Created  int64
	Model    string
	Content  strings.Builder
	
	// Tracking state
	FinishReason string
	Usage        *Usage
}

// NewStreamAggregator creates a new stream aggregator.
func NewStreamAggregator() *StreamAggregator {
	return &StreamAggregator{
		Object: "chat.completion",
	}
}

// AddChunk processes a stream chunk and updates the aggregated state.
func (a *StreamAggregator) AddChunk(chunk *StreamChunk) {
	if a.ID == "" {
		a.ID = chunk.ID
	}
	if a.Created == 0 {
		a.Created = chunk.Created
	}
	if a.Model == "" {
		a.Model = chunk.Model
	}
	
	if len(chunk.Choices) > 0 {
		choice := chunk.Choices[0]
		
		if choice.Delta != nil && choice.Delta.Content != "" {
			a.Content.WriteString(choice.Delta.Content)
		}
		
		if choice.FinishReason != "" {
			a.FinishReason = choice.FinishReason
		}
	}
}

// ToResponse converts the aggregated stream into a ChatCompletionResponse.
func (a *StreamAggregator) ToResponse() *ChatCompletionResponse {
	content := a.Content.String()
	
	response := &ChatCompletionResponse{
		ID:      a.ID,
		Object:  a.Object,
		Created: a.Created,
		Model:   a.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: a.FinishReason,
			},
		},
	}
	
	if a.Usage != nil {
		response.Usage = *a.Usage
	}
	
	return response
}

// AggregateStream reads a stream and returns the aggregated response.
func AggregateStream(stream StreamInterface) (*ChatCompletionResponse, error) {
	aggregator := NewStreamAggregator()
	
	err := StreamToCallback(stream, func(chunk *StreamChunk) error {
		aggregator.AddChunk(chunk)
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return aggregator.ToResponse(), nil
}