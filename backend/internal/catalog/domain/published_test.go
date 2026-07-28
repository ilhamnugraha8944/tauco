package domain

import (
	"encoding/json"
	"testing"
	"time"

	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

func TestPublishedProductValidate(t *testing.T) {
	t.Parallel()

	content, checksum, err := contentdomain.CanonicalJSONChecksum(
		json.RawMessage(`{"name":"Tauco Cap Badak"}`),
	)
	if err != nil {
		t.Fatalf("CanonicalJSONChecksum() error = %v", err)
	}
	product := PublishedProduct{
		ProductID:      "019bfc80-0000-7000-8000-000000000101",
		RevisionID:     "019bfc80-0000-7000-8000-000000000111",
		Slug:           "tauco-cap-badak",
		SortOrder:      0,
		RevisionNumber: 1,
		SchemaVersion:  1,
		ContentJSON:    content,
		Checksum:       checksum,
		PublishedAt:    time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
	}
	if err := product.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	product.SortOrder = -1
	if err := product.Validate(); err == nil {
		t.Fatal("Validate() unexpectedly accepted a negative sort order")
	}

	product.SortOrder = 0
	product.Checksum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := product.Validate(); err == nil {
		t.Fatal("Validate() unexpectedly accepted checksum drift")
	}
}
