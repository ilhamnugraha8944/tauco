package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"gorm.io/gorm"
)

type AdminPostgres struct{ database *gorm.DB }

func NewAdminPostgres(database *gorm.DB) (*AdminPostgres, error) {
	if database == nil {
		return nil, errors.New("content admin repository requires database")
	}
	return &AdminPostgres{database: database}, nil
}

func (repository *AdminPostgres) GetAdminPage(ctx context.Context, key string) (contentapp.AdminPage, error) {
	var page struct {
		ID, Key             string
		PublishedRevisionID *string
		UpdatedAt           time.Time
	}
	if err := repository.database.WithContext(ctx).Raw(`SELECT id::text, key, published_revision_id::text, updated_at FROM tauco_app.pages WHERE key = ?`, key).Scan(&page).Error; err != nil {
		return contentapp.AdminPage{}, err
	}
	if page.ID == "" {
		return contentapp.AdminPage{}, contentapp.ErrAdminPageNotFound
	}
	var rows []revisionRow
	if err := repository.database.WithContext(ctx).Raw(`SELECT id::text, page_id::text AS owner_id, revision_number, status, schema_version, content_json AS content, created_by::text, created_at, published_at FROM tauco_app.page_revisions WHERE page_id = ? ORDER BY revision_number DESC LIMIT 100`, page.ID).Scan(&rows).Error; err != nil {
		return contentapp.AdminPage{}, err
	}
	if len(rows) == 0 {
		return contentapp.AdminPage{}, contentapp.ErrRevisionNotFound
	}
	result := contentapp.AdminPage{ID: page.ID, Key: page.Key, PublishedRevisionID: page.PublishedRevisionID, UpdatedAt: page.UpdatedAt, Latest: rows[0].revision(), Revisions: make([]contentapp.AdminRevisionSummary, 0, len(rows))}
	for _, row := range rows {
		result.Revisions = append(result.Revisions, row.summary())
	}
	return result, nil
}

func (repository *AdminPostgres) GetAdminRevision(ctx context.Context, key, revisionID string) (contentapp.AdminRevision, error) {
	var row revisionRow
	if err := repository.database.WithContext(ctx).Raw(`SELECT revision.id::text, revision.page_id::text AS owner_id, revision.revision_number, revision.status, revision.schema_version, revision.content_json AS content, revision.created_by::text, revision.created_at, revision.published_at FROM tauco_app.page_revisions revision JOIN tauco_app.pages page ON page.id = revision.page_id WHERE page.key = ? AND revision.id = ?`, key, revisionID).Scan(&row).Error; err != nil {
		return contentapp.AdminRevision{}, err
	}
	if row.ID == "" {
		return contentapp.AdminRevision{}, contentapp.ErrRevisionNotFound
	}
	return row.revision(), nil
}

func (repository *AdminPostgres) CreateDraft(ctx context.Context, key, baseID, actorID string, content json.RawMessage, checksum string, media []contentapp.MediaReference) (contentapp.AdminRevision, error) {
	var created contentapp.AdminRevision
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pageID, latestID, number, err := lockPage(tx, key)
		if err != nil {
			return err
		}
		if latestID != baseID {
			return contentapp.ErrPrecondition
		}
		for _, reference := range media {
			var count int64
			if err := tx.Raw(`SELECT count(*) FROM tauco_app.media_assets WHERE id = ?::uuid`, reference.AssetID).Scan(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return contentapp.ErrInvalidPage
			}
		}
		id, _ := uuid.NewV7()
		if err := tx.Exec(`INSERT INTO tauco_app.page_revisions (id, page_id, revision_number, status, schema_version, content_json, content_checksum, created_by) VALUES (?, ?, ?, 'draft', 1, ?::jsonb, ?, ?::uuid)`, id, pageID, number+1, string(content), checksum, actorID).Error; err != nil {
			return fmt.Errorf("insert page draft: %w", err)
		}
		if err := insertPageMedia(tx, id.String(), media); err != nil {
			return err
		}
		logID, _ := uuid.NewV7()
		if err := tx.Exec(`INSERT INTO tauco_app.activity_logs (id,event_type,entity_type,entity_id,actor_type,actor_id,metadata_json) VALUES (?, 'content.draft_saved','page',?,'admin',?::uuid,jsonb_build_object('key',?,'revisionNumber',?))`, logID, pageID, actorID, key, number+1).Error; err != nil {
			return err
		}
		created, err = scanRevision(tx, id.String())
		return err
	})
	return created, err
}

func (repository *AdminPostgres) PublishRevision(ctx context.Context, key, revisionID, expectedID, actorID string) (contentapp.AdminRevision, error) {
	var published contentapp.AdminRevision
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pageID, latestID, number, err := lockPage(tx, key)
		if err != nil {
			return err
		}
		if latestID != expectedID {
			return contentapp.ErrPrecondition
		}
		source, err := scanRevisionForOwner(tx, pageID, revisionID)
		if err != nil {
			return err
		}
		var blocked int64
		if err := tx.Raw(`SELECT count(*) FROM tauco_app.page_revision_media link JOIN tauco_app.media_assets asset ON asset.id=link.media_asset_id WHERE link.page_revision_id=? AND asset.status<>'ready'`, revisionID).Scan(&blocked).Error; err != nil {
			return err
		}
		if blocked > 0 {
			return contentapp.ErrMediaNotReady
		}
		id, _ := uuid.NewV7()
		if err := tx.Exec(`INSERT INTO tauco_app.page_revisions (id,page_id,revision_number,status,schema_version,content_json,content_checksum,created_by,published_at) VALUES (?,?,?,'published',?,?,?,?::uuid,transaction_timestamp())`, id, pageID, number+1, source.SchemaVersion, string(source.Content), source.Checksum, actorID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO tauco_app.page_revision_media (page_revision_id,media_asset_id,field_path,position) SELECT ?,media_asset_id,field_path,position FROM tauco_app.page_revision_media WHERE page_revision_id=?`, id, revisionID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE tauco_app.pages SET published_revision_id=? WHERE id=?`, id, pageID).Error; err != nil {
			return err
		}
		if err := enqueueInvalidation(tx, key, id.String(), "publish"); err != nil {
			return err
		}
		if err := logContentEvent(tx, "content.published", pageID, actorID, key, number+1); err != nil {
			return err
		}
		published, err = scanRevision(tx, id.String())
		return err
	})
	return published, err
}

