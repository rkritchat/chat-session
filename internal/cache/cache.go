package cache

import (
	"chat-session/internal/config"
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type Cache interface {
	Set(key, val string) error
	Get(key string) (string, error)
	Del(key string) error
}

type cache struct {
	rdb *redis.Client
	env config.Env
}

func NewCache(rdb *redis.Client) Cache {
	return &cache{
		rdb: rdb,
	}
}

func (c cache) Set(key, val string) error {
	ttl := time.Duration(c.env.RedisTTL) * time.Millisecond
	_, err := c.rdb.SetEX(context.Background(), key, val, ttl).Result()
	return err
}

func (c cache) Get(key string) (string, error) {
	return c.rdb.Get(context.Background(), key).Result()
}

func (c cache) Del(key string) error {
	_, err := c.rdb.Del(context.Background(), key).Result()
	return err
}
