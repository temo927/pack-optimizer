package redisad

import (
	"context"
	"strings"
	"time"

	gredis "github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *gredis.Client
}

func New(rdb *gredis.Client) *Cache { return &Cache{rdb: rdb} }

func (c *Cache) Get(key string) ([]byte, error) {
	s, err := c.rdb.Get(context.Background(), key).Bytes()
	if err == gredis.Nil {
		return nil, nil
	}
	return s, err
}

func (c *Cache) Set(key string, value []byte, ttlSeconds int) error {
	return c.rdb.Set(context.Background(), key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func (c *Cache) DeleteByPrefix(prefix string) error {
	iter := c.rdb.Scan(context.Background(), 0, prefix+"*", 0).Iterator()
	var firstErr error
	for iter.Next(context.Background()) {
		if err := c.rdb.Del(context.Background(), iter.Val()).Err(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := iter.Err(); err != nil && !strings.Contains(err.Error(), "EOF") {
		return err
	}
	return firstErr
}


