package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
)

func TestNormalizeSeedPersistenceErrorMapsIntegrityViolations(t *testing.T) {
	t.Parallel()

	err := normalizeSeedPersistenceError(&pgconn.PgError{
		Code:           "23505",
		ConstraintName: "pages_key_unique",
	})
	if !errors.Is(err, contentapp.ErrSeedConflict) {
		t.Fatalf("expected seed conflict, got %v", err)
	}
}

func TestNormalizeSeedPersistenceErrorPreservesOperationalFailure(t *testing.T) {
	t.Parallel()

	operational := &pgconn.PgError{Code: "08006"}
	if got := normalizeSeedPersistenceError(operational); got != operational {
		t.Fatalf("expected operational error to remain unchanged, got %v", got)
	}
}

func TestSeedConflictErrorSupportsErrorsIs(t *testing.T) {
	t.Parallel()

	err := seedConflictError("fixture is inconsistent")
	if !errors.Is(err, contentapp.ErrSeedConflict) {
		t.Fatalf("expected errors.Is support, got %v", err)
	}
}
