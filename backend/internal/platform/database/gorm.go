package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultMaxOpenConnections = 5
	defaultMaxIdleConnections = 2
	defaultConnectionLifetime = 30 * time.Minute
	defaultConnectionIdleTime = 5 * time.Minute
)

// RuntimeConfig is the bounded PostgreSQL connection policy used by API
// repositories. The defaults are intentionally conservative for free-tier
// managed databases and horizontally scaled runtimes.
type RuntimeConfig struct {
	URL               string
	MaxOpenConns      int
	MaxIdleConns      int
	ConnMaxLifetime   time.Duration
	ConnMaxIdleTime   time.Duration
	PreferSimpleQuery bool
}

// LoadRuntimeConfig loads the runtime connection policy without ever
// including the secret-bearing URL in an error.
func LoadRuntimeConfig(lookup LookupEnv) (RuntimeConfig, error) {
	if lookup == nil {
		return RuntimeConfig{}, &ConfigError{
			Field:  "environment",
			Reason: "lookup function is required",
		}
	}

	rawURL, found := lookup("DATABASE_URL")
	if !found || strings.TrimSpace(rawURL) == "" {
		return RuntimeConfig{}, &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "is required",
		}
	}
	if _, err := parseDatabaseURL("DATABASE_URL", rawURL, true); err != nil {
		return RuntimeConfig{}, err
	}

	maxOpen, err := loadPositiveInt(
		lookup,
		"DATABASE_MAX_OPEN_CONNS",
		defaultMaxOpenConnections,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}
	maxIdle, err := loadNonNegativeInt(
		lookup,
		"DATABASE_MAX_IDLE_CONNS",
		defaultMaxIdleConnections,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if maxIdle > maxOpen {
		return RuntimeConfig{}, &ConfigError{
			Field:  "DATABASE_MAX_IDLE_CONNS",
			Reason: "must not exceed DATABASE_MAX_OPEN_CONNS",
		}
	}
	lifetime, err := loadPositiveDuration(
		lookup,
		"DATABASE_CONN_MAX_LIFETIME",
		defaultConnectionLifetime,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}
	idleTime, err := loadPositiveDuration(
		lookup,
		"DATABASE_CONN_MAX_IDLE_TIME",
		defaultConnectionIdleTime,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}

	return RuntimeConfig{
		URL:               strings.TrimSpace(rawURL),
		MaxOpenConns:      maxOpen,
		MaxIdleConns:      maxIdle,
		ConnMaxLifetime:   lifetime,
		ConnMaxIdleTime:   idleTime,
		PreferSimpleQuery: true,
	}, nil
}

// OpenGORM opens and verifies a runtime database connection. Prepared
// statements are disabled for Supavisor transaction-pool compatibility.
func OpenGORM(ctx context.Context, config RuntimeConfig) (*gorm.DB, error) {
	if ctx == nil {
		return nil, errors.New("open database: context is required")
	}
	if err := validateRuntimeConfig(config); err != nil {
		return nil, err
	}

	database, err := gorm.Open(
		postgres.New(postgres.Config{
			DSN:                  config.URL,
			PreferSimpleProtocol: config.PreferSimpleQuery,
		}),
		&gorm.Config{
			PrepareStmt:                              false,
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL runtime connection: %w", err)
	}

	sqlDatabase, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access PostgreSQL connection pool: %w", err)
	}
	applyConnectionPolicy(sqlDatabase, config)
	if err := sqlDatabase.PingContext(ctx); err != nil {
		_ = sqlDatabase.Close()
		return nil, fmt.Errorf("verify PostgreSQL runtime connection: %w", err)
	}
	if err := assertRuntimeIdentity(ctx, sqlDatabase); err != nil {
		_ = sqlDatabase.Close()
		return nil, err
	}
	return database, nil
}

