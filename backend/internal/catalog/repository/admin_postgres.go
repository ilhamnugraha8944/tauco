package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"gorm.io/gorm"
)

type AdminPostgres struct{ database *gorm.DB }

func NewAdminPostgres(database *gorm.DB) (*AdminPostgres, error) {
	if database == nil {
		return nil, errors.New("product admin repository requires database")
	}
	return &AdminPostgres{database: database}, nil
}

func (repository *AdminPostgres) ListAdminProducts(ctx context.Context, after *catalogapp.ProductPaginationPosition, limit int) ([]catalogapp.AdminProduct, bool, error) {
	where, arguments := "", []any{}
	if after != nil {
		where = "WHERE (sort_order, id) > (?, ?::uuid)"
		arguments = append(arguments, after.SortOrder(), after.ProductID())
	}
	arguments = append(arguments, limit+1)
	var ids []string
	if err := repository.database.WithContext(ctx).Raw(`SELECT id::text FROM tauco_app.products `+where+` ORDER BY sort_order,id LIMIT ?`, arguments...).Scan(&ids).Error; err != nil {
		return nil, false, err
	}
	more := len(ids) > limit
	if more {
		ids = ids[:limit]
	}
	products := make([]catalogapp.AdminProduct, 0, len(ids))
	for _, id := range ids {
		product, err := repository.GetAdminProduct(ctx, id)
		if err != nil {
			return nil, false, err
		}
		products = append(products, product)
	}
	return products, more, nil
}

func (repository *AdminPostgres) GetAdminProduct(ctx context.Context, id string) (catalogapp.AdminProduct, error) {
	var product productIdentityRow
	if err := repository.database.WithContext(ctx).Raw(`SELECT id::text,slug,sku,sort_order,published_revision_id::text,first_published_at,archived_at,updated_at FROM tauco_app.products WHERE id=?`, id).Scan(&product).Error; err != nil {
		return catalogapp.AdminProduct{}, err
	}
	if product.ID == "" {
		return catalogapp.AdminProduct{}, catalogapp.ErrAdminProductNotFound
	}
	var revisions []productRevisionRow
	if err := repository.database.WithContext(ctx).Raw(`SELECT id::text,product_id::text AS owner_id,revision_number,status,schema_version,content_json AS content,content_checksum AS checksum,created_by::text,created_at,published_at FROM tauco_app.product_revisions WHERE product_id=? ORDER BY revision_number DESC LIMIT 100`, id).Scan(&revisions).Error; err != nil {
		return catalogapp.AdminProduct{}, err
	}
	return hydrateAdminProduct(product, revisions), nil
}

