package repository

import (
	"context"
	"fmt"
	"time"

	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	platformcache "github.com/ilhamnugraha8944/tauco/backend/internal/platform/cache"
	"golang.org/x/sync/singleflight"
)

type Cached struct {
	origin contentapp.PublishedPageRepository
	store  platformcache.Store
	ttl    time.Duration
	group  singleflight.Group
}

func NewCached(origin contentapp.PublishedPageRepository, store platformcache.Store, ttl time.Duration) *Cached {
	return &Cached{origin: origin, store: store, ttl: ttl}
}

func (repository *Cached) FindPublishedPage(ctx context.Context, key domain.PageKey) (domain.PublishedPage, error) {
	if repository == nil || repository.origin == nil {
		return domain.PublishedPage{}, fmt.Errorf("cached page repository is not initialized")
	}
	if repository.store == nil || repository.ttl <= 0 {
		return repository.origin.FindPublishedPage(ctx, key)
	}
	tag := string(key)
	generation, err := repository.store.Generation(ctx, tag)
	if err != nil {
		return repository.origin.FindPublishedPage(ctx, key)
	}
	cacheKey := fmt.Sprintf("v1:page:%s:g%d", key, generation)
	return platformcache.Aside(ctx, repository.store, &repository.group, cacheKey, repository.ttl,
		func(page domain.PublishedPage) error { return page.Validate() },
		func(loadContext context.Context) (domain.PublishedPage, error) {
			return repository.origin.FindPublishedPage(loadContext, key)
		},
	)
}
