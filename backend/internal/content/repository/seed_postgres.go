package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	"gorm.io/gorm"
)

// phase1ASeedAdvisoryLockKey is the stable, transaction-scoped PostgreSQL
// advisory lock used by every Phase 1A importer execution. The hexadecimal
// value is the ASCII representation of "TAUCO_P1".
const phase1ASeedAdvisoryLockKey int64 = 0x544155434f5f5031

var _ contentapp.SeedStore = (*PostgresRepository)(nil)

type pageSeedSnapshotRow struct {
	EntityID          string          `gorm:"column:entity_id"`
	NaturalKey        string          `gorm:"column:natural_key"`
	PublishedRevision sql.NullString  `gorm:"column:published_revision_id"`
	RevisionID        sql.NullString  `gorm:"column:revision_id"`
	RevisionNumber    sql.NullInt64   `gorm:"column:revision_number"`
	SchemaVersion     sql.NullInt64   `gorm:"column:schema_version"`
	Status            sql.NullString  `gorm:"column:status"`
	ContentJSON       json.RawMessage `gorm:"column:content_json"`
	ContentChecksum   sql.NullString  `gorm:"column:content_checksum"`
	PublishedAt       sql.NullTime    `gorm:"column:published_at"`
}

type productSeedSnapshotRow struct {
	EntityID          string          `gorm:"column:entity_id"`
	NaturalKey        string          `gorm:"column:natural_key"`
	SortOrder         int64           `gorm:"column:sort_order"`
	PublishedRevision sql.NullString  `gorm:"column:published_revision_id"`
	FirstPublishedAt  sql.NullTime    `gorm:"column:first_published_at"`
	RevisionID        sql.NullString  `gorm:"column:revision_id"`
	RevisionNumber    sql.NullInt64   `gorm:"column:revision_number"`
	SchemaVersion     sql.NullInt64   `gorm:"column:schema_version"`
	Status            sql.NullString  `gorm:"column:status"`
	ContentJSON       json.RawMessage `gorm:"column:content_json"`
	ContentChecksum   sql.NullString  `gorm:"column:content_checksum"`
	PublishedAt       sql.NullTime    `gorm:"column:published_at"`
}

type occupiedSeedIdentityRow struct {
	Identity   string `gorm:"column:identity"`
	OwnerKind  string `gorm:"column:owner_kind"`
	NaturalKey string `gorm:"column:natural_key"`
	Source     string `gorm:"column:source"`
}

// ApplyPhase1A implements application.SeedStore. It validates before opening a
// transaction, serializes cooperating importers, inspects all deterministic
// identities, and uses insert-only persistence. It never mutates an existing
// published revision.
func (repository *PostgresRepository) ApplyPhase1A(
	ctx context.Context,
	plan contentapp.SeedPlan,
) (contentapp.SeedApplyResult, error) {
	if repository == nil || repository.db == nil {
		return contentapp.SeedApplyResult{}, errors.New(
			"content PostgreSQL repository is not initialized",
		)
	}
	if ctx == nil {
		return contentapp.SeedApplyResult{}, errors.New(
			"apply Phase 1A seed: context is required",
		)
	}

	// Keep this validation outside db.Transaction. Invalid content must not
	// open a transaction, acquire a lock, or issue any persistence statement.
	if err := plan.Validate(); err != nil {
		return contentapp.SeedApplyResult{}, fmt.Errorf(
			"validate Phase 1A seed plan: %w",
			err,
		)
	}

	var applied contentapp.SeedApplyResult
	err := repository.db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		// This must remain the first statement after BEGIN. Acquiring the lock
		// before SET LOCAL ROLE also works with a bootstrap connection whose
		// only elevated capability is membership in tauco_migrator.
		if err := transaction.Exec(
			"SELECT pg_advisory_xact_lock(?)",
			phase1ASeedAdvisoryLockKey,
		).Error; err != nil {
			return fmt.Errorf("acquire Phase 1A importer lock: %w", err)
		}
		if err := transaction.Exec("SET LOCAL ROLE tauco_migrator").Error; err != nil {
			return fmt.Errorf("assume PostgreSQL migrator role: %w", err)
		}

		snapshot, err := loadSeedSnapshot(transaction, plan)
		if err != nil {
			return err
		}
		actions, err := contentapp.Reconcile(plan, snapshot)
		if err != nil {
			return err
		}

		pages, products := indexSeedPlan(plan)
		for _, action := range actions {
			switch action.Kind {
			case contentapp.ReconcileNoop:
				applied.Unchanged++
			case contentapp.ReconcileInsert:
				switch action.Key.Kind {
				case contentapp.SeedEntityPage:
					page, exists := pages[action.Key.NaturalKey]
					if !exists {
						return fmt.Errorf(
							"validated page seed %q is missing from its index",
							action.Key.NaturalKey,
						)
					}
					if err := insertPageSeed(transaction, page); err != nil {
						return err
					}
				case contentapp.SeedEntityProduct:
					product, exists := products[action.Key.NaturalKey]
					if !exists {
						return fmt.Errorf(
							"validated product seed %q is missing from its index",
							action.Key.NaturalKey,
						)
					}
					if err := insertProductSeed(transaction, product); err != nil {
						return err
					}
				default:
					return fmt.Errorf(
						"unsupported seed entity kind %q",
						action.Key.Kind,
					)
				}
				applied.Inserted++
			default:
				return fmt.Errorf(
					"unsupported seed reconcile action %q",
					action.Kind,
				)
			}
		}
		return nil
	})
	if err != nil {
		return contentapp.SeedApplyResult{}, fmt.Errorf(
			"apply Phase 1A seed transaction: %w",
			normalizeSeedPersistenceError(err),
		)
	}
	return applied, nil
}