// OpenMigrationGORM opens a narrowly pooled connection for deterministic seed
// and repository integration work. The seed adapter still performs
// `SET LOCAL ROLE tauco_migrator` inside its transaction, so supplying a
// runtime login fails closed.
func OpenMigrationGORM(
	ctx context.Context,
	databaseURL string,
) (*gorm.DB, error) {
	if ctx == nil {
		return nil, errors.New("open migration database: context is required")
	}
	if _, err := parseDatabaseURL(
		"MIGRATION_DATABASE_URL",
		databaseURL,
		true,
	); err != nil {
		return nil, err
	}

	database, err := gorm.Open(
		postgres.New(postgres.Config{
			DSN:                  databaseURL,
			PreferSimpleProtocol: true,
		}),
		&gorm.Config{
			PrepareStmt:                              false,
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL migration connection: %w", err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access PostgreSQL migration pool: %w", err)
	}
	sqlDatabase.SetMaxOpenConns(1)
	sqlDatabase.SetMaxIdleConns(1)
	sqlDatabase.SetConnMaxLifetime(defaultConnectionLifetime)
	sqlDatabase.SetConnMaxIdleTime(defaultConnectionIdleTime)
	if err := sqlDatabase.PingContext(ctx); err != nil {
		_ = sqlDatabase.Close()
		return nil, fmt.Errorf("verify PostgreSQL migration connection: %w", err)
	}
	return database, nil
}

func validateRuntimeConfig(config RuntimeConfig) error {
	if _, err := parseDatabaseURL("DATABASE_URL", config.URL, true); err != nil {
		return err
	}
	if config.MaxOpenConns < 1 {
		return &ConfigError{
			Field:  "DATABASE_MAX_OPEN_CONNS",
			Reason: "must be greater than zero",
		}
	}
	if config.MaxIdleConns < 0 || config.MaxIdleConns > config.MaxOpenConns {
		return &ConfigError{
			Field:  "DATABASE_MAX_IDLE_CONNS",
			Reason: "must be non-negative and not exceed max open connections",
		}
	}
	if config.ConnMaxLifetime <= 0 {
		return &ConfigError{
			Field:  "DATABASE_CONN_MAX_LIFETIME",
			Reason: "must be greater than zero",
		}
	}
	if config.ConnMaxIdleTime <= 0 {
		return &ConfigError{
			Field:  "DATABASE_CONN_MAX_IDLE_TIME",
			Reason: "must be greater than zero",
		}
	}
	if !config.PreferSimpleQuery {
		return &ConfigError{
			Field:  "DATABASE_URL",
			Reason: "runtime connections must use the pooler-safe simple protocol",
		}
	}
	return nil
}

