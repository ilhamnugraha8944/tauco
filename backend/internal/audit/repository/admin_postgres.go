package repository

import (
	"context"
	"errors"

	auditapp "github.com/ilhamnugraha8944/tauco/backend/internal/audit/application"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	"gorm.io/gorm"
)

type AdminPostgres struct{ database *gorm.DB }

func NewAdminPostgres(database *gorm.DB) (*AdminPostgres, error) {
	if database == nil {
		return nil, errors.New("admin activity repository requires database")
	}
	return &AdminPostgres{database: database}, nil
}

func (repository *AdminPostgres) ListAdminActivities(ctx context.Context, after *catalogapp.ProductPaginationPosition, limit int, filter auditapp.ActivityFilter) ([]auditapp.Activity, bool, error) {
	// metadata_json is deliberately excluded: the API only exposes allowlisted columns.
	statement := `SELECT id::text,event_type,entity_type,entity_id::text,actor_type,actor_id::text,request_id,created_at FROM tauco_app.activity_logs WHERE 1=1`
	arguments := []any{}
	if filter.EventType != nil {
		statement += ` AND event_type=?`
		arguments = append(arguments, *filter.EventType)
	}
	if filter.EntityType != nil {
		statement += ` AND entity_type=?`
		arguments = append(arguments, *filter.EntityType)
	}
	if after != nil {
		statement += ` AND (created_at,id) < (to_timestamp(? / 1000000.0),?::uuid)`
		arguments = append(arguments, after.SortOrder(), after.ProductID())
	}
	statement += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	arguments = append(arguments, limit+1)
	var activities []auditapp.Activity
	if err := repository.database.WithContext(ctx).Raw(statement, arguments...).Scan(&activities).Error; err != nil {
		return nil, false, err
	}
	more := len(activities) > limit
	if more {
		activities = activities[:limit]
	}
	return activities, more, nil
}
