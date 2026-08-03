package cache

import (
	"context"
	"errors"
	"time"
)

type Observer interface {
	ObserveCache(string)
}

// ObservedStore decorates cache commands with bounded outcome counters.
type ObservedStore struct {
	Store
	observer Observer
}

func Observe(store Store, observer Observer) Store {
	if store == nil || observer == nil {
		return store
	}
	return &ObservedStore{Store: store, observer: observer}
}

func (store *ObservedStore) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := store.Store.Get(ctx, key)
	switch {
	case err == nil:
		store.observer.ObserveCache("hit")
	case errors.Is(err, ErrMiss):
		store.observer.ObserveCache("miss")
	default:
		store.observer.ObserveCache("error")
	}
	return value, err
}

func (store *ObservedStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := store.Store.Set(ctx, key, value, ttl)
	if err == nil {
		store.observer.ObserveCache("write")
	} else {
		store.observer.ObserveCache("error")
	}
	return err
}