func assertRuntimeIdentity(ctx context.Context, database *sql.DB) error {
	var (
		isSuperuser       bool
		canCreateDB       bool
		canCreateRole     bool
		canReplicate      bool
		canBypassRLS      bool
		isRuntimeMember   bool
		isMigratorMember  bool
		canCreateSchema   bool
		currentSchema     string
		canReadPages      bool
		canWritePages     bool
		canWritePageRevs  bool
		canWriteProducts  bool
		canWriteProdRevs  bool
		canCreateDatabase bool
		canCreateTemp     bool
		canCreatePublic   bool
	)
	err := database.QueryRowContext(ctx, `
SELECT
    role.rolsuper,
    role.rolcreatedb,
    role.rolcreaterole,
    role.rolreplication,
    role.rolbypassrls,
    pg_has_role(current_user, 'tauco_runtime', 'MEMBER'),
    pg_has_role(current_user, 'tauco_migrator', 'MEMBER'),
    has_schema_privilege(current_user, 'tauco_app', 'CREATE'),
    current_schema(),
    has_table_privilege(current_user, 'tauco_app.pages', 'SELECT'),
    (
        has_table_privilege(current_user, 'tauco_app.pages', 'INSERT')
        OR has_table_privilege(current_user, 'tauco_app.pages', 'UPDATE')
        OR has_table_privilege(current_user, 'tauco_app.pages', 'DELETE')
    ),
    (
        has_table_privilege(current_user, 'tauco_app.page_revisions', 'INSERT')
        OR has_table_privilege(current_user, 'tauco_app.page_revisions', 'UPDATE')
        OR has_table_privilege(current_user, 'tauco_app.page_revisions', 'DELETE')
    ),
    (
        has_table_privilege(current_user, 'tauco_app.products', 'INSERT')
        OR has_table_privilege(current_user, 'tauco_app.products', 'UPDATE')
        OR has_table_privilege(current_user, 'tauco_app.products', 'DELETE')
    ),
    (
        has_table_privilege(current_user, 'tauco_app.product_revisions', 'INSERT')
        OR has_table_privilege(current_user, 'tauco_app.product_revisions', 'UPDATE')
        OR has_table_privilege(current_user, 'tauco_app.product_revisions', 'DELETE')
    ),
    has_database_privilege(current_user, current_database(), 'CREATE'),
    has_database_privilege(current_user, current_database(), 'TEMPORARY'),
    has_schema_privilege(current_user, 'public', 'CREATE')
FROM pg_catalog.pg_roles AS role
WHERE role.rolname = current_user`).Scan(
		&isSuperuser,
		&canCreateDB,
		&canCreateRole,
		&canReplicate,
		&canBypassRLS,
		&isRuntimeMember,
		&isMigratorMember,
		&canCreateSchema,
		&currentSchema,
		&canReadPages,
		&canWritePages,
		&canWritePageRevs,
		&canWriteProducts,
		&canWriteProdRevs,
		&canCreateDatabase,
		&canCreateTemp,
		&canCreatePublic,
	)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL runtime identity: %w", err)
	}
	if isSuperuser ||
		canCreateDB ||
		canCreateRole ||
		canReplicate ||
		canBypassRLS ||
		!isRuntimeMember ||
		isMigratorMember ||
		canCreateSchema ||
		currentSchema != ApplicationSchema ||
		!canReadPages ||
		canWritePages ||
		canWritePageRevs ||
		canWriteProducts ||
		canWriteProdRevs ||
		canCreateDatabase ||
		canCreateTemp ||
		canCreatePublic {
		return errors.New(
			"verify PostgreSQL runtime identity: connection is not the " +
				"least-privilege tauco runtime role",
		)
	}

	var (
		unexpectedMemberships int
		hasAdminOption        bool
	)
	if err := database.QueryRowContext(ctx, `
SELECT
    count(*) FILTER (WHERE granted_role.rolname <> 'tauco_runtime'),
    coalesce(
        bool_or(
            membership.admin_option
            AND granted_role.rolname = 'tauco_runtime'
        ),
        false
    )
FROM pg_catalog.pg_auth_members AS membership
JOIN pg_catalog.pg_roles AS member
  ON member.oid = membership.member
JOIN pg_catalog.pg_roles AS granted_role
  ON granted_role.oid = membership.roleid
WHERE member.rolname = current_user`).Scan(
		&unexpectedMemberships,
		&hasAdminOption,
	); err != nil {
		return fmt.Errorf("verify PostgreSQL runtime memberships: %w", err)
	}
	if unexpectedMemberships != 0 || hasAdminOption {
		return errors.New(
			"verify PostgreSQL runtime identity: unexpected role membership",
		)
	}
	if err := assertRuntimeAuthorizationRoleIsolation(ctx, database); err != nil {
		return err
	}
	if err := assertRuntimeTablePrivilegeMatrix(ctx, database); err != nil {
		return err
	}
	return nil
}

func assertRuntimeAuthorizationRoleIsolation(
	ctx context.Context,
	database *sql.DB,
) error {
	var parentMemberships int
	if err := database.QueryRowContext(ctx, `
SELECT count(*)
FROM pg_catalog.pg_auth_members AS membership
JOIN pg_catalog.pg_roles AS member
  ON member.oid = membership.member
WHERE member.rolname = 'tauco_runtime'`).Scan(&parentMemberships); err != nil {
		return fmt.Errorf(
			"verify PostgreSQL runtime authorization role: %w",
			err,
		)
	}
	if parentMemberships != 0 {
		return errors.New(
			"verify PostgreSQL runtime identity: tauco_runtime inherits " +
				"an unexpected role",
		)
	}
	return nil
}

