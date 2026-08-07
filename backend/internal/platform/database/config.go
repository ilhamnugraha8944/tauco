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

	MigratorRole     = "tauco_migrator"
	RuntimeRole      = "tauco_runtime"
	AdminRuntimeRole = "tauco_admin_runtime"
)

// LookupEnv matches os.LookupEnv and enables deterministic configuration tests.
type LookupEnv func(key string) (value string, found bool)

// MigrationConfig contains secret-bearing database URLs. It deliberately does
// not implement fmt.Stringer and must never be logged as a whole value.
type MigrationConfig struct {
	MigrationURL   string
	RuntimeURL     string
	AdminURL       string
	BootstrapRoles bool
	Profile        DeploymentProfile
}

type DeploymentProfile string

const (
	ProfileOwned    DeploymentProfile = "owned"
	ProfileSupabase DeploymentProfile = "supabase"
)

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
	adminURL, adminFound := lookup("ADMIN_DATABASE_URL")
	bootstrap, err := loadBool(lookup, "MIGRATION_BOOTSTRAP_ROLES", false)
	if err != nil {
		return MigrationConfig{}, err
	}
	profile, err := loadDeploymentProfile(lookup)
	if err != nil {
		return MigrationConfig{}, err
	}
	if bootstrap && (!runtimeFound || strings.TrimSpace(runtimeURL) == "") {
		return MigrationConfig{}, &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "is required when MIGRATION_BOOTSTRAP_ROLES is true",
		}
	}
	if bootstrap && (!adminFound || strings.TrimSpace(adminURL) == "") {
		return MigrationConfig{}, &ConfigError{
			Field:  "ADMIN_DATABASE_URL",
			Reason: "is required when MIGRATION_BOOTSTRAP_ROLES is true",
		}
	}

	cfg := MigrationConfig{
		MigrationURL:   strings.TrimSpace(migrationURL),
		RuntimeURL:     strings.TrimSpace(runtimeURL),
		AdminURL:       strings.TrimSpace(adminURL),
		BootstrapRoles: bootstrap,
		Profile:        profile,
	}
	if err := cfg.Validate(); err != nil {
		return MigrationConfig{}, err
	}
	return cfg, nil
}

