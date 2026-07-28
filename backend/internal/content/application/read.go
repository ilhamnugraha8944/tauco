package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

// ErrPublishedPageNotFound is returned without leaking persistence details.
var ErrPublishedPageNotFound = errors.New("published page not found")

// PublishedPageRepository is the read port consumed by B4 public content use
// cases. Its implementation belongs to the repository layer.
type PublishedPageRepository interface {
	FindPublishedPage(
		context.Context,
		domain.PageKey,
	) (domain.PublishedPage, error)
}

// PublishedReader is the public content read use case.
type PublishedReader struct {
	repository PublishedPageRepository
}

// NewPublishedReader creates a published-only page reader.
func NewPublishedReader(
	repository PublishedPageRepository,
) (*PublishedReader, error) {
	if repository == nil {
		return nil, errors.New("published page reader requires a repository")
	}
	return &PublishedReader{repository: repository}, nil
}

// Get returns one immutable published page.
func (reader *PublishedReader) Get(
	ctx context.Context,
	key domain.PageKey,
) (domain.PublishedPage, error) {
	if reader == nil || reader.repository == nil {
		return domain.PublishedPage{}, errors.New(
			"published page reader is not initialized",
		)
	}
	if !key.Valid() {
		return domain.PublishedPage{}, ErrPublishedPageNotFound
	}
	page, err := reader.repository.FindPublishedPage(ctx, key)
	if err != nil {
		return domain.PublishedPage{}, fmt.Errorf(
			"read published page %q: %w",
			key,
			err,
		)
	}
	return page, nil
}
