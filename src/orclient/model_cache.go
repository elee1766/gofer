package orclient

import (
	"context"
	"sync"
	"time"

	"github.com/elee1766/gofer/src/aisdk"
)

// ModelCache provides caching for model information
type ModelCache struct {
	cache       map[string]*cachedModel
	listCache   *cachedModelList
	mu          sync.RWMutex
	ttl         time.Duration
	client      *Client
}

type cachedModel struct {
	model     *aisdk.ModelInfo
	fetchedAt time.Time
}

type cachedModelList struct {
	models    []*aisdk.ModelInfo
	fetchedAt time.Time
}

// NewModelCache creates a new model cache
func NewModelCache(client *Client, ttl time.Duration) *ModelCache {
	return &ModelCache{
		cache:  make(map[string]*cachedModel),
		ttl:    ttl,
		client: client,
	}
}

// GetModel gets a model from cache or fetches it
func (mc *ModelCache) GetModel(ctx context.Context, modelID string) (*aisdk.ModelInfo, error) {
	mc.mu.RLock()
	cached, exists := mc.cache[modelID]
	mc.mu.RUnlock()

	// Check if we have a valid cached entry
	if exists && time.Since(cached.fetchedAt) < mc.ttl {
		return cached.model, nil
	}

	// Fetch from API
	model, err := mc.client.getModelInfo(ctx, modelID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	mc.mu.Lock()
	mc.cache[modelID] = &cachedModel{
		model:     model,
		fetchedAt: time.Now(),
	}
	mc.mu.Unlock()

	return model, nil
}

// ClearCache clears the entire cache
func (mc *ModelCache) ClearCache() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cache = make(map[string]*cachedModel)
	mc.listCache = nil
}

// RemoveModel removes a specific model from cache
func (mc *ModelCache) RemoveModel(modelID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	delete(mc.cache, modelID)
}

// GetCacheStats returns cache statistics
func (mc *ModelCache) GetCacheStats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var validEntries, expiredEntries int
	now := time.Now()

	for _, cached := range mc.cache {
		if now.Sub(cached.fetchedAt) < mc.ttl {
			validEntries++
		} else {
			expiredEntries++
		}
	}
	
	// Check list cache
	listCacheValid := mc.listCache != nil && now.Sub(mc.listCache.fetchedAt) < mc.ttl

	return CacheStats{
		TotalEntries:   len(mc.cache),
		ValidEntries:   validEntries,
		ExpiredEntries: expiredEntries,
		TTL:            mc.ttl,
		ListCacheValid: listCacheValid,
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries   int           `json:"total_entries"`
	ValidEntries   int           `json:"valid_entries"`
	ExpiredEntries int           `json:"expired_entries"`
	TTL            time.Duration `json:"ttl"`
	ListCacheValid bool          `json:"list_cache_valid"`
}

// CleanupExpired removes expired entries from cache
func (mc *ModelCache) CleanupExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	for modelID, cached := range mc.cache {
		if now.Sub(cached.fetchedAt) >= mc.ttl {
			delete(mc.cache, modelID)
		}
	}
	
	// Also check list cache
	if mc.listCache != nil && now.Sub(mc.listCache.fetchedAt) >= mc.ttl {
		mc.listCache = nil
	}
}

// GetModelList gets the model list from cache or fetches it
func (mc *ModelCache) GetModelList(ctx context.Context) ([]*aisdk.ModelInfo, error) {
	mc.mu.RLock()
	cached := mc.listCache
	mc.mu.RUnlock()

	// Check if we have a valid cached entry
	if cached != nil && time.Since(cached.fetchedAt) < mc.ttl {
		return cached.models, nil
	}

	// Fetch from API - we need to call the underlying list method
	models, err := mc.client.listModelsUncached(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result
	mc.mu.Lock()
	mc.listCache = &cachedModelList{
		models:    models,
		fetchedAt: time.Now(),
	}
	mc.mu.Unlock()

	return models, nil
}

