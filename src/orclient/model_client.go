package orclient

import (
	"context"
	"fmt"

	"github.com/elee1766/gofer/src/aisdk"
)

var _ aisdk.ModelClient = (*ModelClient)(nil)

// ModelClient represents a client bound to a specific model
type ModelClient struct {
	client *Client
	model  *aisdk.ModelInfo
}

// Model creates a ModelClient bound to the specified model
func (c *Client) Model(ctx context.Context, modelName string) (aisdk.ModelClient, error) {
	// Get model information from cache or API
	modelInfo, err := c.modelCache.GetModel(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get model info for %s: %w", modelName, err)
	}

	return &ModelClient{
		client: c,
		model:  modelInfo,
	}, nil
}

// CreateChatCompletion creates a chat completion with the bound model
func (mc *ModelClient) CreateChatCompletion(ctx context.Context, req *aisdk.ChatCompletionRequest) (*aisdk.ChatCompletionResponse, error) {
	// Override the model in the request
	req.Model = mc.model.ID

	// Use the underlying client to make the request
	return mc.client.createChatCompletion(ctx, req)
}

// CreateChatCompletionStream creates a streaming chat completion with the bound model
func (mc *ModelClient) CreateChatCompletionStream(ctx context.Context, req *aisdk.ChatCompletionRequest) (aisdk.StreamInterface, error) {
	// Override the model in the request
	req.Model = mc.model.ID

	// Use the underlying client to make the streaming request
	return mc.client.createChatCompletionStream(ctx, req)
}

// GetModelInfo returns the model information
func (mc *ModelClient) GetModelInfo() *aisdk.ModelInfo {
	return mc.model
}
