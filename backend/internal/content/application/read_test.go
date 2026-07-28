package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

type pageRepositoryStub struct {
	page domain.PublishedPage
	err  error
	key  domain.PageKey
}

func (stub *pageRepositoryStub) FindPublishedPage(
	_ context.Context,
	key domain.PageKey,
) (domain.PublishedPage, error) {
	stub.key = key
	return stub.page, stub.err
}

func TestPublishedReaderDelegatesAndPreservesRepositoryErrors(t *testing.T) {
	t.Parallel()

	expected := domain.PublishedPage{Key: domain.PageKeyHome}
	repository := &pageRepositoryStub{page: expected}
	reader, err := NewPublishedReader(repository)
	if err != nil {
		t.Fatalf("NewPublishedReader() error = %v", err)
	}
	page, err := reader.Get(context.Background(), domain.PageKeyHome)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if page.Key != expected.Key || repository.key != domain.PageKeyHome {
		t.Fatalf("Get() page/key = %q/%q", page.Key, repository.key)
	}

	repository.err = ErrPublishedPageNotFound
	if _, err := reader.Get(
		context.Background(),
		domain.PageKeyAbout,
	); !errors.Is(err, ErrPublishedPageNotFound) {
		t.Fatalf("Get() error = %v, want ErrPublishedPageNotFound", err)
	}
}

func TestPublishedReaderRejectsMissingDependencyAndInvalidKey(t *testing.T) {
	t.Parallel()

	if _, err := NewPublishedReader(nil); err == nil {
		t.Fatal("NewPublishedReader(nil) error = nil")
	}
	reader, err := NewPublishedReader(&pageRepositoryStub{})
	if err != nil {
		t.Fatalf("NewPublishedReader() error = %v", err)
	}
	if _, err := reader.Get(
		context.Background(),
		domain.PageKey("unknown"),
	); !errors.Is(err, ErrPublishedPageNotFound) {
		t.Fatalf("Get(invalid key) error = %v", err)
	}
}
