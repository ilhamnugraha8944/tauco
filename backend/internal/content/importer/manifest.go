// Package importer builds a deterministic persistence plan from the committed
// Phase 1A JSON documents.
package importer

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

const (
	phase1ASchemaVersion  uint32 = 1
	initialRevisionNumber uint32 = 1
)

var phase1APublishedAt = time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)

// PageManifestEntry preassigns the stable persistence identities for a
// singleton page.
type PageManifestEntry struct {
	Key            domain.PageKey
	SourceFilename string
	EntityID       domain.UUIDv7
	RevisionID     domain.UUIDv7
	RevisionNumber uint32
	SchemaVersion  uint32
	PublishedAt    time.Time
}

// ProductManifestEntry preassigns the stable persistence identities for a
// product. New products require an explicit manifest change; IDs are never
// derived from mutable content.
type ProductManifestEntry struct {
	Slug           string
	EntityID       domain.UUIDv7
	RevisionID     domain.UUIDv7
	RevisionNumber uint32
	SchemaVersion  uint32
	PublishedAt    time.Time
}

// Manifest is the versioned identity map for the deterministic import.
type Manifest struct {
	Version  uint32
	Pages    []PageManifestEntry
	Products []ProductManifestEntry
}

// Phase1AManifest returns a defensive copy of the committed identity map.
func Phase1AManifest() Manifest {
	manifest := Manifest{
		Version: application.Phase1AManifestVersion,
		Pages: []PageManifestEntry{
			{
				Key:            domain.PageKeyHome,
				SourceFilename: "home.json",
				EntityID:       "019bfc80-0000-7000-8000-000000000001",
				RevisionID:     "019bfc80-0000-7000-8000-000000000011",
				RevisionNumber: initialRevisionNumber,
				SchemaVersion:  phase1ASchemaVersion,
				PublishedAt:    phase1APublishedAt,
			},
			{
				Key:            domain.PageKeyAbout,
				SourceFilename: "about.json",
				EntityID:       "019bfc80-0000-7000-8000-000000000002",
				RevisionID:     "019bfc80-0000-7000-8000-000000000012",
				RevisionNumber: initialRevisionNumber,
				SchemaVersion:  phase1ASchemaVersion,
				PublishedAt:    phase1APublishedAt,
			},
			{
				Key:            domain.PageKeyTaucoGuide,
				SourceFilename: "tauco-guide.json",
				EntityID:       "019bfc80-0000-7000-8000-000000000003",
				RevisionID:     "019bfc80-0000-7000-8000-000000000013",
				RevisionNumber: initialRevisionNumber,
				SchemaVersion:  phase1ASchemaVersion,
				PublishedAt:    phase1APublishedAt,
			},
			{
				Key:            domain.PageKeyProducts,
				SourceFilename: "products.json",
				EntityID:       "019bfc80-0000-7000-8000-000000000004",
				RevisionID:     "019bfc80-0000-7000-8000-000000000014",
				RevisionNumber: initialRevisionNumber,
				SchemaVersion:  phase1ASchemaVersion,
				PublishedAt:    phase1APublishedAt,
			},
		},
		Products: []ProductManifestEntry{
			{
				Slug:           "tauco-cap-badak",
				EntityID:       "019bfc80-0000-7000-8000-000000000101",
				RevisionID:     "019bfc80-0000-7000-8000-000000000111",
				RevisionNumber: initialRevisionNumber,
				SchemaVersion:  phase1ASchemaVersion,
				PublishedAt:    phase1APublishedAt,
			},
		},
	}
	manifest.Pages = append([]PageManifestEntry(nil), manifest.Pages...)
	manifest.Products = append([]ProductManifestEntry(nil), manifest.Products...)
	return manifest
}