func loadSeedSnapshot(
	transaction *gorm.DB,
	plan contentapp.SeedPlan,
) (contentapp.SeedSnapshot, error) {
	snapshot := contentapp.SeedSnapshot{
		Records: make(
			map[contentapp.SeedRecordKey]contentapp.StoredSeedRecord,
			len(plan.Pages)+len(plan.Products),
		),
		IdentityOwners: make(
			map[contentdomain.UUIDv7]contentapp.SeedRecordKey,
			2*(len(plan.Pages)+len(plan.Products)),
		),
	}

	pageKeys := make([]string, 0, len(plan.Pages))
	productSlugs := make([]string, 0, len(plan.Products))
	expectedIdentities := make(
		[]string,
		0,
		2*(len(plan.Pages)+len(plan.Products)),
	)
	for _, page := range plan.Pages {
		pageKeys = append(pageKeys, string(page.Key))
		expectedIdentities = append(
			expectedIdentities,
			string(page.Revision.EntityID),
			string(page.Revision.RevisionID),
		)
	}
	for _, product := range plan.Products {
		productSlugs = append(productSlugs, product.Slug)
		expectedIdentities = append(
			expectedIdentities,
			string(product.Revision.EntityID),
			string(product.Revision.RevisionID),
		)
	}

	if err := loadPageSeedRecords(transaction, pageKeys, snapshot.Records); err != nil {
		return contentapp.SeedSnapshot{}, err
	}
	if err := loadProductSeedRecords(
		transaction,
		productSlugs,
		snapshot.Records,
	); err != nil {
		return contentapp.SeedSnapshot{}, err
	}
	if err := loadOccupiedSeedIdentities(
		transaction,
		expectedIdentities,
		snapshot.IdentityOwners,
	); err != nil {
		return contentapp.SeedSnapshot{}, err
	}
	return snapshot, nil
}

func loadPageSeedRecords(
	transaction *gorm.DB,
	pageKeys []string,
	records map[contentapp.SeedRecordKey]contentapp.StoredSeedRecord,
) error {
	var rows []pageSeedSnapshotRow
	result := transaction.Raw(`
SELECT
    page.id AS entity_id,
    page.key AS natural_key,
    page.published_revision_id,
    revision.id AS revision_id,
    revision.revision_number,
    revision.schema_version,
    revision.status,
    revision.content_json,
    revision.content_checksum,
    revision.published_at
FROM tauco_app.pages AS page
LEFT JOIN tauco_app.page_revisions AS revision
  ON revision.id = page.published_revision_id
 AND revision.page_id = page.id
WHERE page.key IN ?
ORDER BY page.key`,
		pageKeys,
	).Scan(&rows)
	if result.Error != nil {
		return fmt.Errorf("snapshot Phase 1A page natural keys: %w", result.Error)
	}

	for _, row := range rows {
		key := contentapp.SeedRecordKey{
			Kind:       contentapp.SeedEntityPage,
			NaturalKey: row.NaturalKey,
		}
		if _, duplicate := records[key]; duplicate {
			return seedConflictError(
				"page natural key %q resolved more than once",
				row.NaturalKey,
			)
		}
		record, err := hydratePageSeedRecord(row)
		if err != nil {
			return err
		}
		records[key] = record
	}
	return nil
}

