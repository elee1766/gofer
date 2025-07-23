package aisdk

import (
	"context"
)

// Provider represents an AI provider interface
type Provider interface {
	GetModels(ctx context.Context) ([]*ModelInfo, error)
	Model(ctx context.Context, modelName string) (ModelClient, error)
}

// ModelClient represents a client for a specific model
type ModelClient interface {
	CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (StreamInterface, error)
	GetModelInfo() *ModelInfo
}