func (repository *AdminPostgres) Unpublish(ctx context.Context, key, expectedID, actorID string) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pageID, latestID, number, err := lockPage(tx, key)
		if err != nil {
			return err
		}
		if latestID != expectedID {
			return contentapp.ErrPrecondition
		}
		if err := tx.Exec(`UPDATE tauco_app.pages SET published_revision_id=NULL WHERE id=?`, pageID).Error; err != nil {
			return err
		}
		if err := enqueueInvalidation(tx, key, latestID, "unpublish"); err != nil {
			return err
		}
		return logContentEvent(tx, "content.unpublished", pageID, actorID, key, number)
	})
}

type revisionRow struct {
	ID, OwnerID, Status   string
	Number, SchemaVersion int
	Content               []byte
	Checksum              string
	CreatedBy             *string
	CreatedAt             time.Time
	PublishedAt           *time.Time
}

func (row revisionRow) revision() contentapp.AdminRevision {
	return contentapp.AdminRevision{ID: row.ID, OwnerID: row.OwnerID, Status: row.Status, Number: row.Number, SchemaVersion: row.SchemaVersion, Content: append([]byte(nil), row.Content...), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, PublishedAt: row.PublishedAt}
}
func (row revisionRow) summary() contentapp.AdminRevisionSummary {
	return contentapp.AdminRevisionSummary{ID: row.ID, Status: row.Status, Number: row.Number, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, PublishedAt: row.PublishedAt}
}

func lockPage(tx *gorm.DB, key string) (string, string, int, error) {
	var pageID string
	if err := tx.Raw(`SELECT id::text FROM tauco_app.pages WHERE key=? FOR UPDATE`, key).Scan(&pageID).Error; err != nil {
		return "", "", 0, err
	}
	if pageID == "" {
		return "", "", 0, contentapp.ErrAdminPageNotFound
	}
	var latest struct {
		ID     string
		Number int
	}
	if err := tx.Raw(`SELECT id::text, revision_number AS number FROM tauco_app.page_revisions WHERE page_id=? ORDER BY revision_number DESC LIMIT 1`, pageID).Scan(&latest).Error; err != nil {
		return "", "", 0, err
	}
	if latest.ID == "" {
		return "", "", 0, contentapp.ErrRevisionNotFound
	}
	return pageID, latest.ID, latest.Number, nil
}

func scanRevision(tx *gorm.DB, id string) (contentapp.AdminRevision, error) {
	var row revisionRow
	err := tx.Raw(`SELECT id::text,page_id::text AS owner_id,revision_number,status,schema_version,content_json AS content,created_by::text,created_at,published_at FROM tauco_app.page_revisions WHERE id=?`, id).Scan(&row).Error
	if err != nil {
		return contentapp.AdminRevision{}, err
	}
	return row.revision(), nil
}
func scanRevisionForOwner(tx *gorm.DB, owner, id string) (revisionRow, error) {
	var row revisionRow
	err := tx.Raw(`SELECT id::text,page_id::text AS owner_id,revision_number,status,schema_version,content_json AS content,content_checksum AS checksum,created_by::text,created_at,published_at FROM tauco_app.page_revisions WHERE id=? AND page_id=?`, id, owner).Scan(&row).Error
	if err != nil {
		return row, err
	}
	if row.ID == "" {
		return row, contentapp.ErrRevisionNotFound
	}
	return row, nil
}

func insertPageMedia(tx *gorm.DB, revisionID string, references []contentapp.MediaReference) error {
	for _, ref := range references {
		if err := tx.Exec(`INSERT INTO tauco_app.page_revision_media(page_revision_id,media_asset_id,field_path,position) VALUES (?,?::uuid,?,?)`, revisionID, ref.AssetID, ref.FieldPath, ref.Position).Error; err != nil {
			return fmt.Errorf("link page media: %w", err)
		}
	}
	return nil
}
func enqueueInvalidation(tx *gorm.DB, key, revisionID, action string) error {
	id, _ := uuid.NewV7()
	payload, _ := json.Marshal(map[string]string{"generationTag": key})
	return tx.Exec(`INSERT INTO tauco_app.background_jobs(id,kind,payload_json,idempotency_key) VALUES (?,'content.invalidate_cache',?::jsonb,?) ON CONFLICT(idempotency_key) DO NOTHING`, id, string(payload), "content.invalidate:"+action+":"+revisionID).Error
}
func logContentEvent(tx *gorm.DB, event, pageID, actorID, key string, number int) error {
	id, _ := uuid.NewV7()
	return tx.Exec(`INSERT INTO tauco_app.activity_logs(id,event_type,entity_type,entity_id,actor_type,actor_id,metadata_json) VALUES (?,?, 'page',?,'admin',?::uuid,jsonb_build_object('key',?,'revisionNumber',?))`, id, event, pageID, actorID, key, number).Error
}

var _ contentapp.AdminRepository = (*AdminPostgres)(nil)