// Validate checks endpoint shape and enforces credential separation.
func (c MigrationConfig) Validate() error {
	if c.Profile == "" {
		c.Profile = ProfileOwned
	}
	if c.Profile != ProfileOwned && c.Profile != ProfileSupabase {
		return &ConfigError{Field: "DATABASE_DEPLOYMENT_PROFILE", Reason: "must be owned or supabase"}
	}
	migration, err := parseDatabaseURL("MIGRATION_DATABASE_URL", c.MigrationURL, true)
	if err != nil {
		return err
	}
	var projectRef string
	if c.Profile == ProfileSupabase {
		if c.BootstrapRoles {
			return &ConfigError{Field: "MIGRATION_BOOTSTRAP_ROLES", Reason: "must be false for the supabase profile"}
		}
		if migration.port != "5432" || !secureSSLMode(migration.sslMode) {
			return &ConfigError{Field: "MIGRATION_DATABASE_URL", Reason: "must use TLS and the session/direct port 5432 for the supabase profile"}
		}
		if c.RuntimeURL == "" || c.AdminURL == "" {
			return &ConfigError{Field: "DATABASE_URL", Reason: "runtime and admin URLs are required for the supabase profile"}
		}
		projectRef, err = supabaseProjectRef("MIGRATION_DATABASE_URL", migration.username, ManagedMigrationLogin)
		if err != nil {
			return err
		}
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
	if c.Profile == ProfileSupabase && (runtime.port != "6543" || !secureSSLMode(runtime.sslMode)) {
		return &ConfigError{Field: "DATABASE_URL", Reason: "must use TLS and transaction pooler port 6543 for the supabase profile"}
	}
	if c.Profile == ProfileSupabase {
		runtimeProjectRef, projectErr := supabaseProjectRef("DATABASE_URL", runtime.username, ManagedRuntimeLogin)
		if projectErr != nil {
			return projectErr
		}
		if runtimeProjectRef != projectRef {
			return &ConfigError{Field: "DATABASE_URL", Reason: "must target the same Supabase project as MIGRATION_DATABASE_URL"}
		}
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
	if runtime.username == MigratorRole || runtime.username == RuntimeRole || runtime.username == AdminRuntimeRole {
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
	if c.AdminURL == "" {
		if c.BootstrapRoles {
			return &ConfigError{
				Field:  "ADMIN_DATABASE_URL",
				Reason: "is required when role bootstrap is enabled",
			}
		}
		return nil
	}

	admin, err := parseDatabaseURL("ADMIN_DATABASE_URL", c.AdminURL, c.BootstrapRoles)
	if err != nil {
		return err
	}
	if c.Profile == ProfileSupabase && (admin.port != "6543" || !secureSSLMode(admin.sslMode)) {
		return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "must use TLS and transaction pooler port 6543 for the supabase profile"}
	}
	if c.Profile == ProfileSupabase {
		adminProjectRef, projectErr := supabaseProjectRef("ADMIN_DATABASE_URL", admin.username, ManagedAdminLogin)
		if projectErr != nil {
			return projectErr
		}
		if adminProjectRef != projectRef {
			return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "must target the same Supabase project as MIGRATION_DATABASE_URL"}
		}
	}
	if migration.database != admin.database {
		return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "must target the same database as MIGRATION_DATABASE_URL"}
	}
	if admin.username == migration.username || admin.username == runtime.username {
		return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "must use a login distinct from migration and public runtime logins"}
	}
	if admin.username == MigratorRole || admin.username == RuntimeRole || admin.username == AdminRuntimeRole {
		return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "must use a LOGIN role distinct from fixed NOLOGIN authorization roles"}
	}
	if c.BootstrapRoles &&
		(!strings.EqualFold(migration.host, admin.host) || migration.port != admin.port) {
		return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "must use the same host and port as MIGRATION_DATABASE_URL during role bootstrap"}
	}
	if c.BootstrapRoles && !isBootstrapPassword(admin.password) {
		return &ConfigError{Field: "ADMIN_DATABASE_URL", Reason: "admin password must be 16-256 printable ASCII characters during role bootstrap"}
	}
	return nil
}

type databaseEndpoint struct {
	username string
	password string
	database string
	host     string
	port     string
	sslMode  string
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
		sslMode:  strings.ToLower(parsed.Query().Get("sslmode")),
	}, nil
}

func secureSSLMode(mode string) bool {
	return mode == "require" || mode == "verify-ca" || mode == "verify-full"
}

func supabaseProjectRef(field, username, login string) (string, error) {
	prefix := login + "."
	if !strings.HasPrefix(username, prefix) {
		return "", &ConfigError{Field: field, Reason: "must use the dedicated " + login + ".<project-ref> login"}
	}
	projectRef := strings.TrimPrefix(username, prefix)
	if projectRef == "" || strings.Contains(projectRef, ".") {
		return "", &ConfigError{Field: field, Reason: "must include exactly one Supabase project reference"}
	}
	for _, character := range projectRef {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return "", &ConfigError{Field: field, Reason: "contains an invalid Supabase project reference"}
		}
	}
	return projectRef, nil
}

func loadDeploymentProfile(lookup LookupEnv) (DeploymentProfile, error) {
	value, found := lookup("DATABASE_DEPLOYMENT_PROFILE")
	if !found || strings.TrimSpace(value) == "" {
		return ProfileOwned, nil
	}
	profile := DeploymentProfile(strings.ToLower(strings.TrimSpace(value)))
	if profile != ProfileOwned && profile != ProfileSupabase {
		return "", &ConfigError{Field: "DATABASE_DEPLOYMENT_PROFILE", Reason: "must be owned or supabase"}
	}
	return profile, nil
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
