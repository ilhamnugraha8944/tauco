// Package database owns PostgreSQL infrastructure configuration and migration
// wiring. Domain and application packages must not import this package.
package database

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	// ApplicationSchema is private from PostgREST's default public exposure.
	ApplicationSchema = "tauco_app"

	MigratorRole = "tauco_migrator"
	RuntimeRole  = "tauco_runtime"
)

// LookupEnv matches os.LookupEnv and enables deterministic configuration tests.
type LookupEnv func(key string) (value string, found bool)

// MigrationConfig contains secret-bearing database URLs. It deliberately does
// not implement fmt.Stringer and must never be logged as a whole value.
type MigrationConfig struct {
	MigrationURL   string
	RuntimeURL     string
	BootstrapRoles bool
}

// ConfigError identifies an invalid field without echoing its secret value.
type ConfigError struct {
	Field  string
	Reason string
}

func (e *ConfigError) Error() string {
	if e == nil {
		return "invalid database configuration"
	}
	return "invalid database configuration: " + e.Field + ": " + e.Reason
}

// LoadMigrationConfig reads and validates migration configuration.
func LoadMigrationConfig(lookup LookupEnv) (MigrationConfig, error) {
	if lookup == nil {
		return MigrationConfig{}, &ConfigError{
			Field:  "environment",
			Reason: "lookup function is required",
		}
	}

	migrationURL, found := lookup("MIGRATION_DATABASE_URL")
	if !found || strings.TrimSpace(migrationURL) == "" {
		return MigrationConfig{}, &ConfigError{
			Field:  "MIGRATION_DATABASE_URL",
			Reason: "is required",
		}
	}

	runtimeURL, runtimeFound := lookup("DATABASE_URL")
	bootstrap, err := loadBool(lookup, "MIGRATION_BOOTSTRAP_ROLES", false)
	if err != nil {
		return MigrationConfig{}, err
	}
	if bootstrap && (!runtimeFound || strings.TrimSpace(runtimeURL) == "") {
		return MigrationConfig{}, &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "is required when MIGRATION_BOOTSTRAP_ROLES is true",
		}
	}

	cfg := MigrationConfig{
		MigrationURL:   strings.TrimSpace(migrationURL),
		RuntimeURL:     strings.TrimSpace(runtimeURL),
		BootstrapRoles: bootstrap,
	}
	if err := cfg.Validate(); err != nil {
		return MigrationConfig{}, err
	}
	return cfg, nil
}

// Validate checks endpoint shape and enforces credential separation.
func (c MigrationConfig) Validate() error {
	migration, err := parseDatabaseURL("MIGRATION_DATABASE_URL", c.MigrationURL, true)
	if err != nil {
		return err
	}

	if c.RuntimeURL == "" {
		if c.BootstrapRoles {
			return &ConfigError{
				Field:  "DATABASE_URL",
				Reason: "is required when role bootstrap is enabled",
			}
		}
		return nil
	}

	runtime, err := parseDatabaseURL("DATABASE_URL", c.RuntimeURL, c.BootstrapRoles)
	if err != nil {
		return err
	}
	if migration.database != runtime.database {
		return &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "must target the same database as MIGRATION_DATABASE_URL",
		}
	}
	if migration.username == runtime.username {
		return &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "must use a login distinct from the migration login",
		}
	}
	if runtime.username == MigratorRole || runtime.username == RuntimeRole {
		return &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "must use a LOGIN role distinct from fixed NOLOGIN authorization roles",
		}
	}
	if c.BootstrapRoles &&
		(!strings.EqualFold(migration.host, runtime.host) || migration.port != runtime.port) {
		return &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "must use the same host and port as MIGRATION_DATABASE_URL during role bootstrap",
		}
	}
	if c.BootstrapRoles && !isBootstrapPassword(runtime.password) {
		return &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "runtime password must be 16-256 printable ASCII characters during role bootstrap",
		}
	}
	return nil
}

type databaseEndpoint struct {
	username string
	password string
	database string
	host     string
	port     string
}

func parseDatabaseURL(field, raw string, requirePassword bool) (databaseEndpoint, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "must be a valid PostgreSQL URL"}
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "must use postgres or postgresql scheme"}
	}
	if parsed.Hostname() == "" || parsed.User == nil || parsed.User.Username() == "" {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "must include host and username"}
	}
	if parsed.Fragment != "" {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "must not contain a URL fragment"}
	}
	for key := range parsed.Query() {
		switch strings.ToLower(key) {
		case "user",
			"password",
			"dbname",
			"database",
			"host",
			"hostaddr",
			"port",
			"service",
			"servicefile",
			"passfile",
			"options",
			"role",
			"search_path",
			"x-migrations-table":
			return databaseEndpoint{}, &ConfigError{
				Field:  field,
				Reason: "must not override identity, role, or search path through query parameters",
			}
		}
	}

	databaseName := strings.TrimPrefix(parsed.EscapedPath(), "/")
	unescapedDatabase, unescapeErr := url.PathUnescape(databaseName)
	if unescapeErr != nil || unescapedDatabase == "" || strings.Contains(unescapedDatabase, "/") {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "must include exactly one database name"}
	}

	password, hasPassword := parsed.User.Password()
	if requirePassword && (!hasPassword || password == "") {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "must include a non-empty password"}
	}
	if strings.ContainsRune(password, '\x00') {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "password contains an unsupported character"}
	}

	username := parsed.User.Username()
	if len(username) > 63 ||
		strings.ContainsRune(username, '\x00') ||
		strings.HasPrefix(strings.ToLower(username), "pg_") {
		return databaseEndpoint{}, &ConfigError{Field: field, Reason: "contains an invalid PostgreSQL username"}
	}
	return databaseEndpoint{
		username: username,
		password: password,
		database: unescapedDatabase,
		host:     strings.ToLower(parsed.Hostname()),
		port:     normalizedPostgresPort(parsed.Port()),
	}, nil
}

func normalizedPostgresPort(port string) string {
	if port == "" {
		return "5432"
	}
	return port
}

func isBootstrapPassword(password string) bool {
	if len(password) < 16 || len(password) > 256 {
		return false
	}
	for index := 0; index < len(password); index++ {
		if password[index] < 0x21 || password[index] > 0x7e {
			return false
		}
	}
	return true
}

func loadBool(lookup LookupEnv, field string, fallback bool) (bool, error) {
	value, found := lookup(field)
	if !found || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	result, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, &ConfigError{Field: field, Reason: "must be true or false"}
	}
	return result, nil
}

// IsConfigError reports whether err is a redacted database ConfigError.
func IsConfigError(err error) bool {
	var target *ConfigError
	return errors.As(err, &target)
}

func migrationDatabaseURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse migration database URL: %w", err)
	}
	query := parsed.Query()
	query.Set("search_path", ApplicationSchema+",pg_catalog")
	query.Set("x-migrations-table", "schema_migrations")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
