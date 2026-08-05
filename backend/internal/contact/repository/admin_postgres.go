package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	"gorm.io/gorm"
)

func (store *PostgresStore) ListAdminMessages(ctx context.Context, after *catalogapp.ProductPaginationPosition, limit int, status *string) ([]contactapp.AdminMessage, bool, error) {
	statement := `SELECT id::text,name,email,phone,subject,message,status,created_at,updated_at FROM tauco_app.contact_messages WHERE 1=1`
	arguments := []any{}
	if status != nil {
		statement += ` AND status=?`
		arguments = append(arguments, *status)
	}
	if after != nil {
		statement += ` AND (created_at,id) < (to_timestamp(? / 1000000.0),?::uuid)`
		arguments = append(arguments, after.SortOrder(), after.ProductID())
	}
	statement += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	arguments = append(arguments, limit+1)
	var messages []contactapp.AdminMessage
	if err := store.database.WithContext(ctx).Raw(statement, arguments...).Scan(&messages).Error; err != nil {
		return nil, false, err
	}
	more := len(messages) > limit
	if more {
		messages = messages[:limit]
	}
	return messages, more, nil
}

func (store *PostgresStore) GetAdminMessage(ctx context.Context, id string) (contactapp.AdminMessage, error) {
	var message contactapp.AdminMessage
	if err := store.database.WithContext(ctx).Raw(`SELECT id::text,name,email,phone,subject,message,status,created_at,updated_at FROM tauco_app.contact_messages WHERE id=?`, id).Scan(&message).Error; err != nil {
		return contactapp.AdminMessage{}, err
	}
	if message.ID == "" {
		return contactapp.AdminMessage{}, contactapp.ErrAdminMessageNotFound
	}
	return message, nil
}

func (store *PostgresStore) UpdateAdminMessageStatus(ctx context.Context, id string, expected time.Time, status, actorID string) (contactapp.AdminMessage, error) {
	err := store.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current contactapp.AdminMessage
		if err := tx.Raw(`SELECT id::text,status,updated_at FROM tauco_app.contact_messages WHERE id=? FOR UPDATE`, id).Scan(&current).Error; err != nil {
			return err
		}
		if current.ID == "" {
			return contactapp.ErrAdminMessageNotFound
		}
		if !current.UpdatedAt.Equal(expected) {
			return contactapp.ErrAdminMessageConflict
		}
		if current.Status == status {
			return nil
		}
		if err := tx.Exec(`UPDATE tauco_app.contact_messages SET status=? WHERE id=?`, status, id).Error; err != nil {
			return err
		}
		logID, _ := uuid.NewV7()
		return tx.Exec(`INSERT INTO tauco_app.activity_logs(id,event_type,entity_type,entity_id,actor_type,actor_id,metadata_json) VALUES (?,'contact.status_changed','contact_message',?,'admin',?::uuid,jsonb_build_object('fromStatus',?,'toStatus',?))`, logID, id, actorID, current.Status, status).Error
	})
	if err != nil {
		return contactapp.AdminMessage{}, err
	}
	return store.GetAdminMessage(ctx, id)
}
