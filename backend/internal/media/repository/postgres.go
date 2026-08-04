package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
	"gorm.io/gorm"
)

type Postgres struct {
	database *gorm.DB
}

func (repository *Postgres) ListAdmin(ctx context.Context, after *catalogapp.ProductPaginationPosition, limit int) ([]mediaapp.AdminAsset, bool, error) {
	arguments := []any{}
	where := ""
	if after != nil {
		where = "WHERE (floor(extract(epoch from created_at) * 1000000)::bigint, id) < (?, ?::uuid)"
		arguments = append(arguments, after.SortOrder(), after.ProductID())
	}
	arguments = append(arguments, limit+1)
	var rows []struct {
		ID, Status, MIME, AltText string
		Width, Height             int
		Bytes                     int64
		Decorative                bool
		LastErrorCode             *string
		CreatedAt, UpdatedAt      time.Time
	}
	err := repository.database.WithContext(ctx).Raw(`
		SELECT id::text AS id, status, original_mime_type AS mime,
			original_width AS width, original_height AS height,
			original_bytes AS bytes, alt_text, decorative, last_error_code,
			created_at, updated_at
		FROM tauco_app.media_assets `+where+`
		ORDER BY created_at DESC, id DESC LIMIT ?`, arguments...).Scan(&rows).Error
	if err != nil {
		return nil, false, fmt.Errorf("list admin media: %w", err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	assets := make([]mediaapp.AdminAsset, 0, len(rows))
	for _, row := range rows {
		asset, loadErr := repository.GetAdmin(ctx, row.ID)
		if loadErr != nil {
			return nil, false, loadErr
		}
		assets = append(assets, asset)
	}
	return assets, hasMore, nil
}

func (repository *Postgres) GetAdmin(ctx context.Context, assetID string) (mediaapp.AdminAsset, error) {
	var row struct {
		ID, Status, MIME, AltText string
		Width, Height             int
		Bytes                     int64
		Decorative                bool
		LastErrorCode             *string
		CreatedAt, UpdatedAt      time.Time
	}
	err := repository.database.WithContext(ctx).Raw(`
		SELECT id::text AS id, status, original_mime_type AS mime,
			original_width AS width, original_height AS height,
			original_bytes AS bytes, alt_text, decorative, last_error_code,
			created_at, updated_at
		FROM tauco_app.media_assets WHERE id = ?`, assetID).Scan(&row).Error
	if err != nil {
		return mediaapp.AdminAsset{}, fmt.Errorf("get admin media: %w", err)
	}
	if row.ID == "" {
		return mediaapp.AdminAsset{}, mediaapp.ErrAssetNotFound
	}
	var variants []struct {
		Width, Height int
		ObjectKey     string
		Bytes         int64
		SHA256        string
	}
	if err := repository.database.WithContext(ctx).Raw(`
		SELECT width, height, object_key, bytes, sha256
		FROM tauco_app.media_variants WHERE media_asset_id = ?
		ORDER BY width ASC`, assetID).Scan(&variants).Error; err != nil {
		return mediaapp.AdminAsset{}, fmt.Errorf("get admin media variants: %w", err)
	}
	result := mediaapp.AdminAsset{ID: row.ID, Status: row.Status, MIME: row.MIME,
		Width: row.Width, Height: row.Height, Bytes: row.Bytes, AltText: row.AltText,
		Decorative: row.Decorative, LastErrorCode: row.LastErrorCode,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Variants: make([]mediaapp.AdminVariant, 0, len(variants))}
	for _, variant := range variants {
		result.Variants = append(result.Variants, mediaapp.AdminVariant(variant))
	}
	return result, nil
}

func (repository *Postgres) Retry(ctx context.Context, assetID, actorID string) error {
	return repository.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		result := transaction.Exec(`
			UPDATE tauco_app.media_assets SET status = 'processing', last_error_code = NULL
			WHERE id = ? AND status = 'failed'`, assetID)
		if result.Error != nil {
			return fmt.Errorf("retry media asset: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			var count int64
			if err := transaction.Raw(`SELECT count(*) FROM tauco_app.media_assets WHERE id = ?`, assetID).Scan(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return mediaapp.ErrAssetNotFound
			}
			return mediaapp.ErrRetryConflict
		}
		job := transaction.Exec(`
			UPDATE tauco_app.background_jobs
			SET status = 'retry', attempts = 0, run_at = now(), locked_at = NULL,
				lock_owner = NULL, lease_expires_at = NULL, last_error_code = NULL, last_error_message = NULL,
				completed_at = NULL, dead_at = NULL
			WHERE idempotency_key = ? AND status <> 'running'`, "media.variants:"+assetID)
		if job.Error != nil {
			return fmt.Errorf("retry media job: %w", job.Error)
		}
		if job.RowsAffected != 1 {
			return mediaapp.ErrRetryConflict
		}
		logID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		return transaction.Exec(`
			INSERT INTO tauco_app.activity_logs
			(id, event_type, entity_type, entity_id, actor_type, actor_id)
			VALUES (?, 'media.retry', 'media_asset', ?, 'admin', ?::uuid)`, logID, assetID, actorID).Error
	})
}

func (repository *Postgres) GetReadyVariant(ctx context.Context, assetID string, width *int) (mediaapp.AdminVariant, error) {
	var row struct {
		Width, Height int
		ObjectKey     string
		Bytes         int64
		SHA256        string
	}
	query := `SELECT variant.width, variant.height, variant.object_key, variant.bytes, variant.sha256
		FROM tauco_app.media_variants variant
		JOIN tauco_app.media_assets asset ON asset.id = variant.media_asset_id
		WHERE asset.id = ? AND asset.status = 'ready'`
	arguments := []any{assetID}
	if width != nil {
		query += " AND variant.width = ?"
		arguments = append(arguments, *width)
		query += " LIMIT 1"
	} else {
		query += " ORDER BY variant.width DESC LIMIT 1"
	}
	if err := repository.database.WithContext(ctx).Raw(query, arguments...).Scan(&row).Error; err != nil {
		return mediaapp.AdminVariant{}, fmt.Errorf("get public media variant: %w", err)
	}
	if row.ObjectKey == "" {
		return mediaapp.AdminVariant{}, mediaapp.ErrAssetNotFound
	}
	return mediaapp.AdminVariant(row), nil
}

func NewPostgres(database *gorm.DB) (*Postgres, error) {
	if database == nil {
		return nil, errors.New("media repository requires a database")
	}
	return &Postgres{database: database}, nil
}

func (repository *Postgres) CreateProcessing(ctx context.Context, asset domain.Asset) (string, bool, error) {
	if err := asset.Validate(); err != nil {
		return "", false, err
	}
	var assetID string
	replayed := false
	err := repository.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		identifier, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate media ID: %w", err)
		}
		insert := transaction.Exec(`
			INSERT INTO tauco_app.media_assets (
				id, status, original_object_key, original_mime_type,
				original_width, original_height, original_bytes, sha256,
				alt_text, decorative
			) VALUES (?, 'processing', ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (original_object_key) DO NOTHING
		`, identifier, asset.OriginalKey, asset.OriginalMIME,
			asset.OriginalWidth, asset.OriginalHeight, asset.OriginalBytes,
			asset.SHA256, asset.AltText, asset.Decorative)
		if insert.Error != nil {
			return fmt.Errorf("insert media asset: %w", insert.Error)
		}
		if insert.RowsAffected == 0 {
			var existing struct {
				ID     string `gorm:"column:id"`
				SHA256 string `gorm:"column:sha256"`
			}
			if err := transaction.Raw(`
				SELECT id::text AS id, sha256 FROM tauco_app.media_assets
				WHERE original_object_key = ?
			`, asset.OriginalKey).Scan(&existing).Error; err != nil {
				return fmt.Errorf("read existing media asset: %w", err)
			}
			if existing.ID == "" || existing.SHA256 != asset.SHA256 {
				return errors.New("media object key conflict")
			}
			assetID, replayed = existing.ID, true
			return nil
		}
		assetID = identifier.String()
		jobID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate media job ID: %w", err)
		}
		payload, _ := json.Marshal(map[string]string{"mediaAssetId": assetID})
		if err := transaction.Exec(`
			INSERT INTO tauco_app.background_jobs (
				id, kind, payload_json, idempotency_key
			) VALUES (?, 'media.generate_variants', ?::jsonb, ?)
		`, jobID, string(payload), "media.variants:"+assetID).Error; err != nil {
			return fmt.Errorf("insert media variant job: %w", err)
		}
		return nil
	})
	return assetID, replayed, err
}