func loadProductSeedRecords(
	transaction *gorm.DB,
	productSlugs []string,
	records map[contentapp.SeedRecordKey]contentapp.StoredSeedRecord,
) error {
	var rows []productSeedSnapshotRow
	result := transaction.Raw(`
SELECT
    product.id AS entity_id,
    product.slug AS natural_key,
    product.sort_order,
    product.published_revision_id,
    product.first_published_at,
    revision.id AS revision_id,
    revision.revision_number,
    revision.schema_version,
    revision.status,
    revision.content_json,
    revision.content_checksum,
    revision.published_at
FROM tauco_app.products AS product
LEFT JOIN tauco_app.product_revisions AS revision
  ON revision.id = product.published_revision_id
 AND revision.product_id = product.id
WHERE product.slug IN ?
ORDER BY product.slug`,
		productSlugs,
	).Scan(&rows)
	if result.Error != nil {
		return fmt.Errorf(
			"snapshot Phase 1A product natural keys: %w",
			result.Error,
		)
	}

	for _, row := range rows {
		key := contentapp.SeedRecordKey{
			Kind:       contentapp.SeedEntityProduct,
			NaturalKey: row.NaturalKey,
		}
		if _, duplicate := records[key]; duplicate {
			return seedConflictError(
				"product natural key %q resolved more than once",
				row.NaturalKey,
			)
		}
		record, err := hydrateProductSeedRecord(row)
		if err != nil {
			return err
		}
		records[key] = record
	}
	return nil
}

func loadOccupiedSeedIdentities(
	transaction *gorm.DB,
	expectedIdentities []string,
	owners map[contentdomain.UUIDv7]contentapp.SeedRecordKey,
) error {
	// UNION ALL is intentional. A UUID that appears in more than one of these
	// tables is a cross-table collision even though each table's primary key
	// constraint alone permits it.
	var rows []occupiedSeedIdentityRow
	result := transaction.Raw(`
SELECT
    occupied.identity,
    occupied.owner_kind,
    occupied.natural_key,
    occupied.source
FROM (
    SELECT
        page.id AS identity,
        'page'::text AS owner_kind,
        page.key AS natural_key,
        'pages.id'::text AS source
    FROM tauco_app.pages AS page
    UNION ALL
    SELECT
        revision.id AS identity,
        'page'::text AS owner_kind,
        page.key AS natural_key,
        'page_revisions.id'::text AS source
    FROM tauco_app.page_revisions AS revision
    JOIN tauco_app.pages AS page
      ON page.id = revision.page_id
    UNION ALL
    SELECT
        product.id AS identity,
        'product'::text AS owner_kind,
        product.slug AS natural_key,
        'products.id'::text AS source
    FROM tauco_app.products AS product
    UNION ALL
    SELECT
        revision.id AS identity,
        'product'::text AS owner_kind,
        product.slug AS natural_key,
        'product_revisions.id'::text AS source
    FROM tauco_app.product_revisions AS revision
    JOIN tauco_app.products AS product
      ON product.id = revision.product_id
) AS occupied
WHERE occupied.identity IN ?
ORDER BY occupied.identity, occupied.source`,
		expectedIdentities,
	).Scan(&rows)
	if result.Error != nil {
		return fmt.Errorf(
			"snapshot Phase 1A global UUID occupancy: %w",
			result.Error,
		)
	}

	sources := make(map[contentdomain.UUIDv7]string, len(rows))
	for _, row := range rows {
		identity, err := contentdomain.ParseUUIDv7(row.Identity)
		if err != nil {
			return seedConflictError(
				"persisted identity %q from %s is invalid",
				row.Identity,
				row.Source,
			)
		}
		if previousSource, duplicate := sources[identity]; duplicate {
			return seedConflictError(
				"UUID %q is occupied by both %s and %s",
				identity,
				previousSource,
				row.Source,
			)
		}

		var kind contentapp.SeedEntityKind
		switch row.OwnerKind {
		case string(contentapp.SeedEntityPage):
			kind = contentapp.SeedEntityPage
		case string(contentapp.SeedEntityProduct):
			kind = contentapp.SeedEntityProduct
		default:
			return seedConflictError(
				"UUID %q has unsupported owner kind %q",
				identity,
				row.OwnerKind,
			)
		}
		sources[identity] = row.Source
		owners[identity] = contentapp.SeedRecordKey{
			Kind:       kind,
			NaturalKey: row.NaturalKey,
		}
	}
	return nil
}