func assertRuntimeTablePrivilegeMatrix(
	ctx context.Context,
	database *sql.DB,
) error {
	var (
		mismatchedPrivileges int
		ownedTables          int
	)
	if err := database.QueryRowContext(ctx, `
WITH expected_tables(table_name) AS (
    VALUES
        ('pages'),
        ('page_revisions'),
        ('products'),
        ('product_revisions'),
        ('media_assets'),
        ('media_variants'),
        ('contact_messages'),
        ('background_jobs'),
        ('activity_logs')
),
expected_privileges(privilege_name) AS (
    VALUES
        ('SELECT'),
        ('INSERT'),
        ('UPDATE'),
        ('DELETE'),
        ('TRUNCATE'),
        ('REFERENCES'),
        ('TRIGGER')
),
matrix AS (
    SELECT
        table_name,
        privilege_name,
        CASE
            WHEN privilege_name = 'SELECT' THEN true
            WHEN table_name = 'contact_messages'
              AND privilege_name = 'INSERT' THEN true
            WHEN table_name = 'background_jobs'
              AND privilege_name IN ('INSERT', 'UPDATE') THEN true
            WHEN table_name = 'activity_logs'
              AND privilege_name = 'INSERT' THEN true
            ELSE false
        END AS expected
    FROM expected_tables
    CROSS JOIN expected_privileges
)
SELECT
    count(*) FILTER (
        WHERE has_table_privilege(
            current_user,
            format('tauco_app.%I', table_name),
            privilege_name
        ) IS DISTINCT FROM expected
    ),
    (
        SELECT count(*)
        FROM pg_catalog.pg_class AS relation
        JOIN pg_catalog.pg_namespace AS namespace
          ON namespace.oid = relation.relnamespace
        JOIN pg_catalog.pg_roles AS owner
          ON owner.oid = relation.relowner
        WHERE namespace.nspname = 'tauco_app'
          AND owner.rolname = current_user
          AND relation.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
    )
FROM matrix`).Scan(
		&mismatchedPrivileges,
		&ownedTables,
	); err != nil {
		return fmt.Errorf("verify PostgreSQL runtime table privileges: %w", err)
	}
	if mismatchedPrivileges != 0 || ownedTables != 0 {
		return errors.New(
			"verify PostgreSQL runtime identity: table privilege matrix " +
				"is not least-privilege",
		)
	}
	return nil
}

func applyConnectionPolicy(database *sql.DB, config RuntimeConfig) {
	database.SetMaxOpenConns(config.MaxOpenConns)
	database.SetMaxIdleConns(config.MaxIdleConns)
	database.SetConnMaxLifetime(config.ConnMaxLifetime)
	database.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

func loadPositiveInt(
	lookup LookupEnv,
	field string,
	fallback int,
) (int, error) {
	value, found := lookup(field)
	if !found || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 {
		return 0, &ConfigError{Field: field, Reason: "must be a positive integer"}
	}
	return parsed, nil
}

func loadNonNegativeInt(
	lookup LookupEnv,
	field string,
	fallback int,
) (int, error) {
	value, found := lookup(field)
	if !found || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0, &ConfigError{
			Field:  field,
			Reason: "must be a non-negative integer",
		}
	}
	return parsed, nil
}

func loadPositiveDuration(
	lookup LookupEnv,
	field string,
	fallback time.Duration,
) (time.Duration, error) {
	value, found := lookup(field)
	if !found || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, &ConfigError{
			Field:  field,
			Reason: "must be a positive Go duration",
		}
	}
	return parsed, nil
}
