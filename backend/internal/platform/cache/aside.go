package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"golang.org/x/sync/singleflight"
)

// Aside is fail-open: every cache error falls back to the source of truth.
func Aside[T any](
	ctx context.Context,
	store Store,
	group *singleflight.Group,
	key string,
	ttl time.Duration,
	validate func(T) error,
	load func(context.Context) (T, error),
) (T, error) {
	var zero T
	if store == nil || group == nil {
		return load(ctx)
	}
	if raw, err := store.Get(ctx, key); err == nil {
		var cached T
		if json.Unmarshal(raw, &cached) == nil && (validate == nil || validate(cached) == nil) {
			return cached, nil
		}
		_ = store.Delete(ctx, key)
	}
	value, err, _ := group.Do(key, func() (any, error) {
		if raw, getErr := store.Get(ctx, key); getErr == nil {
			var cached T
			if json.Unmarshal(raw, &cached) == nil && (validate == nil || validate(cached) == nil) {
				return cached, nil
			}
		}
		loaded, loadErr := load(ctx)
		if loadErr != nil {
			return zero, loadErr
		}
		raw, marshalErr := json.Marshal(loaded)
		if marshalErr == nil {
			_ = store.Set(ctx, key, raw, Jitter(ttl))
		}
		return loaded, nil
	})
	if err != nil {
		return zero, err
	}
	result, ok := value.(T)
	if !ok {
		return zero, errors.New("cache coalescing type mismatch")
	}
	return result, nil
}