func hydratePageSeedRecord(
	row pageSeedSnapshotRow,
) (contentapp.StoredSeedRecord, error) {
	if !row.PublishedRevision.Valid ||
		!row.RevisionID.Valid ||
		!row.RevisionNumber.Valid ||
		!row.SchemaVersion.Valid ||
		!row.Status.Valid ||
		!row.ContentChecksum.Valid ||
		!row.PublishedAt.Valid ||
		len(row.ContentJSON) == 0 {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"page %q is only partially persisted",
			row.NaturalKey,
		)
	}
	if row.PublishedRevision.String != row.RevisionID.String {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"page %q published pointer does not resolve to its revision",
			row.NaturalKey,
		)
	}
	return hydrateStoredRevision(
		contentapp.SeedEntityPage,
		row.NaturalKey,
		row.EntityID,
		row.RevisionID.String,
		row.PublishedRevision.String,
		row.RevisionNumber.Int64,
		row.SchemaVersion.Int64,
		row.Status.String,
		row.ContentJSON,
		row.ContentChecksum.String,
		row.PublishedAt.Time,
		nil,
	)
}

func hydrateProductSeedRecord(
	row productSeedSnapshotRow,
) (contentapp.StoredSeedRecord, error) {
	if !row.PublishedRevision.Valid ||
		!row.FirstPublishedAt.Valid ||
		!row.RevisionID.Valid ||
		!row.RevisionNumber.Valid ||
		!row.SchemaVersion.Valid ||
		!row.Status.Valid ||
		!row.ContentChecksum.Valid ||
		!row.PublishedAt.Valid ||
		len(row.ContentJSON) == 0 {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"product %q is only partially persisted",
			row.NaturalKey,
		)
	}
	if row.PublishedRevision.String != row.RevisionID.String {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"product %q published pointer does not resolve to its revision",
			row.NaturalKey,
		)
	}
	if !row.FirstPublishedAt.Time.Equal(row.PublishedAt.Time) {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"product %q has inconsistent first publication time",
			row.NaturalKey,
		)
	}
	if row.SortOrder < 0 || row.SortOrder > int64(math.MaxInt) {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"product %q has invalid sort order",
			row.NaturalKey,
		)
	}
	sortOrder := int(row.SortOrder)
	return hydrateStoredRevision(
		contentapp.SeedEntityProduct,
		row.NaturalKey,
		row.EntityID,
		row.RevisionID.String,
		row.PublishedRevision.String,
		row.RevisionNumber.Int64,
		row.SchemaVersion.Int64,
		row.Status.String,
		row.ContentJSON,
		row.ContentChecksum.String,
		row.PublishedAt.Time,
		&sortOrder,
	)
}

func hydrateStoredRevision(
	kind contentapp.SeedEntityKind,
	naturalKey string,
	entityIDValue string,
	revisionIDValue string,
	publishedRevisionIDValue string,
	revisionNumber int64,
	schemaVersion int64,
	statusValue string,
	contentJSON json.RawMessage,
	checksumValue string,
	publishedAt time.Time,
	sortOrder *int,
) (contentapp.StoredSeedRecord, error) {
	entityID, err := contentdomain.ParseUUIDv7(entityIDValue)
	if err != nil {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q has an invalid entity ID",
			kind,
			naturalKey,
		)
	}
	revisionID, err := contentdomain.ParseUUIDv7(revisionIDValue)
	if err != nil {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q has an invalid revision ID",
			kind,
			naturalKey,
		)
	}
	publishedRevisionID, err := contentdomain.ParseUUIDv7(
		publishedRevisionIDValue,
	)
	if err != nil {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q has an invalid published pointer",
			kind,
			naturalKey,
		)
	}
	if revisionNumber <= 0 || revisionNumber > int64(math.MaxUint32) {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q has an invalid revision number",
			kind,
			naturalKey,
		)
	}
	if schemaVersion <= 0 || schemaVersion > int64(math.MaxUint32) {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q has an invalid schema version",
			kind,
			naturalKey,
		)
	}
	status := contentdomain.RevisionStatus(statusValue)
	if !status.Valid() || status != contentdomain.RevisionStatusPublished {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q does not point to a published revision",
			kind,
			naturalKey,
		)
	}
	checksum, err := contentdomain.ParseSHA256Checksum(checksumValue)
	if err != nil {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q has an invalid content checksum",
			kind,
			naturalKey,
		)
	}
	_, computedChecksum, err := contentdomain.CanonicalJSONChecksum(contentJSON)
	if err != nil || computedChecksum != checksum {
		return contentapp.StoredSeedRecord{}, seedConflictError(
			"%s %q content does not match its checksum",
			kind,
			naturalKey,
		)
	}

	return contentapp.StoredSeedRecord{
		EntityID:            entityID,
		RevisionID:          revisionID,
		RevisionNumber:      uint32(revisionNumber),
		SchemaVersion:       uint32(schemaVersion),
		Status:              status,
		Checksum:            checksum,
		PublishedAt:         publishedAt.UTC(),
		PublishedRevisionID: &publishedRevisionID,
		ProductSortOrder:    copyIntPointer(sortOrder),
	}, nil
}

