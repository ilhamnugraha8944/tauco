package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ilhamnugraha8944/tauco/backend/internal/jobs/domain"
	"gorm.io/gorm"
)

var ErrLeaseLost = errors.New("job lease lost")

type PostgresRepository struct {
	database *gorm.DB
}

func NewPostgresRepository(database *gorm.DB) (*PostgresRepository, error) {
	if database == nil {
		return nil, errors.New("job repository requires a database")
	}
	return &PostgresRepository{database: database}, nil
}

func (repository *PostgresRepository) Claim(
	ctx context.Context,
	owner string,
	limit int,
	lease time.Duration,
) ([]domain.Job, error) {
	if owner == "" || limit < 1 || limit > 100 || lease <= 0 {
		return nil, errors.New("invalid claim arguments")
	}
	type row struct {
		ID          string    `gorm:"column:id"`
		Kind        string    `gorm:"column:kind"`
		Payload     []byte    `gorm:"column:payload_json"`
		Attempts    int       `gorm:"column:attempts"`
		MaxAttempts int       `gorm:"column:max_attempts"`
		LockedAt    time.Time `gorm:"column:locked_at"`
		LeaseUntil  time.Time `gorm:"column:lease_expires_at"`
	}
	var rows []row
	err := repository.database.WithContext(ctx).Raw(`
		WITH candidates AS (
			SELECT id
			FROM tauco_app.background_jobs
			WHERE (
				(status IN ('pending', 'retry') AND run_at <= transaction_timestamp())
				OR (status = 'running' AND lease_expires_at <= transaction_timestamp())
			)
			AND attempts < max_attempts
			ORDER BY priority DESC, run_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT ?
		)
		UPDATE tauco_app.background_jobs AS job
		SET status = 'running',
			attempts = job.attempts + 1,
			locked_at = transaction_timestamp(),
			lock_owner = ?,
			lease_expires_at = transaction_timestamp() + (? * interval '1 millisecond'),
			last_error_code = NULL,
			last_error_message = NULL,
			updated_at = transaction_timestamp(),
			completed_at = NULL,
			dead_at = NULL
		FROM candidates
		WHERE job.id = candidates.id
		RETURNING job.id, job.kind, job.payload_json, job.attempts,
			job.max_attempts, job.locked_at, job.lease_expires_at
	`, limit, owner, lease.Milliseconds()).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("claim PostgreSQL jobs: %w", err)
	}
	jobs := make([]domain.Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, domain.Job{
			ID: row.ID, Kind: row.Kind, Payload: append([]byte(nil), row.Payload...),
			Attempts: row.Attempts, MaxAttempts: row.MaxAttempts,
			LockedAt: row.LockedAt, LeaseUntil: row.LeaseUntil,
		})
	}
	return jobs, nil
}

func (repository *PostgresRepository) Heartbeat(
	ctx context.Context,
	jobID, owner string,
	lease time.Duration,
) error {
	result := repository.database.WithContext(ctx).Exec(`
		UPDATE tauco_app.background_jobs
		SET lease_expires_at = transaction_timestamp() + (? * interval '1 millisecond'),
			updated_at = transaction_timestamp()
		WHERE id = ? AND status = 'running' AND lock_owner = ?
	`, lease.Milliseconds(), jobID, owner)
	return affected(result, "heartbeat job")
}

func (repository *PostgresRepository) Succeed(ctx context.Context, jobID, owner string) error {
	result := repository.database.WithContext(ctx).Exec(`
		UPDATE tauco_app.background_jobs
		SET status = 'succeeded', locked_at = NULL, lock_owner = NULL,
			lease_expires_at = NULL, completed_at = transaction_timestamp(),
			dead_at = NULL, updated_at = transaction_timestamp()
		WHERE id = ? AND status = 'running' AND lock_owner = ?
	`, jobID, owner)
	return affected(result, "complete job")
}

func (repository *PostgresRepository) Fail(
	ctx context.Context,
	jobID, owner string,
	retryAt time.Time,
	errorCode string,
) error {
	result := repository.database.WithContext(ctx).Exec(`
		UPDATE tauco_app.background_jobs
		SET status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'retry' END,
			run_at = CASE WHEN attempts >= max_attempts THEN run_at ELSE ? END,
			locked_at = NULL, lock_owner = NULL, lease_expires_at = NULL,
			last_error_code = ?, last_error_message = NULL,
			completed_at = NULL,
			dead_at = CASE WHEN attempts >= max_attempts THEN transaction_timestamp() ELSE NULL END,
			updated_at = transaction_timestamp()
		WHERE id = ? AND status = 'running' AND lock_owner = ?
	`, retryAt.UTC(), errorCode, jobID, owner)
	return affected(result, "fail job")
}

func (repository *PostgresRepository) Release(ctx context.Context, jobID, owner string) error {
	result := repository.database.WithContext(ctx).Exec(`
		UPDATE tauco_app.background_jobs
		SET status = 'retry', run_at = transaction_timestamp(),
			locked_at = NULL, lock_owner = NULL, lease_expires_at = NULL,
			updated_at = transaction_timestamp()
		WHERE id = ? AND status = 'running' AND lock_owner = ?
	`, jobID, owner)
	return affected(result, "release job")
}

func (repository *PostgresRepository) Replay(ctx context.Context, jobID, requestID string) error {
	return repository.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		result := transaction.Exec(`
			UPDATE tauco_app.background_jobs
			SET status = 'retry', attempts = 0, run_at = transaction_timestamp(),
				last_error_code = NULL, last_error_message = NULL, dead_at = NULL,
				updated_at = transaction_timestamp()
			WHERE id = ? AND status = 'dead'
		`, jobID)
		if err := affected(result, "replay job"); err != nil {
			return err
		}
		activityID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate replay activity ID: %w", err)
		}
		if err := transaction.Exec(`
			INSERT INTO tauco_app.activity_logs (
				id, event_type, entity_type, entity_id, actor_type,
				metadata_json, request_id
			) VALUES (?, 'background_job.replayed', 'background_job', ?, 'system', '{}'::jsonb, ?)
		`, activityID, jobID, requestID).Error; err != nil {
			return fmt.Errorf("insert replay activity: %w", err)
		}
		return nil
	})
}

func affected(result *gorm.DB, operation string) error {
	if result.Error != nil {
		return fmt.Errorf("%s: %w", operation, result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrLeaseLost
	}
	return nil
}
