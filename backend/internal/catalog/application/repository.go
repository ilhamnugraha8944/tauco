package application

import (
	"context"
	"errors"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
)

// ErrPublishedProductNotFound is stable across persistence adapters.
var ErrPublishedProductNotFound = errors.New("published product not found")

// PublishedProductPage contains one bounded keyset page and whether another
// item exists after its final element. The repository probes one extra row so
// B4 can create a truthful next cursor without a count query.
type PublishedProductPage struct {
	Products []domain.PublishedProduct
	HasMore  bool
}

// PublishedProductRepository is the B4 catalog read port implemented by the
// B3 PostgreSQL adapter.
type PublishedProductRepository interface {
	FindPublishedProduct(
		context.Context,
		string,
	) (domain.PublishedProduct, error)
	ListPublishedProducts(
		context.Context,
		*domain.PaginationPosition,
		int,
	) (PublishedProductPage, error)
}
