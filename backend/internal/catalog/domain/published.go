package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

// PublishedProduct is the immutable revision selected by a product's current
// published pointer.
type PublishedProduct struct {
	ProductID      contentdomain.UUIDv7
	RevisionID     contentdomain.UUIDv7
	Slug           string
	SortOrder      int
	RevisionNumber uint32
	SchemaVersion  uint32
	ContentJSON    json.RawMessage
	Checksum       contentdomain.SHA256Checksum
	PublishedAt    time.Time
}

// Validate checks repository hydration invariants without importing GORM.
func (product PublishedProduct) Validate() error {
	if _, err := contentdomain.ParseUUIDv7(string(product.ProductID)); err != nil {
		return fmt.Errorf("invalid product ID: %w", err)
	}
	if _, err := contentdomain.ParseUUIDv7(string(product.RevisionID)); err != nil {
		return fmt.Errorf("invalid product revision ID: %w", err)
	}
	if product.ProductID == product.RevisionID {
		return errors.New("product and revision IDs must differ")
	}
	if err := contentdomain.ValidateProductSlug(product.Slug); err != nil {
		return err
	}
	if product.SortOrder < 0 {
		return errors.New("product sort order must not be negative")
	}
	if product.RevisionNumber == 0 || product.SchemaVersion == 0 {
		return errors.New("product revision and schema versions must be positive")
	}
	if err := contentdomain.ValidateCanonicalJSON(
		product.ContentJSON,
		product.Checksum,
	); err != nil {
		return fmt.Errorf("invalid published product content: %w", err)
	}
	if product.PublishedAt.IsZero() {
		return errors.New("published product must have a publication timestamp")
	}
	return nil
}

// PaginationPosition is the repository-neutral keyset boundary for the stable
// `(sort_order ASC, id ASC)` catalog order.
type PaginationPosition struct {
	SortOrder int
	ProductID contentdomain.UUIDv7
}
