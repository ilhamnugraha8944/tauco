package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Counter interface {
	Take(context.Context, string, int, time.Duration) (bool, time.Duration, error)
}

type Limiter struct {
	primary  Counter
	fallback *Local
	observe  func(string)
}

func New(primary Counter, fallback *Local, observer ...func(string)) (*Limiter, error) {
	if fallback == nil {
		return nil, errors.New("rate limiter requires a local fallback")
	}
	var observe func(string)
	if len(observer) > 0 {
		observe = observer[0]
	}
	return &Limiter{primary: primary, fallback: fallback, observe: observe}, nil
}

func (limiter *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration) {
	if limiter.primary != nil {
		allowed, retry, err := limiter.primary.Take(ctx, key, limit, window)
		if err == nil {
			limiter.record(map[bool]string{true: "allowed", false: "blocked"}[allowed])
			return allowed, retry
		}
	}
	allowed, retry, _ := limiter.fallback.Take(ctx, key, limit, window)
	limiter.record("fallback")
	if !allowed {
		limiter.record("blocked")
	}
	return allowed, retry
}

func (limiter *Limiter) record(outcome string) {
	if limiter.observe != nil {
		limiter.observe(outcome)
	}
}

type localEntry struct {
	count   int
	expires time.Time
}

type Local struct {
	mu      sync.Mutex
	entries map[string]localEntry
	now     func() time.Time
	maxKeys int
}

func NewLocal(maxKeys int) (*Local, error) {
	if maxKeys < 1 {
		return nil, errors.New("local rate limit capacity must be positive")
	}
	return &Local{entries: make(map[string]localEntry), now: time.Now, maxKeys: maxKeys}, nil
}

func (local *Local) Take(_ context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if key == "" || limit < 1 || window <= 0 {
		return false, 0, errors.New("invalid rate-limit request")
	}
	local.mu.Lock()
	defer local.mu.Unlock()
	now := local.now()
	entry, found := local.entries[key]
	if !found || !entry.expires.After(now) {
		if len(local.entries) >= local.maxKeys {
			for existingKey, existing := range local.entries {
				if !existing.expires.After(now) {
					delete(local.entries, existingKey)
				}
			}
		}
		if len(local.entries) >= local.maxKeys {
			// Memory safety wins over an unbounded attacker-controlled key map.
			return true, 0, nil
		}
		entry = localEntry{expires: now.Add(window)}
	}
	entry.count++
	local.entries[key] = entry
	return entry.count <= limit, entry.expires.Sub(now), nil
}
