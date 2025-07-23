package orclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/elee1766/gofer/src/aisdk"
)

// ModelsResponse represents the response from the OpenRouter models API
type ModelsResponse struct {
	Data []*aisdk.ModelInfo `json:"data"`
}

// getModelInfo fetches model information from the OpenRouter API
func (c *Client) getModelInfo(ctx context.Context, modelName string) (*aisdk.ModelInfo, error) {
	url := c.baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Find the requested model
	for _, model := range modelsResp.Data {
		if model.ID == modelName {
			return model, nil
		}
	}

	return nil, fmt.Errorf("model %s not found", modelName)
}

// ListModels returns all available models (with caching)
func (c *Client) ListModels(ctx context.Context) ([]*aisdk.ModelInfo, error) {
	return c.modelCache.GetModelList(ctx)
}

// listModelsUncached returns all available models without caching
func (c *Client) listModelsUncached(ctx context.Context) ([]*aisdk.ModelInfo, error) {
	url := c.baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return modelsResp.Data, nil
}

// GetModelByID returns a specific model by ID
func (c *Client) GetModelByID(ctx context.Context, modelID string) (*aisdk.ModelInfo, error) {
	return c.getModelInfo(ctx, modelID)
}

// FindModelByName searches for a model by name (case-insensitive)
func (c *Client) FindModelByName(ctx context.Context, name string) (*aisdk.ModelInfo, error) {
	models, err := c.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	searchName := strings.ToLower(name)

	// First try exact match on ID
	for _, model := range models {
		if strings.ToLower(model.ID) == searchName {
			return model, nil
		}
	}

	// Then try partial match on ID or name
	for _, model := range models {
		if strings.Contains(strings.ToLower(model.ID), searchName) ||
			strings.Contains(strings.ToLower(model.Name), searchName) {
			return model, nil
		}
	}

	return nil, fmt.Errorf("model matching %s not found", name)
}
