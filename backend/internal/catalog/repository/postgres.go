// Package repository contains PostgreSQL adapters for catalog application
// ports.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	catalogdomain "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	"gorm.io/gorm"
)

// PostgresRepository reads published products from PostgreSQL. It never calls
// AutoMigrate; versioned SQL owns the schema.
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository binds the adapter to an already configured GORM
// connection.
func NewPostgresRepository(db *gorm.DB) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("catalog PostgreSQL repository requires a database")
	}
	return &PostgresRepository{db: db}, nil
}

type publishedProductRow struct {
	ProductID      string          `gorm:"column:product_id"`
	RevisionID     string          `gorm:"column:revision_id"`
	Slug           string          `gorm:"column:slug"`
	SortOrder      int             `gorm:"column:sort_order"`
	RevisionNumber uint32          `gorm:"column:revision_number"`
	SchemaVersion  uint32          `gorm:"column:schema_version"`
	ContentJSON    json.RawMessage `gorm:"column:content_json"`
	Checksum       string          `gorm:"column:content_checksum"`
	PublishedAt    time.Time       `gorm:"column:published_at"`
}

const publishedProductProjection = `
SELECT
    product.id AS product_id,
    revision.id AS revision_id,
    product.slug,
    product.sort_order,
    revision.revision_number,
    revision.schema_version,
    revision.content_json,
    revision.content_checksum,
    revision.published_at
FROM tauco_app.products AS product
JOIN tauco_app.product_revisions AS revision
  ON revision.id = product.published_revision_id
 AND revision.product_id = product.id
`

// FindPublishedProduct returns the current published revision for one stable
// slug.
func (repository *PostgresRepository) FindPublishedProduct(
	ctx context.Context,
	slug string,
) (catalogdomain.PublishedProduct, error) {
	if repository == nil || repository.db == nil {
		return catalogdomain.PublishedProduct{}, errors.New(
			"catalog PostgreSQL repository is not initialized",
		)
	}
	if err := contentdomain.ValidateProductSlug(slug); err != nil {
		return catalogdomain.PublishedProduct{},
			catalogapp.ErrPublishedProductNotFound
	}

	var row publishedProductRow
	result := repository.db.WithContext(ctx).Raw(
		publishedProductProjection+`
WHERE product.slug = ?
  AND product.archived_at IS NULL
  AND revision.status = 'published'
LIMIT 1`,
		slug,
	).Scan(&row)
	if result.Error != nil {
		return catalogdomain.PublishedProduct{}, fmt.Errorf(
			"query published product %q: %w",
			slug,
			result.Error,
		)
	}
	if result.RowsAffected == 0 {
		return catalogdomain.PublishedProduct{},
			catalogapp.ErrPublishedProductNotFound
	}
	return hydratePublishedProduct(row)
}

// ListPublishedProducts uses stable keyset pagination ordered by
// `(sort_order ASC, id ASC)`.
func (repository *PostgresRepository) ListPublishedProducts(
	ctx context.Context,
	after *catalogdomain.PaginationPosition,
	limit int,
) (catalogapp.PublishedProductPage, error) {
	if repository == nil || repository.db == nil {
		return catalogapp.PublishedProductPage{}, errors.New(
			"catalog PostgreSQL repository is not initialized",
		)
	}
	if err := catalogapp.ValidatePageLimit(limit); err != nil {
		return catalogapp.PublishedProductPage{}, err
	}

	statement := publishedProductProjection + `
WHERE product.archived_at IS NULL
  AND revision.status = 'published'`
	arguments := make([]any, 0, 3)
	if after != nil {
		if after.SortOrder < 0 {
			return catalogapp.PublishedProductPage{},
				catalogapp.ErrInvalidPaginationPosition
		}
		if _, err := contentdomain.ParseUUIDv7(string(after.ProductID)); err != nil {
			return catalogapp.PublishedProductPage{},
				catalogapp.ErrInvalidPaginationPosition
		}
		statement += `
  AND (
      product.sort_order > ?
      OR (product.sort_order = ? AND product.id > ?::uuid)
  )`
		arguments = append(
			arguments,
			after.SortOrder,
			after.SortOrder,
			string(after.ProductID),
		)
	}
	statement += `
ORDER BY product.sort_order ASC, product.id ASC
LIMIT ?`
	arguments = append(arguments, limit+1)

	var rows []publishedProductRow
	result := repository.db.WithContext(ctx).Raw(statement, arguments...).Scan(&rows)
	if result.Error != nil {
		return catalogapp.PublishedProductPage{}, fmt.Errorf(
			"list published products: %w",
			result.Error,
		)
	}

	products := make([]catalogdomain.PublishedProduct, 0, len(rows))
	for index, row := range rows {
		product, err := hydratePublishedProduct(row)
		if err != nil {
			return catalogapp.PublishedProductPage{}, fmt.Errorf(
				"hydrate published product at index %d: %w",
				index,
				err,
			)
		}
		products = append(products, product)
	}
	hasMore := len(products) > limit
	if hasMore {
		products = products[:limit]
	}
	return catalogapp.PublishedProductPage{
		Products: products,
		HasMore:  hasMore,
	}, nil
}

func hydratePublishedProduct(
	row publishedProductRow,
) (catalogdomain.PublishedProduct, error) {
	productID, err := contentdomain.ParseUUIDv7(row.ProductID)
	if err != nil {
		return catalogdomain.PublishedProduct{}, fmt.Errorf(
			"hydrate product ID: %w",
			err,
		)
	}
	revisionID, err := contentdomain.ParseUUIDv7(row.RevisionID)
	if err != nil {
		return catalogdomain.PublishedProduct{}, fmt.Errorf(
			"hydrate product revision ID: %w",
			err,
		)
	}
	checksum, err := contentdomain.ParseSHA256Checksum(row.Checksum)
	if err != nil {
		return catalogdomain.PublishedProduct{}, fmt.Errorf(
			"hydrate product checksum: %w",
			err,
		)
	}
	canonicalContent, actualChecksum, err :=
		contentdomain.CanonicalJSONChecksum(row.ContentJSON)
	if err != nil {
		return catalogdomain.PublishedProduct{}, fmt.Errorf(
			"canonicalize persisted published product content: %w",
			err,
		)
	}
	if actualChecksum != checksum {
		return catalogdomain.PublishedProduct{}, errors.New(
			"persisted published product content checksum mismatch",
		)
	}
	product := catalogdomain.PublishedProduct{
		ProductID:      productID,
		RevisionID:     revisionID,
		Slug:           row.Slug,
		SortOrder:      row.SortOrder,
		RevisionNumber: row.RevisionNumber,
		SchemaVersion:  row.SchemaVersion,
		ContentJSON:    append(json.RawMessage(nil), canonicalContent...),
		Checksum:       checksum,
		PublishedAt:    row.PublishedAt.UTC(),
	}
	if err := product.Validate(); err != nil {
		return catalogdomain.PublishedProduct{}, err
	}
	return product, nil
}
