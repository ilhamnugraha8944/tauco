package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
)

var ErrInvalidCacheInvalidation = errors.New("invalid cache invalidation payload")

type GenerationStore interface {
	Bump(context.Context, string) error
}

type CacheInvalidationHandler struct{ store GenerationStore }

func NewCacheInvalidationHandler(store GenerationStore) (*CacheInvalidationHandler, error) {
	if store == nil {
		return nil, errors.New("cache invalidation handler requires store")
	}
	return &CacheInvalidationHandler{store: store}, nil
}

var generationTagPattern = regexp.MustCompile(`^(home|about|products|product:[a-z0-9]+(?:-[a-z0-9]+)*)$`)

func (handler *CacheInvalidationHandler) Handle(ctx context.Context, payload json.RawMessage) error {
	var value struct {
		GenerationTag  string   `json:"generationTag"`
		GenerationTags []string `json:"generationTags"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return ErrInvalidCacheInvalidation
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrInvalidCacheInvalidation
	}
	tags := append([]string(nil), value.GenerationTags...)
	if value.GenerationTag != "" {
		tags = append(tags, value.GenerationTag)
	}
	if len(tags) < 1 || len(tags) > 3 {
		return ErrInvalidCacheInvalidation
	}
	sort.Strings(tags)
	for index, tag := range tags {
		if !generationTagPattern.MatchString(tag) || (index > 0 && tags[index-1] == tag) {
			return ErrInvalidCacheInvalidation
		}
	}
	for _, tag := range tags {
		if err := handler.store.Bump(ctx, tag); err != nil {
			return err
		}
	}
	return nil
}
