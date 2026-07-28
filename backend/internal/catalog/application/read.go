package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

const publishedCatalogQuery = "published=true"

var errInvalidPublishedProductPage = errors.New(
	"invalid published product page",
)

// PublishedProductList is one cursor-paginated public catalog result.
type PublishedProductList struct {
	Products   []domain.PublishedProduct
	Limit      int
	HasMore    bool
	NextCursor *string
}

// PublishedReader is the public catalog read use case.
type PublishedReader struct {
	repository PublishedProductRepository
	cursors    CursorCodec
	queryHash  ProductQueryHash
}

// NewPublishedReader creates a published-only catalog reader.
func NewPublishedReader(
	repository PublishedProductRepository,
	cursors CursorCodec,
) (*PublishedReader, error) {
	if repository == nil {
		return nil, errors.New("published product reader requires a repository")
	}
	if cursors == nil {
		return nil, errors.New("published product reader requires a cursor codec")
	}
	return &PublishedReader{
		repository: repository,
		cursors:    cursors,
		queryHash:  NewProductQueryHash(publishedCatalogQuery),
	}, nil
}

// Get returns one immutable published product.
func (reader *PublishedReader) Get(
	ctx context.Context,
	slug string,
) (domain.PublishedProduct, error) {
	if reader == nil || reader.repository == nil {
		return domain.PublishedProduct{}, errors.New(
			"published product reader is not initialized",
		)
	}
	product, err := reader.repository.FindPublishedProduct(ctx, slug)
	if err != nil {
		return domain.PublishedProduct{}, fmt.Errorf(
			"read published product: %w",
			err,
		)
	}
	return product, nil
}

// List authenticates the cursor, reads one bounded page, and creates the next
// cursor from the final returned product when needed.
func (reader *PublishedReader) List(
	ctx context.Context,
	encodedCursor *string,
	requestedLimit *int,
) (PublishedProductList, error) {
	if reader == nil || reader.repository == nil || reader.cursors == nil {
		return PublishedProductList{}, errors.New(
			"published product reader is not initialized",
		)
	}

	limit, err := ResolvePageLimit(requestedLimit)
	if err != nil {
		return PublishedProductList{}, err
	}

	var after *domain.PaginationPosition
	if encodedCursor != nil {
		position, decodeErr := reader.cursors.Decode(
			*encodedCursor,
			reader.queryHash,
		)
		if decodeErr != nil {
			return PublishedProductList{}, ErrInvalidCursor
		}
		productID, parseErr := contentdomain.ParseUUIDv7(position.ProductID())
		if parseErr != nil {
			return PublishedProductList{}, ErrInvalidCursor
		}
		after = &domain.PaginationPosition{
			SortOrder: int(position.SortOrder()),
			ProductID: productID,
		}
	}

	page, err := reader.repository.ListPublishedProducts(ctx, after, limit)
	if err != nil {
		return PublishedProductList{}, fmt.Errorf(
			"list published products: %w",
			err,
		)
	}

	result := PublishedProductList{
		Products: append([]domain.PublishedProduct(nil), page.Products...),
		Limit:    limit,
		HasMore:  page.HasMore,
	}
	if !page.HasMore {
		return result, nil
	}
	if len(page.Products) == 0 {
		return PublishedProductList{}, errInvalidPublishedProductPage
	}

	last := page.Products[len(page.Products)-1]
	position, err := NewProductPaginationPosition(
		int64(last.SortOrder),
		string(last.ProductID),
	)
	if err != nil {
		return PublishedProductList{}, errInvalidPublishedProductPage
	}
	nextCursor, err := reader.cursors.Encode(position, reader.queryHash)
	if err != nil {
		return PublishedProductList{}, errInvalidPublishedProductPage
	}
	result.NextCursor = &nextCursor
	return result, nil
}
