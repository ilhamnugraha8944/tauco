package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
	platformcache "github.com/ilhamnugraha8944/tauco/backend/internal/platform/cache"
	"golang.org/x/sync/singleflight"
)

type Cached struct {
	origin catalogapp.PublishedProductRepository
	store  platformcache.Store
	ttl    time.Duration
	group  singleflight.Group
}

func NewCached(origin catalogapp.PublishedProductRepository, store platformcache.Store, ttl time.Duration) *Cached {
	return &Cached{origin: origin, store: store, ttl: ttl}
}

func (repository *Cached) FindPublishedProduct(ctx context.Context, slug string) (domain.PublishedProduct, error) {
	if repository == nil || repository.origin == nil {
		return domain.PublishedProduct{}, errors.New("cached catalog repository is not initialized")
	}
	productsGeneration, productGeneration, ok := repository.generations(ctx, slug)
	if !ok {
		return repository.origin.FindPublishedProduct(ctx, slug)
	}
	key := fmt.Sprintf("v1:product:%s:g%d:%d", slug, productsGeneration, productGeneration)
	return platformcache.Aside(ctx, repository.store, &repository.group, key, repository.ttl,
		func(product domain.PublishedProduct) error { return product.Validate() },
		func(loadContext context.Context) (domain.PublishedProduct, error) {
			return repository.origin.FindPublishedProduct(loadContext, slug)
		},
	)
}

func (repository *Cached) ListPublishedProducts(ctx context.Context, after *domain.PaginationPosition, limit int) (catalogapp.PublishedProductPage, error) {
	if repository == nil || repository.origin == nil {
		return catalogapp.PublishedProductPage{}, errors.New("cached catalog repository is not initialized")
	}
	if repository.store == nil || repository.ttl <= 0 {
		return repository.origin.ListPublishedProducts(ctx, after, limit)
	}
	generation, err := repository.store.Generation(ctx, "products")
	if err != nil {
		return repository.origin.ListPublishedProducts(ctx, after, limit)
	}
	position := "start"
	if after != nil {
		position = fmt.Sprintf("%d:%s", after.SortOrder, after.ProductID)
	}
	key := fmt.Sprintf("v1:products:g%d:after:%s:limit:%d", generation, position, limit)
	return platformcache.Aside(ctx, repository.store, &repository.group, key, repository.ttl,
		func(page catalogapp.PublishedProductPage) error {
			if len(page.Products) > limit {
				return errors.New("cached product page exceeds requested limit")
			}
			for _, product := range page.Products {
				if err := product.Validate(); err != nil {
					return err
				}
			}
			return nil
		},
		func(loadContext context.Context) (catalogapp.PublishedProductPage, error) {
			return repository.origin.ListPublishedProducts(loadContext, after, limit)
		},
	)
}

func (repository *Cached) generations(ctx context.Context, slug string) (int64, int64, bool) {
	if repository == nil || repository.origin == nil || repository.store == nil || repository.ttl <= 0 {
		return 0, 0, false
	}
	products, err := repository.store.Generation(ctx, "products")
	if err != nil {
		return 0, 0, false
	}
	product, err := repository.store.Generation(ctx, "product:"+slug)
	if err != nil {
		return 0, 0, false
	}
	return products, product, true
}
