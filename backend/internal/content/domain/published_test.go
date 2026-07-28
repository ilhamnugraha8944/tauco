package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPublishedPageValidate(t *testing.T) {
	t.Parallel()

	content, checksum, err := CanonicalJSONChecksum(
		json.RawMessage(`{"title":"Tauco"}`),
	)
	if err != nil {
		t.Fatalf("CanonicalJSONChecksum() error = %v", err)
	}
	page := PublishedPage{
		PageID:         "019bfc80-0000-7000-8000-000000000001",
		RevisionID:     "019bfc80-0000-7000-8000-000000000011",
		Key:            PageKeyHome,
		RevisionNumber: 1,
		SchemaVersion:  1,
		ContentJSON:    content,
		Checksum:       checksum,
		PublishedAt:    time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	page.ContentJSON = json.RawMessage(`{`)
	if err := page.Validate(); err == nil {
		t.Fatal("Validate() unexpectedly accepted invalid JSON")
	}

	page.ContentJSON = content
	page.Checksum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := page.Validate(); err == nil {
		t.Fatal("Validate() unexpectedly accepted checksum drift")
	}
}