// Validate rejects identity collisions and incomplete manifest entries before
// any source file is parsed.
func (manifest Manifest) Validate() error {
	if manifest.Version != application.Phase1AManifestVersion {
		return fmt.Errorf("unsupported manifest version %d", manifest.Version)
	}
	if len(manifest.Pages) != 4 {
		return fmt.Errorf("manifest must contain 4 singleton pages, got %d", len(manifest.Pages))
	}
	if len(manifest.Products) == 0 {
		return errors.New("manifest must contain at least one product")
	}

	pageKeys := make(map[domain.PageKey]struct{}, len(manifest.Pages))
	filenames := make(map[string]struct{}, len(manifest.Pages))
	productSlugs := make(map[string]struct{}, len(manifest.Products))
	identities := make(map[domain.UUIDv7]string, 2*(len(manifest.Pages)+len(manifest.Products)))

	for index, page := range manifest.Pages {
		if !page.Key.Valid() {
			return fmt.Errorf("pages[%d] has invalid key %q", index, page.Key)
		}
		if page.SourceFilename == "" {
			return fmt.Errorf("page %q has an empty source filename", page.Key)
		}
		if filepath.Base(page.SourceFilename) != page.SourceFilename ||
			filepath.Ext(page.SourceFilename) != ".json" {
			return fmt.Errorf(
				"page %q source filename must be a local .json basename",
				page.Key,
			)
		}
		if _, duplicate := pageKeys[page.Key]; duplicate {
			return fmt.Errorf("page key %q is duplicated", page.Key)
		}
		pageKeys[page.Key] = struct{}{}
		if _, duplicate := filenames[page.SourceFilename]; duplicate {
			return fmt.Errorf("source file %q is duplicated", page.SourceFilename)
		}
		filenames[page.SourceFilename] = struct{}{}
		if err := validateManifestRevision(
			identities,
			"page:"+string(page.Key),
			page.EntityID,
			page.RevisionID,
			page.RevisionNumber,
			page.SchemaVersion,
			page.PublishedAt,
		); err != nil {
			return err
		}
	}

	requiredPages := []domain.PageKey{
		domain.PageKeyHome,
		domain.PageKeyAbout,
		domain.PageKeyTaucoGuide,
		domain.PageKeyProducts,
	}
	for _, key := range requiredPages {
		if _, exists := pageKeys[key]; !exists {
			return fmt.Errorf("manifest is missing page key %q", key)
		}
	}
	expectedFilenames := map[domain.PageKey]string{
		domain.PageKeyHome:       "home.json",
		domain.PageKeyAbout:      "about.json",
		domain.PageKeyTaucoGuide: "tauco-guide.json",
		domain.PageKeyProducts:   "products.json",
	}
	for _, page := range manifest.Pages {
		if page.SourceFilename != expectedFilenames[page.Key] {
			return fmt.Errorf(
				"page %q source filename must be %q",
				page.Key,
				expectedFilenames[page.Key],
			)
		}
	}

	for index, product := range manifest.Products {
		if err := domain.ValidateProductSlug(product.Slug); err != nil {
			return fmt.Errorf("products[%d]: %w", index, err)
		}
		if _, duplicate := productSlugs[product.Slug]; duplicate {
			return fmt.Errorf("product slug %q is duplicated", product.Slug)
		}
		productSlugs[product.Slug] = struct{}{}
		if err := validateManifestRevision(
			identities,
			"product:"+product.Slug,
			product.EntityID,
			product.RevisionID,
			product.RevisionNumber,
			product.SchemaVersion,
			product.PublishedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateManifestRevision(
	identities map[domain.UUIDv7]string,
	owner string,
	entityID domain.UUIDv7,
	revisionID domain.UUIDv7,
	revisionNumber uint32,
	schemaVersion uint32,
	publishedAt time.Time,
) error {
	for label, identifier := range map[string]domain.UUIDv7{
		"entity":   entityID,
		"revision": revisionID,
	} {
		if _, err := domain.ParseUUIDv7(string(identifier)); err != nil {
			return fmt.Errorf("%s %s ID: %w", owner, label, err)
		}
		if previous, duplicate := identities[identifier]; duplicate {
			return fmt.Errorf("%s %s ID is already assigned to %s", owner, label, previous)
		}
		identities[identifier] = owner + ":" + label
	}
	if entityID == revisionID {
		return fmt.Errorf("%s entity and revision IDs must differ", owner)
	}
	if revisionNumber == 0 || schemaVersion == 0 {
		return fmt.Errorf("%s revision and schema versions must be positive", owner)
	}
	if publishedAt.IsZero() || publishedAt.Location() != time.UTC {
		return fmt.Errorf("%s publication timestamp must be non-zero UTC", owner)
	}
	return nil
}
