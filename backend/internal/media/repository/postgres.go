package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
	"gorm.io/gorm"
)

type Postgres struct {
	database *gorm.DB
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
