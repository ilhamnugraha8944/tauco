package database

import (
	"strings"
	"testing"
)

func TestLoadMigrationConfig(t *testing.T) {
	t.Parallel()

	lookup := mapLookup(map[string]string{
		"MIGRATION_DATABASE_URL":    "postgres://tauco_owner:owner-secret@127.0.0.1:5432/tauco?sslmode=disable",
		"DATABASE_URL":              "postgres://tauco_app_runtime:runtime-secret-123@127.0.0.1:5432/tauco?sslmode=disable",
		"MIGRATION_BOOTSTRAP_ROLES": "true",
	})
	cfg, err := LoadMigrationConfig(lookup)
	if err != nil {
		t.Fatalf("LoadMigrationConfig() error = %v", err)
	}
	if !cfg.BootstrapRoles {
		t.Fatal("BootstrapRoles = false, want true")
	}
}

func TestMigrationConfigRejectsBootstrapHostMismatch(t *testing.T) {
	t.Parallel()

	cfg := MigrationConfig{
		MigrationURL:   "postgres://owner:owner-secret-123@direct.example/tauco",
		RuntimeURL:     "postgres://runtime:runtime-secret-123@pooler.example/tauco",
		BootstrapRoles: true,
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "same host and port") {
		t.Fatalf("Validate() error = %v, want bootstrap host rejection", err)
	}
}

func TestMigrationConfigRejectsConnectionIdentityOverrides(t *testing.T) {
	t.Parallel()

	cfg := MigrationConfig{
		MigrationURL: "postgres://owner:owner-secret-123@127.0.0.1/tauco?options=-c%20role%3Dpostgres",
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "must not override") {
		t.Fatalf("Validate() error = %v, want query override rejection", err)
	}
}

func TestMigrationConfigRejectsNonASCIIBootstrapPassword(t *testing.T) {
	t.Parallel()

	cfg := MigrationConfig{
		MigrationURL:   "postgres://owner:owner-secret-123@127.0.0.1/tauco",
		RuntimeURL:     "postgres://runtime:rahasia-panjang-%F0%9F%94%92@127.0.0.1/tauco",
		BootstrapRoles: true,
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "printable ASCII") {
		t.Fatalf("Validate() error = %v, want bootstrap password rejection", err)
	}
}

func TestLoadMigrationConfigAllowsPreprovisionedRemoteRoles(t *testing.T) {
	t.Parallel()

	cfg, err := LoadMigrationConfig(mapLookup(map[string]string{
		"MIGRATION_DATABASE_URL": "postgres://tauco_owner:owner-secret@db.example/tauco?sslmode=verify-full",
	}))
	if err != nil {
		t.Fatalf("LoadMigrationConfig() error = %v", err)
	}
	if cfg.BootstrapRoles {
		t.Fatal("BootstrapRoles = true, want false")
	}
	if cfg.RuntimeURL != "" {
		t.Fatal("RuntimeURL should be optional when bootstrap is disabled")
	}
}

func TestMigrationConfigRejectsSharedLogin(t *testing.T) {
	t.Parallel()

	cfg := MigrationConfig{
		MigrationURL: "postgres://same:owner-secret@127.0.0.1/tauco",
		RuntimeURL:   "postgres://same:runtime-secret@127.0.0.1/tauco",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want shared-login rejection")
	}
	if !strings.Contains(err.Error(), "distinct") {
		t.Fatalf("Validate() error = %q, want credential separation reason", err)
	}
}

func TestMigrationConfigErrorsNeverExposePasswords(t *testing.T) {
	t.Parallel()

	const secret = "do-not-leak-this-password"
	_, err := LoadMigrationConfig(mapLookup(map[string]string{
		"MIGRATION_DATABASE_URL": "mysql://owner:" + secret + "@127.0.0.1/tauco",
	}))
	if err == nil {
		t.Fatal("LoadMigrationConfig() error = nil, want scheme rejection")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("configuration error leaked password: %v", err)
	}
	if !IsConfigError(err) {
		t.Fatalf("IsConfigError() = false for %T", err)
	}
}

func TestMigrationDatabaseURLLocksPrivateSearchPath(t *testing.T) {
	t.Parallel()

	got, err := migrationDatabaseURL(
		"postgres://owner:secret@127.0.0.1:5432/tauco?sslmode=disable",
	)
	if err != nil {
		t.Fatalf("migrationDatabaseURL() error = %v", err)
	}
	for _, expected := range []string{
		"search_path=tauco_app%2Cpg_catalog",
		"x-migrations-table=schema_migrations",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("migration URL %q does not contain %q", got, expected)
		}
	}
}

func TestNewSCRAMVerifierDoesNotContainPlaintext(t *testing.T) {
	t.Parallel()

	const password = "plain-runtime-password"
	verifier, err := newSCRAMVerifier(password)
	if err != nil {
		t.Fatalf("newSCRAMVerifier() error = %v", err)
	}
	if strings.Contains(verifier, password) {
		t.Fatal("SCRAM verifier contains plaintext password")
	}
	if !strings.HasPrefix(verifier, "SCRAM-SHA-256$4096:") {
		t.Fatalf("SCRAM verifier prefix = %q", verifier)
	}
}

func mapLookup(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, found := values[key]
		return value, found
	}
}
