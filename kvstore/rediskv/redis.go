package rediskv

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// RedisStore represents a Redis key-value store with a prefix.
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewStore initializes a new RedisStore with the given Redis options and prefix.
func NewStore(options *redis.Options, prefix string) *RedisStore {
	client := redis.NewClient(options)
	return &RedisStore{client: client, prefix: prefix}
}

// buildKey constructs a key with the given prefix.
func (rs *RedisStore) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", rs.prefix, key)
}

// Has checks if the key exists in the Redis store.
func (rs *RedisStore) Has(ctx context.Context, key string) (bool, error) {
	val, err := rs.client.Exists(ctx, rs.buildKey(key)).Result()
	return val > 0, err
}

// Get retrieves the value associated with the key from the Redis store.
func (rs *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := rs.client.Get(ctx, rs.buildKey(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

// Set stores the key-value pair in the Redis store.
func (rs *RedisStore) Set(ctx context.Context, key string, val []byte) error {
	return rs.client.Set(ctx, rs.buildKey(key), val, 0).Err()
}

// Del deletes the key from the Redis store.
func (rs *RedisStore) Del(ctx context.Context, key string) (bool, error) {
	val, err := rs.client.Del(ctx, rs.buildKey(key)).Result()
	return val > 0, err
}

// ForEach iterates over all keys with the given prefix in the Redis store and executes the provided function.
func (rs *RedisStore) ForEach(ctx context.Context, fn func(ctx context.Context, key string, value []byte) bool) error {
	pattern := fmt.Sprintf("%s:*", rs.prefix)
	iter := rs.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		fullKey := iter.Val()

		// Strip prefix from fullKey
		key := fullKey[len(rs.prefix)+1:]

		value, ok, err := rs.Get(ctx, key)
		if err != nil {
			return err
		}
		if ok {
			if !fn(ctx, key, value) {
				break
			}
		}
	}
	return iter.Err()
}
