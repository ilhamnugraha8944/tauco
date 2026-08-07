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
	Profile           DeploymentProfile
}

// LoadRuntimeConfig loads the runtime connection policy without ever
// including the secret-bearing URL in an error.
func LoadRuntimeConfig(lookup LookupEnv) (RuntimeConfig, error) {
	return loadRuntimeConfig(lookup, "DATABASE_URL")
}

// LoadAdminRuntimeConfig loads the isolated CMS connection policy.
func LoadAdminRuntimeConfig(lookup LookupEnv) (RuntimeConfig, error) {
	return loadRuntimeConfig(lookup, "ADMIN_DATABASE_URL")
}

func loadRuntimeConfig(lookup LookupEnv, urlField string) (RuntimeConfig, error) {
	if lookup == nil {
		return RuntimeConfig{}, &ConfigError{
			Field:  "environment",
			Reason: "lookup function is required",
		}
	}

	rawURL, found := lookup(urlField)
	if !found || strings.TrimSpace(rawURL) == "" {
		return RuntimeConfig{}, &ConfigError{
			Field:  urlField,
			Reason: "is required",
		}
	}
	endpoint, err := parseDatabaseURL(urlField, rawURL, true)
	if err != nil {
		return RuntimeConfig{}, err
	}
	profile, err := loadDeploymentProfile(lookup)
	if err != nil {
		return RuntimeConfig{}, err
	}
	defaultOpen, defaultIdle := defaultMaxOpenConnections, defaultMaxIdleConnections
	if profile == ProfileSupabase {
		defaultOpen, defaultIdle = 1, 0
		if endpoint.port != "6543" || !secureSSLMode(endpoint.sslMode) {
			return RuntimeConfig{}, &ConfigError{Field: urlField, Reason: "must use TLS and transaction pooler port 6543 for the supabase profile"}
		}
	}

	maxOpen, err := loadPositiveInt(
		lookup,
		"DATABASE_MAX_OPEN_CONNS",
		defaultOpen,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}
	maxIdle, err := loadNonNegativeInt(
		lookup,
		"DATABASE_MAX_IDLE_CONNS",
		defaultIdle,
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
	if profile == ProfileSupabase && (maxOpen > 2 || maxIdle != 0) {
		return RuntimeConfig{}, &ConfigError{Field: "DATABASE_MAX_OPEN_CONNS", Reason: "supabase profile requires at most 2 open and exactly 0 idle connections"}
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
		Profile:           profile,
	}, nil
}

// OpenGORM opens and verifies a runtime database connection. Prepared
// statements are disabled for Supavisor transaction-pool compatibility.
func OpenGORM(ctx context.Context, config RuntimeConfig) (*gorm.DB, error) {
	return openGORM(ctx, config, "DATABASE_URL", func(ctx context.Context, db *sql.DB) error {
		return assertRuntimeIdentity(ctx, db, config.Profile)
	})
}

// OpenAdminGORM opens the CMS connection and fails closed unless it inherits
// exactly the fixed least-privilege admin authorization role.
func OpenAdminGORM(ctx context.Context, config RuntimeConfig) (*gorm.DB, error) {
	return openGORM(ctx, config, "ADMIN_DATABASE_URL", func(ctx context.Context, db *sql.DB) error {
		return assertAdminIdentity(ctx, db, config.Profile)
	})
}

func openGORM(
	ctx context.Context,
	config RuntimeConfig,
	urlField string,
	assertIdentity func(context.Context, *sql.DB) error,
) (*gorm.DB, error) {
	if ctx == nil {
		return nil, errors.New("open database: context is required")
	}
	if err := validateRuntimeConfig(config, urlField); err != nil {
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
	if err := assertIdentity(ctx, sqlDatabase); err != nil {
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

func validateRuntimeConfig(config RuntimeConfig, urlField string) error {
	endpoint, err := parseDatabaseURL(urlField, config.URL, true)
	if err != nil {
		return err
	}
	profile := config.Profile
	if profile == "" {
		profile = ProfileOwned
	}
	if profile != ProfileOwned && profile != ProfileSupabase {
		return &ConfigError{Field: "DATABASE_DEPLOYMENT_PROFILE", Reason: "must be owned or supabase"}
	}
	if profile == ProfileSupabase && (endpoint.port != "6543" || !secureSSLMode(endpoint.sslMode) || config.MaxOpenConns > 2 || config.MaxIdleConns != 0) {
		return &ConfigError{Field: urlField, Reason: "supabase runtime requires TLS, transaction port 6543, at most 2 open, and 0 idle connections"}
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
			Field:  urlField,
			Reason: "runtime connections must use the pooler-safe simple protocol",
		}
	}
	return nil
}

func assertAdminIdentity(ctx context.Context, database *sql.DB, profile DeploymentProfile) error {
	var (
		isSuperuser, canCreateDB, canCreateRole, canReplicate, canBypassRLS bool
		isAdminMember, isRuntimeMember, isMigratorMember                    bool
		canCreateSchema, canCreateDatabase, canCreateTemp, canCreatePublic  bool
		canReadUsers, canWriteUsers, canInsertRevision, canUpdatePages      bool
		canDeleteInbox, canWritePermissions                                 bool
		currentSchema                                                       string
	)
	err := database.QueryRowContext(ctx, `
SELECT
    role.rolsuper, role.rolcreatedb, role.rolcreaterole,
    role.rolreplication, role.rolbypassrls,
    pg_has_role(current_user, 'tauco_admin_runtime', 'MEMBER'),
    pg_has_role(current_user, 'tauco_runtime', 'MEMBER'),
    pg_has_role(current_user, 'tauco_migrator', 'MEMBER'),
    has_schema_privilege(current_user, 'tauco_app', 'CREATE'),
    has_database_privilege(current_user, current_database(), 'CREATE'),
    has_database_privilege(current_user, current_database(), 'TEMPORARY'),
    has_schema_privilege(current_user, 'public', 'CREATE'),
    has_table_privilege(current_user, 'tauco_app.admin_users', 'SELECT'),
    has_table_privilege(current_user, 'tauco_app.admin_users', 'INSERT,UPDATE'),
    has_table_privilege(current_user, 'tauco_app.page_revisions', 'INSERT'),
    has_table_privilege(current_user, 'tauco_app.pages', 'UPDATE'),
    has_table_privilege(current_user, 'tauco_app.contact_messages', 'DELETE'),
    has_table_privilege(current_user, 'tauco_app.permissions', 'INSERT,UPDATE,DELETE'),
    current_schema()
FROM pg_catalog.pg_roles AS role
WHERE role.rolname = current_user`).Scan(
		&isSuperuser, &canCreateDB, &canCreateRole, &canReplicate, &canBypassRLS,
		&isAdminMember, &isRuntimeMember, &isMigratorMember,
		&canCreateSchema, &canCreateDatabase, &canCreateTemp, &canCreatePublic,
		&canReadUsers, &canWriteUsers, &canInsertRevision, &canUpdatePages,
		&canDeleteInbox, &canWritePermissions, &currentSchema,
	)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL admin identity: %w", err)
	}
	if isSuperuser || canCreateDB || canCreateRole || canReplicate || canBypassRLS ||
		!isAdminMember || isRuntimeMember || isMigratorMember || canCreateSchema ||
		canCreateDatabase || (canCreateTemp && profile != ProfileSupabase) || canCreatePublic || currentSchema != ApplicationSchema ||
		!canReadUsers || !canWriteUsers || !canInsertRevision || !canUpdatePages ||
		canDeleteInbox || canWritePermissions {
		return errors.New("verify PostgreSQL admin identity: connection is not least-privilege tauco admin runtime")
	}
	if err := assertAdminAuthorizationIsolation(ctx, database); err != nil {
		return err
	}
	if err := assertAdminPrivilegeMatrix(ctx, database); err != nil {
		return err
	}
	return nil
}

func assertAdminAuthorizationIsolation(ctx context.Context, database *sql.DB) error {
	var unexpectedMemberships int
	var hasAdminOption bool
	if err := database.QueryRowContext(ctx, `
SELECT
    count(*) FILTER (WHERE granted.rolname <> 'tauco_admin_runtime'),
    coalesce(bool_or(membership.admin_option), false)
FROM pg_catalog.pg_auth_members AS membership
JOIN pg_catalog.pg_roles AS member ON member.oid = membership.member
JOIN pg_catalog.pg_roles AS granted ON granted.oid = membership.roleid
WHERE member.rolname = current_user`).Scan(&unexpectedMemberships, &hasAdminOption); err != nil {
		return fmt.Errorf("verify PostgreSQL admin memberships: %w", err)
	}
	if unexpectedMemberships != 0 || hasAdminOption {
		return errors.New("verify PostgreSQL admin identity: unexpected role membership")
	}

	var inheritedRoles int
	if err := database.QueryRowContext(ctx, `
SELECT count(*)
FROM pg_catalog.pg_auth_members AS membership
JOIN pg_catalog.pg_roles AS member ON member.oid = membership.member
WHERE member.rolname = 'tauco_admin_runtime'`).Scan(&inheritedRoles); err != nil {
		return fmt.Errorf("verify PostgreSQL admin authorization role: %w", err)
	}
	if inheritedRoles != 0 {
		return errors.New("verify PostgreSQL admin identity: tauco_admin_runtime inherits an unexpected role")
	}
	return nil
}

func assertAdminPrivilegeMatrix(ctx context.Context, database *sql.DB) error {
	var mismatched, owned int
	if err := database.QueryRowContext(ctx, `
WITH expected(table_name, selects, inserts, updates) AS (
    VALUES
        ('admin_users', true, true, true),
        ('admin_sessions', true, true, true),
        ('admin_refresh_tokens', true, true, true),
        ('mfa_credentials', true, true, true),
        ('mfa_recovery_codes', true, true, true),
        ('roles', true, false, false),
        ('permissions', true, false, false),
        ('user_roles', true, true, false),
        ('role_permissions', true, true, false),
        ('pages', true, true, true),
        ('products', true, true, true),
        ('media_assets', true, true, true),
        ('media_variants', true, true, true),
        ('media_upload_intents', true, true, true),
        ('contact_messages', true, true, true),
        ('background_jobs', true, true, true),
        ('page_revisions', true, true, false),
        ('product_revisions', true, true, false),
        ('page_revision_media', true, true, false),
        ('product_revision_media', true, true, false),
        ('activity_logs', true, true, false)
), relations AS (
    SELECT relation.relname AS table_name
    FROM pg_catalog.pg_class AS relation
    JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
    WHERE namespace.nspname = 'tauco_app'
      AND relation.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
), matrix AS (
    SELECT relations.table_name, privilege_name,
        coalesce(CASE privilege_name WHEN 'SELECT' THEN selects WHEN 'INSERT' THEN inserts WHEN 'UPDATE' THEN updates ELSE false END, false) AS expected
    FROM relations
    CROSS JOIN (VALUES ('SELECT'), ('INSERT'), ('UPDATE'), ('DELETE'), ('TRUNCATE'), ('REFERENCES'), ('TRIGGER')) AS privileges(privilege_name)
    LEFT JOIN expected USING (table_name)
)
SELECT
    count(*) FILTER (WHERE has_table_privilege(current_user, format('tauco_app.%I', table_name), privilege_name) IS DISTINCT FROM expected),
    (SELECT count(*) FROM pg_catalog.pg_class AS relation
     JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
     JOIN pg_catalog.pg_roles AS owner ON owner.oid = relation.relowner
     WHERE namespace.nspname = 'tauco_app' AND owner.rolname = current_user
       AND relation.relkind IN ('r', 'p', 'v', 'm', 'S', 'f'))
FROM matrix`).Scan(&mismatched, &owned); err != nil {
		return fmt.Errorf("verify PostgreSQL admin table privileges: %w", err)
	}
	if mismatched != 0 || owned != 0 {
		return errors.New("verify PostgreSQL admin identity: table privilege matrix is not least-privilege")
	}

	var unexpectedFunctions int
	if err := database.QueryRowContext(ctx, `
SELECT count(*)
FROM pg_catalog.pg_proc AS procedure
JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = procedure.pronamespace
WHERE namespace.nspname = 'tauco_app'
  AND has_function_privilege(current_user, procedure.oid, 'EXECUTE') IS DISTINCT FROM
      (procedure.proname IN (
          'tauco_set_updated_at',
          'tauco_reject_revision_mutation',
          'tauco_assert_page_published_revision',
          'tauco_assert_product_published_revision',
          'tauco_reject_activity_log_mutation'
      ))`).Scan(&unexpectedFunctions); err != nil {
		return fmt.Errorf("verify PostgreSQL admin function privileges: %w", err)
	}
	if unexpectedFunctions != 0 {
		return errors.New("verify PostgreSQL admin identity: function privilege matrix is not least-privilege")
	}
	return nil
}

func assertRuntimeIdentity(ctx context.Context, database *sql.DB, profile DeploymentProfile) error {
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
		(canCreateTemp && profile != ProfileSupabase) ||
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
        ('media_upload_intents'),
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
            WHEN table_name IN ('media_assets', 'media_variants')
              AND privilege_name IN ('INSERT', 'UPDATE') THEN true
            WHEN table_name = 'media_upload_intents'
              AND privilege_name = 'UPDATE' THEN true
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