func (repository *AdminPostgres) CreateAdminProduct(ctx context.Context, slug string, sku *string, sortOrder int, actorID string) (catalogapp.AdminProduct, error) {
	id, _ := uuid.NewV7()
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO tauco_app.products(id,slug,sku,sort_order) VALUES (?,?,?,?)`, id, slug, sku, sortOrder).Error; err != nil {
			if uniqueViolation(err) {
				return catalogapp.ErrProductConflict
			}
			return err
		}
		return logProductEvent(tx, "product.created", id.String(), actorID, map[string]any{"slug": slug, "sortOrder": sortOrder})
	})
	if err != nil {
		return catalogapp.AdminProduct{}, err
	}
	return repository.GetAdminProduct(ctx, id.String())
}

func (repository *AdminPostgres) UpdateAdminProduct(ctx context.Context, id, expected string, slug, sku *string, sortOrder *int, actorID string) (catalogapp.AdminProduct, error) {
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, latest, _, err := lockProduct(tx, id)
		if err != nil {
			return err
		}
		if latest != expected {
			return catalogapp.ErrProductPrecondition
		}
		if slug != nil && product.FirstPublishedAt != nil && *slug != product.Slug {
			return catalogapp.ErrProductConflict
		}
		updates := map[string]any{}
		if slug != nil {
			updates["slug"] = *slug
		}
		if sku != nil {
			updates["sku"] = *sku
		}
		if sortOrder != nil {
			updates["sort_order"] = *sortOrder
		}
		if len(updates) == 0 {
			return catalogapp.ErrInvalidProduct
		}
		if err := tx.Table("tauco_app.products").Where("id = ?", id).Updates(updates).Error; err != nil {
			if uniqueViolation(err) {
				return catalogapp.ErrProductConflict
			}
			return err
		}
		return logProductEvent(tx, "product.identity_updated", id, actorID, map[string]any{"fields": sortedKeys(updates)})
	})
	if err != nil {
		return catalogapp.AdminProduct{}, err
	}
	return repository.GetAdminProduct(ctx, id)
}

func (repository *AdminPostgres) GetAdminProductRevision(ctx context.Context, productID, revisionID string) (contentapp.AdminRevision, error) {
	var row productRevisionRow
	if err := repository.database.WithContext(ctx).Raw(`SELECT id::text,product_id::text AS owner_id,revision_number,status,schema_version,content_json AS content,content_checksum AS checksum,created_by::text,created_at,published_at FROM tauco_app.product_revisions WHERE product_id=? AND id=?`, productID, revisionID).Scan(&row).Error; err != nil {
		return contentapp.AdminRevision{}, err
	}
	if row.ID == "" {
		return contentapp.AdminRevision{}, catalogapp.ErrProductRevisionNotFound
	}
	return row.revision(), nil
}

func (repository *AdminPostgres) CreateAdminProductDraft(ctx context.Context, productID, expected, actorID string, content json.RawMessage, checksum string, media []contentapp.MediaReference) (contentapp.AdminRevision, error) {
	var created contentapp.AdminRevision
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, latest, number, err := lockProduct(tx, productID)
		if err != nil {
			return err
		}
		if latest != expected {
			return catalogapp.ErrProductPrecondition
		}
		if product.ArchivedAt != nil {
			return catalogapp.ErrProductConflict
		}
		if err := validateMediaExist(tx, media); err != nil {
			return err
		}
		id, _ := uuid.NewV7()
		if err := tx.Exec(`INSERT INTO tauco_app.product_revisions(id,product_id,revision_number,status,schema_version,content_json,content_checksum,created_by) VALUES (?,?,?,'draft',1,?::jsonb,?,?::uuid)`, id, productID, number+1, string(content), checksum, actorID).Error; err != nil {
			return err
		}
		if err := insertProductMedia(tx, id.String(), media); err != nil {
			return err
		}
		if err := logProductEvent(tx, "product.draft_saved", productID, actorID, map[string]any{"revisionNumber": number + 1}); err != nil {
			return err
		}
		created, err = scanProductRevision(tx, productID, id.String())
		return err
	})
	return created, err
}

func (repository *AdminPostgres) PublishAdminProductRevision(ctx context.Context, productID, revisionID, expected, actorID string) (contentapp.AdminRevision, error) {
	var published contentapp.AdminRevision
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, latest, number, err := lockProduct(tx, productID)
		if err != nil {
			return err
		}
		if latest != expected {
			return catalogapp.ErrProductPrecondition
		}
		if product.ArchivedAt != nil {
			return catalogapp.ErrProductConflict
		}
		source, err := scanProductRevisionRow(tx, productID, revisionID)
		if err != nil {
			return err
		}
		var blocked int64
		if err := tx.Raw(`SELECT count(*) FROM tauco_app.product_revision_media link JOIN tauco_app.media_assets asset ON asset.id=link.media_asset_id WHERE link.product_revision_id=? AND asset.status<>'ready'`, revisionID).Scan(&blocked).Error; err != nil {
			return err
		}
		if blocked > 0 {
			return catalogapp.ErrProductMediaNotReady
		}
		id, _ := uuid.NewV7()
		if err := tx.Exec(`INSERT INTO tauco_app.product_revisions(id,product_id,revision_number,status,schema_version,content_json,content_checksum,created_by,published_at) VALUES (?,?,?,'published',?,?,?,?::uuid,transaction_timestamp())`, id, productID, number+1, source.SchemaVersion, string(source.Content), source.Checksum, actorID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO tauco_app.product_revision_media(product_revision_id,media_asset_id,field_path,position) SELECT ?,media_asset_id,field_path,position FROM tauco_app.product_revision_media WHERE product_revision_id=?`, id, revisionID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE tauco_app.products SET published_revision_id=?,first_published_at=COALESCE(first_published_at,transaction_timestamp()) WHERE id=?`, id, productID).Error; err != nil {
			return err
		}
		if err := enqueueProductInvalidation(tx, product.Slug, id.String(), "publish"); err != nil {
			return err
		}
		if err := logProductEvent(tx, "product.published", productID, actorID, map[string]any{"slug": product.Slug, "revisionNumber": number + 1}); err != nil {
			return err
		}
		published, err = scanProductRevision(tx, productID, id.String())
		return err
	})
	return published, err
}

func (repository *AdminPostgres) UnpublishAdminProduct(ctx context.Context, id, expected, actorID string) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, latest, _, err := lockProduct(tx, id)
		if err != nil {
			return err
		}
		if latest != expected {
			return catalogapp.ErrProductPrecondition
		}
		if err := tx.Exec(`UPDATE tauco_app.products SET published_revision_id=NULL WHERE id=?`, id).Error; err != nil {
			return err
		}
		if err := enqueueProductInvalidation(tx, product.Slug, latest, "unpublish"); err != nil {
			return err
		}
		return logProductEvent(tx, "product.unpublished", id, actorID, map[string]any{"slug": product.Slug})
	})
}

func (repository *AdminPostgres) ArchiveAdminProduct(ctx context.Context, id, expected, actorID string) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, latest, _, err := lockProduct(tx, id)
		if err != nil {
			return err
		}
		if latest != expected {
			return catalogapp.ErrProductPrecondition
		}
		if product.PublishedRevisionID != nil || product.ArchivedAt != nil {
			return catalogapp.ErrProductConflict
		}
		if err := tx.Exec(`UPDATE tauco_app.products SET archived_at=transaction_timestamp() WHERE id=?`, id).Error; err != nil {
			return err
		}
		return logProductEvent(tx, "product.archived", id, actorID, map[string]any{"slug": product.Slug})
	})
}

func (repository *AdminPostgres) UnarchiveAdminProduct(ctx context.Context, id, expected, actorID string) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, latest, _, err := lockProduct(tx, id)
		if err != nil {
			return err
		}
		if latest != expected {
			return catalogapp.ErrProductPrecondition
		}
		if product.ArchivedAt == nil {
			return catalogapp.ErrProductConflict
		}
		if err := tx.Exec(`UPDATE tauco_app.products SET archived_at=NULL WHERE id=?`, id).Error; err != nil {
			return err
		}
		return logProductEvent(tx, "product.unarchived", id, actorID, map[string]any{"slug": product.Slug})
	})
}

type productIdentityRow struct {
	ID, Slug                     string
	SKU                          *string
	SortOrder                    int
	PublishedRevisionID          *string
	FirstPublishedAt, ArchivedAt *time.Time
	UpdatedAt                    time.Time
}
type productRevisionRow struct {
	ID, OwnerID, Status   string
	Number, SchemaVersion int
	Content               []byte
	Checksum              string
	CreatedBy             *string
	CreatedAt             time.Time
	PublishedAt           *time.Time
}

func (row productRevisionRow) revision() contentapp.AdminRevision {
	return contentapp.AdminRevision{ID: row.ID, OwnerID: row.OwnerID, Status: row.Status, Number: row.Number, SchemaVersion: row.SchemaVersion, Content: append([]byte(nil), row.Content...), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, PublishedAt: row.PublishedAt}
}
func (row productRevisionRow) summary() contentapp.AdminRevisionSummary {
	return contentapp.AdminRevisionSummary{ID: row.ID, Status: row.Status, Number: row.Number, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, PublishedAt: row.PublishedAt}
}
func hydrateAdminProduct(identity productIdentityRow, rows []productRevisionRow) catalogapp.AdminProduct {
	result := catalogapp.AdminProduct{ID: identity.ID, Slug: identity.Slug, SKU: identity.SKU, SortOrder: identity.SortOrder, PublishedRevisionID: identity.PublishedRevisionID, FirstPublishedAt: identity.FirstPublishedAt, ArchivedAt: identity.ArchivedAt, UpdatedAt: identity.UpdatedAt, Revisions: make([]contentapp.AdminRevisionSummary, 0, len(rows))}
	for _, row := range rows {
		result.Revisions = append(result.Revisions, row.summary())
	}
	return result
}

func lockProduct(tx *gorm.DB, id string) (productIdentityRow, string, int, error) {
	var product productIdentityRow
	if err := tx.Raw(`SELECT id::text,slug,sku,sort_order,published_revision_id::text,first_published_at,archived_at,updated_at FROM tauco_app.products WHERE id=? FOR UPDATE`, id).Scan(&product).Error; err != nil {
		return product, "", 0, err
	}
	if product.ID == "" {
		return product, "", 0, catalogapp.ErrAdminProductNotFound
	}
	var latest struct {
		ID     string
		Number int
	}
	if err := tx.Raw(`SELECT id::text,revision_number AS number FROM tauco_app.product_revisions WHERE product_id=? ORDER BY revision_number DESC LIMIT 1`, id).Scan(&latest).Error; err != nil {
		return product, "", 0, err
	}
	if latest.ID == "" {
		return product, product.ID, 0, nil
	}
	return product, latest.ID, latest.Number, nil
}
func scanProductRevision(tx *gorm.DB, owner, id string) (contentapp.AdminRevision, error) {
	row, err := scanProductRevisionRow(tx, owner, id)
	return row.revision(), err
}
func scanProductRevisionRow(tx *gorm.DB, owner, id string) (productRevisionRow, error) {
	var row productRevisionRow
	err := tx.Raw(`SELECT id::text,product_id::text AS owner_id,revision_number,status,schema_version,content_json AS content,content_checksum AS checksum,created_by::text,created_at,published_at FROM tauco_app.product_revisions WHERE product_id=? AND id=?`, owner, id).Scan(&row).Error
	if err != nil {
		return row, err
	}
	if row.ID == "" {
		return row, catalogapp.ErrProductRevisionNotFound
	}
	return row, nil
}
func validateMediaExist(tx *gorm.DB, refs []contentapp.MediaReference) error {
	for _, ref := range refs {
		var count int64
		if err := tx.Raw(`SELECT count(*) FROM tauco_app.media_assets WHERE id=?::uuid`, ref.AssetID).Scan(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return catalogapp.ErrInvalidProduct
		}
	}
	return nil
}
func insertProductMedia(tx *gorm.DB, revisionID string, refs []contentapp.MediaReference) error {
	for _, ref := range refs {
		if err := tx.Exec(`INSERT INTO tauco_app.product_revision_media(product_revision_id,media_asset_id,field_path,position) VALUES (?,?::uuid,?,?)`, revisionID, ref.AssetID, ref.FieldPath, ref.Position).Error; err != nil {
			return err
		}
	}
	return nil
}
func enqueueProductInvalidation(tx *gorm.DB, slug, revisionID, action string) error {
	id, _ := uuid.NewV7()
	payload, _ := json.Marshal(map[string]any{"generationTags": []string{"products", "product:" + slug}})
	return tx.Exec(`INSERT INTO tauco_app.background_jobs(id,kind,payload_json,idempotency_key) VALUES (?,'content.invalidate_cache',?::jsonb,?) ON CONFLICT(idempotency_key) DO NOTHING`, id, string(payload), "product.invalidate:"+action+":"+revisionID).Error
}
func logProductEvent(tx *gorm.DB, event, productID, actorID string, metadata map[string]any) error {
	id, _ := uuid.NewV7()
	payload, _ := json.Marshal(metadata)
	return tx.Exec(`INSERT INTO tauco_app.activity_logs(id,event_type,entity_type,entity_id,actor_type,actor_id,metadata_json) VALUES (?,?,'product',?,'admin',?::uuid,?::jsonb)`, id, event, productID, actorID, string(payload)).Error
}
func uniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "SQLSTATE 23505") || strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}
func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for _, key := range []string{"slug", "sku", "sort_order"} {
		if _, ok := values[key]; ok {
			keys = append(keys, key)
		}
	}
	return keys
}

var _ catalogapp.AdminProductRepository = (*AdminPostgres)(nil)
