package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// ModelsCache is a specialized cache for LLM models
type ModelsCache struct {
	cache       *Cache
	modelsTTL   time.Duration
	embedTTL    time.Duration
	embeddings  *Cache
}

// ModelsConfig holds configuration for the models cache
type ModelsConfig struct {
	ModelsTTL   time.Duration // TTL for model list (default: 1 hour)
	EmbedTTL    time.Duration // TTL for embeddings (default: 24 hours)
	MaxEmbeddings int         // Max cached embeddings (default: 10000)
}

// DefaultModelsConfig returns default models cache configuration
func DefaultModelsConfig() ModelsConfig {
	return ModelsConfig{
		ModelsTTL:    1 * time.Hour,
		EmbedTTL:     24 * time.Hour,
		MaxEmbeddings: 10000,
	}
}

// NewModelsCache creates a new models cache
func NewModelsCache(cfg ModelsConfig) *ModelsCache {
	if cfg.ModelsTTL <= 0 {
		cfg.ModelsTTL = 1 * time.Hour
	}
	if cfg.EmbedTTL <= 0 {
		cfg.EmbedTTL = 24 * time.Hour
	}
	if cfg.MaxEmbeddings <= 0 {
		cfg.MaxEmbeddings = 10000
	}

	return &ModelsCache{
		cache: New(Config{
			MaxItems: 100,
			TTL:      cfg.ModelsTTL,
		}),
		modelsTTL: cfg.ModelsTTL,
		embedTTL:  cfg.EmbedTTL,
		embeddings: New(Config{
			MaxItems: cfg.MaxEmbeddings,
			TTL:      cfg.EmbedTTL,
		}),
	}
}

// GetModels retrieves the cached model list
func (c *ModelsCache) GetModels() (interface{}, bool) {
	return c.cache.Get("models:list")
}

// SetModels caches the model list
func (c *ModelsCache) SetModels(models interface{}) {
	c.cache.SetWithTTL("models:list", models, c.modelsTTL)
}

// InvalidateModels invalidates the model list cache
func (c *ModelsCache) InvalidateModels() {
	c.cache.Delete("models:list")
}

// GetModel retrieves a specific cached model
func (c *ModelsCache) GetModel(name string) (interface{}, bool) {
	return c.cache.Get("model:" + name)
}

// SetModel caches a specific model
func (c *ModelsCache) SetModel(name string, model interface{}) {
	c.cache.SetWithTTL("model:"+name, model, c.modelsTTL)
}

// EmbeddingKey generates a cache key for an embedding
func EmbeddingKey(model, text string) string {
	hash := sha256.Sum256([]byte(model + "|" + text))
	return "embed:" + hex.EncodeToString(hash[:16]) // Use first 16 bytes
}

// GetEmbedding retrieves a cached embedding
func (c *ModelsCache) GetEmbedding(model, text string) ([]float64, bool) {
	key := EmbeddingKey(model, text)
	if val, ok := c.embeddings.Get(key); ok {
		if embed, ok := val.([]float64); ok {
			return embed, true
		}
	}
	return nil, false
}

// SetEmbedding caches an embedding
func (c *ModelsCache) SetEmbedding(model, text string, embedding []float64) {
	key := EmbeddingKey(model, text)
	c.embeddings.SetWithTTL(key, embedding, c.embedTTL)
}

// GetBatchEmbeddings retrieves cached embeddings for multiple texts
// Returns found embeddings and indices of missing ones
func (c *ModelsCache) GetBatchEmbeddings(model string, texts []string) (map[int][]float64, []int) {
	found := make(map[int][]float64)
	var missing []int

	for i, text := range texts {
		if embed, ok := c.GetEmbedding(model, text); ok {
			found[i] = embed
		} else {
			missing = append(missing, i)
		}
	}

	return found, missing
}

// SetBatchEmbeddings caches multiple embeddings
func (c *ModelsCache) SetBatchEmbeddings(model string, texts []string, embeddings [][]float64) {
	for i, text := range texts {
		if i < len(embeddings) {
			c.SetEmbedding(model, text, embeddings[i])
		}
	}
}

// Stats returns cache statistics
func (c *ModelsCache) Stats() map[string]interface{} {
	modelsHits, modelsMisses, modelsRate := c.cache.Stats()
	embedHits, embedMisses, embedRate := c.embeddings.Stats()

	return map[string]interface{}{
		"models_cache_size":    c.cache.Size(),
		"models_hits":          modelsHits,
		"models_misses":        modelsMisses,
		"models_hit_rate":      modelsRate,
		"embeddings_cache_size": c.embeddings.Size(),
		"embeddings_hits":      embedHits,
		"embeddings_misses":    embedMisses,
		"embeddings_hit_rate":  embedRate,
	}
}

// Clear clears all caches
func (c *ModelsCache) Clear() {
	c.cache.Clear()
	c.embeddings.Clear()
}

// Global models cache singleton
var (
	globalModelsCache     *ModelsCache
	globalModelsCacheOnce sync.Once
)

// GetModelsCache returns the global models cache
func GetModelsCache() *ModelsCache {
	globalModelsCacheOnce.Do(func() {
		globalModelsCache = NewModelsCache(DefaultModelsConfig())
	})
	return globalModelsCache
}