func (repository *Postgres) Load(ctx context.Context, assetID string) (domain.Asset, error) {
	var row struct {
		ID, Status, OriginalKey, OriginalMIME, SHA256, AltText string
		OriginalWidth, OriginalHeight                          int
		OriginalBytes                                          int64
		Decorative                                             bool
	}
	err := repository.database.WithContext(ctx).Raw(`
		SELECT id::text AS id, status, original_object_key AS original_key,
			original_mime_type AS original_mime,
			original_width, original_height, original_bytes, sha256,
			alt_text, decorative
		FROM tauco_app.media_assets WHERE id = ?
	`, assetID).Scan(&row).Error
	if err != nil {
		return domain.Asset{}, fmt.Errorf("load media asset: %w", err)
	}
	if row.ID == "" {
		return domain.Asset{}, gorm.ErrRecordNotFound
	}
	return domain.Asset{
		ID: row.ID, Status: row.Status, OriginalKey: row.OriginalKey,
		OriginalMIME: row.OriginalMIME, OriginalWidth: row.OriginalWidth,
		OriginalHeight: row.OriginalHeight, OriginalBytes: row.OriginalBytes,
		SHA256: row.SHA256, AltText: row.AltText, Decorative: row.Decorative,
	}, nil
}

func (repository *Postgres) MarkReady(ctx context.Context, assetID string, variants []domain.Variant, skipped []int) error {
	return repository.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		for _, variant := range variants {
			identifier, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate media variant ID: %w", err)
			}
			if err := transaction.Exec(`
				INSERT INTO tauco_app.media_variants (
					id, media_asset_id, width, height, format, object_key,
					mime_type, bytes, sha256
				) VALUES (?, ?, ?, ?, 'webp', ?, 'image/webp', ?, ?)
				ON CONFLICT (media_asset_id, width, format) DO UPDATE
				SET height = EXCLUDED.height, object_key = EXCLUDED.object_key,
					mime_type = EXCLUDED.mime_type, bytes = EXCLUDED.bytes,
					sha256 = EXCLUDED.sha256
			`, identifier, assetID, variant.Width, variant.Height,
				variant.ObjectKey, variant.Bytes, variant.SHA256).Error; err != nil {
				return fmt.Errorf("upsert media variant: %w", err)
			}
		}
		result := transaction.Exec(`
			UPDATE tauco_app.media_assets
			SET status = 'ready', last_error_code = NULL
			WHERE id = ? AND status IN ('processing', 'failed', 'ready')
		`, assetID)
		if result.Error != nil {
			return fmt.Errorf("mark media ready: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		metadata, _ := json.Marshal(map[string]any{"skippedWidths": skipped})
		if err := transaction.Exec(`
			INSERT INTO tauco_app.activity_logs (
				id, event_type, entity_type, entity_id, actor_type, metadata_json
			) VALUES (?, 'media.ready', 'media_asset', ?, 'system', ?::jsonb)
			ON CONFLICT (id) DO NOTHING
		`, assetID, assetID, string(metadata)).Error; err != nil {
			return fmt.Errorf("insert media ready activity: %w", err)
		}
		return nil
	})
}

func (repository *Postgres) MarkFailed(ctx context.Context, assetID, errorCode string) error {
	result := repository.database.WithContext(ctx).Exec(`
		UPDATE tauco_app.media_assets
		SET status = 'failed', last_error_code = ?
		WHERE id = ? AND status <> 'ready'
	`, errorCode, assetID)
	if result.Error != nil {
		return fmt.Errorf("mark media failed: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ mediaapp.Repository = (*Postgres)(nil)
var _ mediaapp.AdminRepository = (*Postgres)(nil)
