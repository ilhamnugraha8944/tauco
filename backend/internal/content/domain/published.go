package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// PublishedPage is the immutable page revision selected by a page's published
// pointer. ContentJSON remains transport-neutral so the B4 application layer
// can validate and project it through generated API types.
type PublishedPage struct {
	PageID         UUIDv7
	RevisionID     UUIDv7
	Key            PageKey
	RevisionNumber uint32
	SchemaVersion  uint32
	ContentJSON    json.RawMessage
	Checksum       SHA256Checksum
	PublishedAt    time.Time
}

// Validate checks the invariants a repository must preserve when hydrating a
// published page.
func (page PublishedPage) Validate() error {
	if _, err := ParseUUIDv7(string(page.PageID)); err != nil {
		return fmt.Errorf("invalid page ID: %w", err)
	}
	if _, err := ParseUUIDv7(string(page.RevisionID)); err != nil {
		return fmt.Errorf("invalid page revision ID: %w", err)
	}
	if page.PageID == page.RevisionID {
		return errors.New("page and revision IDs must differ")
	}
	if !page.Key.Valid() {
		return fmt.Errorf("invalid page key %q", page.Key)
	}
	if page.RevisionNumber == 0 || page.SchemaVersion == 0 {
		return errors.New("page revision and schema versions must be positive")
	}
	if err := ValidateCanonicalJSON(page.ContentJSON, page.Checksum); err != nil {
		return fmt.Errorf("invalid published page content: %w", err)
	}
	if page.PublishedAt.IsZero() {
		return errors.New("published page must have a publication timestamp")
	}
	return nil
}
