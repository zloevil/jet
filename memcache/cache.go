package memcache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

const (
	defaultExpiration    = time.Hour
	defaultCleanupPeriod = time.Minute * 5

	DefaultTtl = 0
	Forever    = -1
)

// MemCache provides in simple memory cache with ttl
type MemCache interface {
	// Get retrieves item by key
	Get(key string) (interface{}, bool)
	// Set sets item with key and ttl
	// If the duration is 0, the cache's default expiration time is used.
	// If it is -1, the item never expires.
	Set(key string, v interface{}, ttl time.Duration)
	// Delete deletes key
	Delete(key string)
}

func NewMemCache() MemCache {
	return &cacheImpl{cache: cache.New(defaultExpiration, defaultCleanupPeriod)}
}

type cacheImpl struct {
	cache *cache.Cache
}

func (c *cacheImpl) Delete(key string) {
	c.cache.Delete(key)
}

func (c *cacheImpl) Set(key string, v interface{}, ttl time.Duration) {
	c.cache.Set(key, v, ttl)
}

func (c *cacheImpl) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}
