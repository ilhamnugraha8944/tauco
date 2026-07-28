package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/ilhamnugraha8944/tauco/backend/migrations"
)

// Migrator applies the embedded, immutable SQL migration set.
type Migrator struct {
	inner *migrate.Migrate
}

// NewMigrator constructs a migrator whose metadata table lives in the private
// application schema. BootstrapRoles or equivalent remote provisioning must
// have created that schema first.
func NewMigrator(databaseURL string) (*Migrator, error) {
	source, err := iofs.New(migrations.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}

	targetURL, err := migrationDatabaseURL(databaseURL)
	if err != nil {
		_ = source.Close()
		return nil, err
	}

	inner, err := migrate.NewWithSourceInstance("iofs", source, targetURL)
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("initialize database migrator: %w", err)
	}
	return &Migrator{inner: inner}, nil
}

// Up applies every pending migration. An already-current database succeeds.
func (m *Migrator) Up() error {
	if m == nil || m.inner == nil {
		return errors.New("database migrator is not initialized")
	}
	return normalizeNoChange(m.inner.Up())
}

// DownOne reverses exactly one version. It never performs an implicit full
// rollback.
func (m *Migrator) DownOne() error {
	if m == nil || m.inner == nil {
		return errors.New("database migrator is not initialized")
	}
	return normalizeNoChange(m.inner.Steps(-1))
}

// DownAll reverses all versions. Callers must require explicit human intent.
func (m *Migrator) DownAll() error {
	if m == nil || m.inner == nil {
		return errors.New("database migrator is not initialized")
	}
	return normalizeNoChange(m.inner.Down())
}

// Steps moves an explicit number of versions.
func (m *Migrator) Steps(count int) error {
	if m == nil || m.inner == nil {
		return errors.New("database migrator is not initialized")
	}
	if count == 0 {
		return errors.New("migration step count must not be zero")
	}
	return normalizeNoChange(m.inner.Steps(count))
}

// Version returns the current version and dirty state.
func (m *Migrator) Version() (version uint, dirty bool, err error) {
	if m == nil || m.inner == nil {
		return 0, false, errors.New("database migrator is not initialized")
	}
	version, dirty, err = m.inner.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return version, dirty, err
}

// Close releases both migration source and PostgreSQL resources.
func (m *Migrator) Close() error {
	if m == nil || m.inner == nil {
		return nil
	}
	sourceErr, databaseErr := m.inner.Close()
	m.inner = nil
	return errors.Join(sourceErr, databaseErr)
}

func normalizeNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
