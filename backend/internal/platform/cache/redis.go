package cache

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrMiss = errors.New("cache miss")

type Store interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	Delete(context.Context, string) error
	Generation(context.Context, string) (int64, error)
	Bump(context.Context, string) error
}

type Redis struct {
	client *redis.Client
}

var rateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {current, ttl}
`)

func NewRedis(rawURL string) (*Redis, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, errors.New("invalid Redis URL")
	}
	options.Protocol = 2
	options.DialTimeout = time.Second
	options.ReadTimeout = time.Second
	options.WriteTimeout = time.Second
	return &Redis{client: redis.NewClient(options)}, nil
}

func (store *Redis) Ping(ctx context.Context) error {
	return store.client.Ping(ctx).Err()
}

func (store *Redis) Close() error {
	if store == nil || store.client == nil {
		return nil
	}
	return store.client.Close()
}

func (store *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := store.client.Get(ctx, dataKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrMiss
	}
	return value, err
}

func (store *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return store.client.Set(ctx, dataKey(key), value, ttl).Err()
}

func (store *Redis) Delete(ctx context.Context, key string) error {
	return store.client.Del(ctx, dataKey(key)).Err()
}

func (store *Redis) Generation(ctx context.Context, tag string) (int64, error) {
	value, err := store.client.Get(ctx, generationKey(tag)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	generation, err := strconv.ParseInt(value, 10, 64)
	if err != nil || generation < 0 {
		return 0, errors.New("invalid cache generation")
	}
	return generation, nil
}

func (store *Redis) Bump(ctx context.Context, tag string) error {
	return store.client.Incr(ctx, generationKey(tag)).Err()
}

func (store *Redis) Take(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	values, err := rateLimitScript.Run(
		ctx, store.client, []string{"tauco:rate:" + key}, window.Milliseconds(),
	).Int64Slice()
	if err != nil || len(values) != 2 {
		if err == nil {
			err = errors.New("invalid Redis rate-limit response")
		}
		return false, 0, err
	}
	retry := time.Duration(values[1]) * time.Millisecond
	if retry < 0 {
		retry = window
	}
	return values[0] <= int64(limit), retry, nil
}

func Jitter(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	// Frozen B8 policy: +/-10 percent, always positive.
	offset := rand.Int64N(int64(base)/5+1) - int64(base)/10
	return base + time.Duration(offset)
}

func Invalidate(ctx context.Context, store Store, tags ...string) error {
	if store == nil {
		return errors.New("cache store is required")
	}
	var failures []error
	for _, tag := range tags {
		if tag == "" {
			failures = append(failures, errors.New("cache tag is empty"))
			continue
		}
		if err := store.Bump(ctx, tag); err != nil {
			failures = append(failures, fmt.Errorf("bump cache tag: %w", err))
		}
	}
	return errors.Join(failures...)
}

func dataKey(key string) string       { return "tauco:cache:" + key }
func generationKey(tag string) string { return "tauco:generation:" + tag }
