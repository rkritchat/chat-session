package cache

import (
	"chat-session/internal/config"
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type Cache interface {
	Set(key, val string, ttl ...time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
	Pub(channel, msg string) *redis.IntCmd
	Sub(channel string) *redis.PubSub
}

type cache struct {
	rdb *redis.Client
	env config.Env
}

func NewCache(rdb *redis.Client, env config.Env) Cache {
	return &cache{
		rdb: rdb,
		env: env,
	}
}

func (c cache) Set(key, val string, ttl ...time.Duration) error {
	exp := time.Duration(c.env.RedisTTL) * time.Millisecond
	if len(ttl) > 0 {
		exp = ttl[0]
	}
	_, err := c.rdb.SetEX(context.Background(), key, val, exp).Result()
	return err
}

func (c cache) Get(key string) (string, error) {
	return c.rdb.Get(context.Background(), key).Result()
}

func (c cache) Del(key string) error {
	_, err := c.rdb.Del(context.Background(), key).Result()
	return err
}

func (c cache) Pub(channel, msg string) *redis.IntCmd {
	return c.rdb.Publish(context.Background(), channel, msg)
}

func (c cache) Sub(channel string) *redis.PubSub {
	return c.rdb.Subscribe(context.Background(), channel)
}
