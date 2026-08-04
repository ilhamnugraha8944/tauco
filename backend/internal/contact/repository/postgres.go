package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	"gorm.io/gorm"
)

type PostgresStore struct {
	database *gorm.DB
}

func NewPostgresStore(database *gorm.DB) (*PostgresStore, error) {
	if database == nil {
		return nil, errors.New("contact PostgreSQL store requires a database")
	}
	return &PostgresStore{database: database}, nil
}

func (store *PostgresStore) Create(
	ctx context.Context,
	record application.Record,
) (application.CreateResult, error) {
	var result application.CreateResult
	err := store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		messageID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate contact message ID: %w", err)
		}
		phone := any(nil)
		if record.Message.Phone != "" {
			phone = record.Message.Phone
		}
		insert := transaction.Exec(`
			INSERT INTO tauco_app.contact_messages (
				id, idempotency_key_hash, request_payload_hash, name, email,
				phone, subject, message, privacy_consent, privacy_notice_version,
				consent_at, retention_delete_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				LEAST(?::timestamptz, transaction_timestamp()),
				LEAST(?::timestamptz, transaction_timestamp()) + interval '12 months'
			)
			ON CONFLICT (idempotency_key_hash) DO NOTHING
		`,
			messageID, record.IdempotencyKeyHash, record.RequestPayloadHash,
			record.Message.Name, record.Message.Email, phone,
			string(record.Message.Subject), record.Message.Body,
			record.Message.PrivacyConsent, record.PrivacyNoticeVersion,
			record.ConsentAt, record.ConsentAt,
		)
		if insert.Error != nil {
			return fmt.Errorf("insert contact message: %w", insert.Error)
		}
		if insert.RowsAffected == 0 {
			var existing struct {
				PayloadHash string `gorm:"column:request_payload_hash"`
			}
			if err := transaction.Raw(`
				SELECT request_payload_hash
				FROM tauco_app.contact_messages
				WHERE idempotency_key_hash = ?
			`, record.IdempotencyKeyHash).Scan(&existing).Error; err != nil {
				return fmt.Errorf("read idempotent contact message: %w", err)
			}
			if existing.PayloadHash != record.RequestPayloadHash {
				return application.ErrIdempotencyConflict
			}
			result.Replayed = true
			return nil
		}

		for _, job := range []struct {
			kind string
			key  string
		}{
			{kind: "contact.email_notification", key: "contact.email:" + messageID.String()},
			{kind: "contact.activity_log", key: "contact.activity:" + messageID.String()},
		} {
			jobID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate background job ID: %w", err)
			}
			payload, _ := json.Marshal(map[string]string{
				"contactMessageId": messageID.String(),
			})
			if err := transaction.Exec(`
				INSERT INTO tauco_app.background_jobs (
					id, kind, payload_json, idempotency_key
				) VALUES (?, ?, ?::jsonb, ?)
			`, jobID, job.kind, string(payload), job.key).Error; err != nil {
				return fmt.Errorf("insert %s job: %w", job.kind, err)
			}
		}
		return nil
	})
	if err != nil {
		return application.CreateResult{}, err
	}
	return result, nil
}

// PurgeExpired removes contact PII whose approved retention deadline passed.
func (store *PostgresStore) PurgeExpired(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	if limit < 1 || limit > 1000 {
		return 0, errors.New("purge limit must be between 1 and 1000")
	}
	result := store.database.WithContext(ctx).Exec(`
		DELETE FROM tauco_app.contact_messages
		WHERE id IN (
			SELECT id FROM tauco_app.contact_messages
			WHERE retention_delete_at <= ?
			ORDER BY retention_delete_at, id
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
	`, before.UTC(), limit)
	if result.Error != nil {
		return 0, fmt.Errorf("purge expired contact messages: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (store *PostgresStore) LoadNotification(
	ctx context.Context,
	messageID string,
) (application.Notification, error) {
	var notification application.Notification
	if err := store.database.WithContext(ctx).Raw(`
		SELECT id::text AS message_id, name, email, subject, message
		FROM tauco_app.contact_messages
		WHERE id = ?
	`, messageID).Scan(&notification).Error; err != nil {
		return application.Notification{}, fmt.Errorf("load contact notification: %w", err)
	}
	if notification.MessageID == "" {
		return application.Notification{}, gorm.ErrRecordNotFound
	}
	return notification, nil
}

func (store *PostgresStore) RecordReceivedActivity(
	ctx context.Context,
	messageID string,
) error {
	result := store.database.WithContext(ctx).Exec(`
		INSERT INTO tauco_app.activity_logs (
			id, event_type, entity_type, entity_id, actor_type, metadata_json
		) VALUES (?, 'contact.received', 'contact_message', ?, 'visitor', '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, messageID, messageID)
	if result.Error != nil {
		return fmt.Errorf("insert contact activity: %w", result.Error)
	}
	return nil
}