func insertPageSeed(
	transaction *gorm.DB,
	page contentapp.PageSeed,
) error {
	revision := page.Revision
	if err := transaction.Exec(`
INSERT INTO tauco_app.pages (
    id,
    key,
    published_revision_id,
    created_at,
    updated_at
) VALUES (?::uuid, ?, ?::uuid, ?, ?)`,
		string(revision.EntityID),
		string(page.Key),
		string(revision.RevisionID),
		revision.PublishedAt,
		revision.PublishedAt,
	).Error; err != nil {
		return fmt.Errorf("insert page %q identity: %w", page.Key, err)
	}
	if err := transaction.Exec(`
INSERT INTO tauco_app.page_revisions (
    id,
    page_id,
    revision_number,
    status,
    schema_version,
    content_json,
    content_checksum,
    created_by,
    created_at,
    published_at
) VALUES (?::uuid, ?::uuid, ?, ?, ?, ?::jsonb, ?, NULL, ?, ?)`,
		string(revision.RevisionID),
		string(revision.EntityID),
		revision.RevisionNumber,
		string(revision.Status),
		revision.SchemaVersion,
		string(revision.ContentJSON),
		string(revision.Checksum),
		revision.PublishedAt,
		revision.PublishedAt,
	).Error; err != nil {
		return fmt.Errorf("insert page %q immutable revision: %w", page.Key, err)
	}
	return nil
}

func insertProductSeed(
	transaction *gorm.DB,
	product contentapp.ProductSeed,
) error {
	revision := product.Revision
	if err := transaction.Exec(`
INSERT INTO tauco_app.products (
    id,
    slug,
    sku,
    sort_order,
    published_revision_id,
    first_published_at,
    created_at,
    updated_at
) VALUES (?::uuid, ?, NULL, ?, ?::uuid, ?, ?, ?)`,
		string(revision.EntityID),
		product.Slug,
		product.SortOrder,
		string(revision.RevisionID),
		revision.PublishedAt,
		revision.PublishedAt,
		revision.PublishedAt,
	).Error; err != nil {
		return fmt.Errorf("insert product %q identity: %w", product.Slug, err)
	}
	if err := transaction.Exec(`
INSERT INTO tauco_app.product_revisions (
    id,
    product_id,
    revision_number,
    status,
    schema_version,
    content_json,
    content_checksum,
    created_by,
    created_at,
    published_at
) VALUES (?::uuid, ?::uuid, ?, ?, ?, ?::jsonb, ?, NULL, ?, ?)`,
		string(revision.RevisionID),
		string(revision.EntityID),
		revision.RevisionNumber,
		string(revision.Status),
		revision.SchemaVersion,
		string(revision.ContentJSON),
		string(revision.Checksum),
		revision.PublishedAt,
		revision.PublishedAt,
	).Error; err != nil {
		return fmt.Errorf(
			"insert product %q immutable revision: %w",
			product.Slug,
			err,
		)
	}
	return nil
}

func indexSeedPlan(
	plan contentapp.SeedPlan,
) (map[string]contentapp.PageSeed, map[string]contentapp.ProductSeed) {
	pages := make(map[string]contentapp.PageSeed, len(plan.Pages))
	products := make(map[string]contentapp.ProductSeed, len(plan.Products))
	for _, page := range plan.Pages {
		pages[string(page.Key)] = page
	}
	for _, product := range plan.Products {
		products[product.Slug] = product
	}
	return pages, products
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func seedConflictError(format string, arguments ...any) error {
	return fmt.Errorf(
		"%w: %s",
		contentapp.ErrSeedConflict,
		fmt.Sprintf(format, arguments...),
	)
}

func normalizeSeedPersistenceError(err error) error {
	if err == nil || errors.Is(err, contentapp.ErrSeedConflict) {
		return err
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) ||
		errors.Is(err, gorm.ErrForeignKeyViolated) ||
		errors.Is(err, gorm.ErrCheckConstraintViolated) {
		return seedConflictError("PostgreSQL rejected conflicting seed data")
	}

	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) &&
		strings.HasPrefix(postgresError.Code, "23") {
		if postgresError.ConstraintName != "" {
			return seedConflictError(
				"PostgreSQL constraint %q rejected seed data",
				postgresError.ConstraintName,
			)
		}
		return seedConflictError(
			"PostgreSQL integrity rule %s rejected seed data",
			postgresError.Code,
		)
	}
	return err
}
