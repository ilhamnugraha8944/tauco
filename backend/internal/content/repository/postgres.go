// Package repository contains PostgreSQL adapters for the content application
// ports. SQL is schema-qualified deliberately: the application schema is
// private and must not depend on a connection's search_path.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	"gorm.io/gorm"
)

// PostgresRepository implements content read and seed ports with PostgreSQL.
// GORM AutoMigrate is intentionally never used; schema changes are owned by
// versioned SQL migrations.
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository binds the adapter to an already configured GORM
// connection.
func NewPostgresRepository(db *gorm.DB) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("content PostgreSQL repository requires a database")
	}
	return &PostgresRepository{db: db}, nil
}

type publishedPageRow struct {
	PageID         string          `gorm:"column:page_id"`
	RevisionID     string          `gorm:"column:revision_id"`
	Key            string          `gorm:"column:key"`
	RevisionNumber uint32          `gorm:"column:revision_number"`
	SchemaVersion  uint32          `gorm:"column:schema_version"`
	ContentJSON    json.RawMessage `gorm:"column:content_json"`
	Checksum       string          `gorm:"column:content_checksum"`
	PublishedAt    time.Time       `gorm:"column:published_at"`
}

// FindPublishedPage returns only the revision selected by the page's current
// published pointer. Draft and archived revisions are never eligible.
func (repository *PostgresRepository) FindPublishedPage(
	ctx context.Context,
	key contentdomain.PageKey,
) (contentdomain.PublishedPage, error) {
	if repository == nil || repository.db == nil {
		return contentdomain.PublishedPage{}, errors.New(
			"content PostgreSQL repository is not initialized",
		)
	}
	if !key.Valid() {
		return contentdomain.PublishedPage{}, contentapp.ErrPublishedPageNotFound
	}

	var row publishedPageRow
	result := repository.db.WithContext(ctx).Raw(`
SELECT
    page.id AS page_id,
    revision.id AS revision_id,
    page.key,
    revision.revision_number,
    revision.schema_version,
    revision.content_json,
    revision.content_checksum,
    revision.published_at
FROM tauco_app.pages AS page
JOIN tauco_app.page_revisions AS revision
  ON revision.id = page.published_revision_id
 AND revision.page_id = page.id
WHERE page.key = ?
  AND revision.status = 'published'
LIMIT 1`,
		string(key),
	).Scan(&row)
	if result.Error != nil {
		return contentdomain.PublishedPage{}, fmt.Errorf(
			"query published page %q: %w",
			key,
			result.Error,
		)
	}
	if result.RowsAffected == 0 {
		return contentdomain.PublishedPage{}, contentapp.ErrPublishedPageNotFound
	}

	pageID, err := contentdomain.ParseUUIDv7(row.PageID)
	if err != nil {
		return contentdomain.PublishedPage{}, fmt.Errorf(
			"hydrate published page ID: %w",
			err,
		)
	}
	revisionID, err := contentdomain.ParseUUIDv7(row.RevisionID)
	if err != nil {
		return contentdomain.PublishedPage{}, fmt.Errorf(
			"hydrate published page revision ID: %w",
			err,
		)
	}
	checksum, err := contentdomain.ParseSHA256Checksum(row.Checksum)
	if err != nil {
		return contentdomain.PublishedPage{}, fmt.Errorf(
			"hydrate published page checksum: %w",
			err,
		)
	}
	canonicalContent, actualChecksum, err :=
		contentdomain.CanonicalJSONChecksum(row.ContentJSON)
	if err != nil {
		return contentdomain.PublishedPage{}, fmt.Errorf(
			"canonicalize persisted published page content: %w",
			err,
		)
	}
	if actualChecksum != checksum {
		return contentdomain.PublishedPage{}, errors.New(
			"persisted published page content checksum mismatch",
		)
	}

	page := contentdomain.PublishedPage{
		PageID:         pageID,
		RevisionID:     revisionID,
		Key:            contentdomain.PageKey(row.Key),
		RevisionNumber: row.RevisionNumber,
		SchemaVersion:  row.SchemaVersion,
		ContentJSON:    append(json.RawMessage(nil), canonicalContent...),
		Checksum:       checksum,
		PublishedAt:    row.PublishedAt.UTC(),
	}
	if err := page.Validate(); err != nil {
		return contentdomain.PublishedPage{}, fmt.Errorf(
			"validate hydrated published page: %w",
			err,
		)
	}
	return page, nil
}
