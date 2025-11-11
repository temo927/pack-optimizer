// Package redisad implements the Redis adapter for caching operations.
// This adapter implements the domain.Cache port, providing a Redis-based caching layer.
package redisad

import (
	"context"
	"strings"
	"time"

	gredis "github.com/redis/go-redis/v9"
)

// Cache implements the domain.Cache interface using Redis.
type Cache struct {
	rdb *gredis.Client // Redis client connection
}

// New creates a new Redis cache adapter.
func New(rdb *gredis.Client) *Cache { return &Cache{rdb: rdb} }

// Get retrieves a value from Redis by key.
// Returns nil if the key doesn't exist (Redis.Nil error is converted to nil).
// Returns the error for other failures (connection issues, etc.).
func (c *Cache) Get(key string) ([]byte, error) {
	s, err := c.rdb.Get(context.Background(), key).Bytes()
	if err == gredis.Nil {
		// Key doesn't exist - return nil instead of error
		return nil, nil
	}
	return s, err
}

// Set stores a value in Redis with a time-to-live (TTL).
// The TTL is specified in seconds and determines how long the value will be cached.
func (c *Cache) Set(key string, value []byte, ttlSeconds int) error {
	return c.rdb.Set(context.Background(), key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

// DeleteByPrefix removes all keys matching the given prefix.
// Uses Redis SCAN to iterate through keys matching the pattern, then deletes them.
// This is used for cache invalidation when data changes (e.g., pack sizes updated).
// Returns the first error encountered, if any.
func (c *Cache) DeleteByPrefix(prefix string) error {
	// Use SCAN to find all keys matching the prefix pattern
	iter := c.rdb.Scan(context.Background(), 0, prefix+"*", 0).Iterator()
	var firstErr error
	
	// Delete each matching key
	for iter.Next(context.Background()) {
		if err := c.rdb.Del(context.Background(), iter.Val()).Err(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	
	// Check for iterator errors (ignore EOF which indicates end of scan)
	if err := iter.Err(); err != nil && !strings.Contains(err.Error(), "EOF") {
		return err
	}
	
	return firstErr
}
