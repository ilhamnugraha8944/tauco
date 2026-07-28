package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

type productRepositoryStub struct {
	product domain.PublishedProduct
	page    PublishedProductPage
	after   *domain.PaginationPosition
	limit   int
	err     error
}

func (stub *productRepositoryStub) FindPublishedProduct(
	context.Context,
	string,
) (domain.PublishedProduct, error) {
	return stub.product, stub.err
}

func (stub *productRepositoryStub) ListPublishedProducts(
	_ context.Context,
	after *domain.PaginationPosition,
	limit int,
) (PublishedProductPage, error) {
	stub.after = after
	stub.limit = limit
	return stub.page, stub.err
}

type cursorCodecStub struct {
	decoded ProductPaginationPosition
	encoded string
	err     error
}

func (stub *cursorCodecStub) Encode(
	ProductPaginationPosition,
	ProductQueryHash,
) (string, error) {
	return stub.encoded, stub.err
}

func (stub *cursorCodecStub) Decode(
	string,
	ProductQueryHash,
) (ProductPaginationPosition, error) {
	return stub.decoded, stub.err
}

func TestPublishedReaderListsWithAuthenticatedCursor(t *testing.T) {
	t.Parallel()

	productID := contentdomain.UUIDv7(
		"019c0a8d-f070-7a9a-b9f4-d39e50ad7370",
	)
	decoded, err := NewProductPaginationPosition(3, string(productID))
	if err != nil {
		t.Fatalf("NewProductPaginationPosition() error = %v", err)
	}
	repository := &productRepositoryStub{
		page: PublishedProductPage{
			Products: []domain.PublishedProduct{{
				ProductID: productID,
				SortOrder: 4,
			}},
			HasMore: true,
		},
	}
	codec := &cursorCodecStub{decoded: decoded, encoded: "next.cursor"}
	reader, err := NewPublishedReader(repository, codec)
	if err != nil {
		t.Fatalf("NewPublishedReader() error = %v", err)
	}
	cursor := "current.cursor"
	limit := 1
	result, err := reader.List(context.Background(), &cursor, &limit)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repository.after == nil ||
		repository.after.SortOrder != 3 ||
		repository.limit != 1 {
		t.Fatalf(
			"repository after/limit = %+v/%d",
			repository.after,
			repository.limit,
		)
	}
	if !result.HasMore ||
		result.NextCursor == nil ||
		*result.NextCursor != "next.cursor" {
		t.Fatalf("List() result = %+v", result)
	}
}

func TestPublishedReaderRejectsInvalidCursorAndLimit(t *testing.T) {
	t.Parallel()

	repository := &productRepositoryStub{}
	codec := &cursorCodecStub{err: ErrInvalidCursor}
	reader, err := NewPublishedReader(repository, codec)
	if err != nil {
		t.Fatalf("NewPublishedReader() error = %v", err)
	}
	cursor := "invalid"
	if _, err := reader.List(
		context.Background(),
		&cursor,
		nil,
	); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("List(invalid cursor) error = %v", err)
	}

	codec.err = nil
	invalidLimit := 0
	if _, err := reader.List(
		context.Background(),
		nil,
		&invalidLimit,
	); !errors.Is(err, ErrInvalidPageLimit) {
		t.Fatalf("List(invalid limit) error = %v", err)
	}
}
