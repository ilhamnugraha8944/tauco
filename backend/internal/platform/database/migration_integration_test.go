package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const migrationIntegrationLock int64 = 839_103_221_701

func TestMigrationRoundTripAndRuntimePrivileges(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("MIGRATION_TEST_DATABASE_URL"))
	if baseURL == "" {
		t.Skip("set MIGRATION_TEST_DATABASE_URL to run PostgreSQL migration integration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect migration test database: %v", err)
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(
		ctx,
		"SELECT pg_advisory_lock($1)",
		migrationIntegrationLock,
	); err != nil {
		t.Fatalf("acquire migration integration lock: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationIntegrationLock)
	}()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	mainLikeDatabaseName := "tauco_main_like_test_" + suffix
	databaseName := "tauco_migration_test_" + suffix
	migrationRoleName := "tauco_migrator_test_" + suffix
	rotatedMigrationRoleName := "tauco_migrator_rotated_test_" + suffix
	mainLikeRuntimeRoleName := "tauco_main_runtime_test_" + suffix
	runtimeRoleName := "tauco_runtime_test_" + suffix
	mainLikeAdminRoleName := "tauco_main_admin_test_" + suffix
	adminRoleName := "tauco_admin_test_" + suffix
	if len(migrationRoleName) > 63 {
		migrationRoleName = migrationRoleName[:63]
	}
	if len(runtimeRoleName) > 63 {
		runtimeRoleName = runtimeRoleName[:63]
	}
	if len(rotatedMigrationRoleName) > 63 {
		rotatedMigrationRoleName = rotatedMigrationRoleName[:63]
	}
	if len(mainLikeRuntimeRoleName) > 63 {
		mainLikeRuntimeRoleName = mainLikeRuntimeRoleName[:63]
	}
	if len(mainLikeAdminRoleName) > 63 {
		mainLikeAdminRoleName = mainLikeAdminRoleName[:63]
	}
	if len(adminRoleName) > 63 {
		adminRoleName = adminRoleName[:63]
	}
	const migrationPassword = "B3-migration-only-test-password"
	const rotatedMigrationPassword = "B3-rotated-migration-test-password"
	const mainLikeRuntimePassword = "B3-main-runtime-only-test-password"
	const runtimePassword = "B3-runtime-only-test-password"
	const mainLikeAdminPassword = "C1-main-admin-only-test-password"
	const adminPassword = "C1-admin-only-test-password"

	mainLikeDatabaseIdentifier := pgx.Identifier{mainLikeDatabaseName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+mainLikeDatabaseIdentifier); err != nil {
		t.Fatalf("create main-like database: %v", err)
	}
	databaseIdentifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+databaseIdentifier); err != nil {
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+mainLikeDatabaseIdentifier+" WITH (FORCE)")
		t.Fatalf("create disposable database: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(
			context.Background(),
			"DROP DATABASE IF EXISTS "+databaseIdentifier+" WITH (FORCE)",
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP DATABASE IF EXISTS "+mainLikeDatabaseIdentifier+" WITH (FORCE)",
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{runtimeRoleName}.Sanitize(),
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{migrationRoleName}.Sanitize(),
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{rotatedMigrationRoleName}.Sanitize(),
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{mainLikeRuntimeRoleName}.Sanitize(),
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{mainLikeAdminRoleName}.Sanitize(),
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{adminRoleName}.Sanitize(),
		)
	}()

	mainLikeBootstrapURL := replaceDatabaseAndUser(
		t,
		baseURL,
		mainLikeDatabaseName,
		"",
		"",
	)
	mainLikeRuntimeURL := replaceDatabaseAndUser(
		t,
		baseURL,
		mainLikeDatabaseName,
		mainLikeRuntimeRoleName,
		mainLikeRuntimePassword,
	)
	mainLikeAdminURL := replaceDatabaseAndUser(
		t,
		baseURL,
		mainLikeDatabaseName,
		mainLikeAdminRoleName,
		mainLikeAdminPassword,
	)
	mainLikeConfig := MigrationConfig{
		MigrationURL:   mainLikeBootstrapURL,
		RuntimeURL:     mainLikeRuntimeURL,
		AdminURL:       mainLikeAdminURL,
		BootstrapRoles: true,
	}
	if err := BootstrapRoles(ctx, mainLikeConfig); err != nil {
		t.Fatalf("BootstrapRoles(main-like) error = %v", err)
	}
	mainLikeMigrator, err := NewMigrator(mainLikeBootstrapURL)
	if err != nil {
		t.Fatalf("NewMigrator(main-like) error = %v", err)
	}
	if err := mainLikeMigrator.Up(); err != nil {
		_ = mainLikeMigrator.Close()
		t.Fatalf("migration Up(main-like) error = %v", err)
	}
	if err := mainLikeMigrator.Close(); err != nil {
		t.Fatalf("close main-like migrator: %v", err)
	}

	bootstrapURL := replaceDatabaseAndUser(t, baseURL, databaseName, "", "")
	runtimeURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		runtimeRoleName,
		runtimePassword,
	)
	adminURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		adminRoleName,
		adminPassword,
	)
	cfg := MigrationConfig{
		MigrationURL:   bootstrapURL,
		RuntimeURL:     runtimeURL,
		AdminURL:       adminURL,
		BootstrapRoles: true,
	}
	if err := BootstrapRoles(ctx, cfg); err != nil {
		t.Fatalf("BootstrapRoles() error = %v", err)
	}
	if _, err := admin.Exec(
		ctx,
		"GRANT "+RuntimeRole+" TO "+
			pgx.Identifier{runtimeRoleName}.Sanitize()+
			" WITH ADMIN OPTION",
	); err != nil {
		t.Fatalf("inject runtime membership admin-option drift: %v", err)
	}
	if err := BootstrapRoles(ctx, cfg); err != nil {
		t.Fatalf("second idempotent BootstrapRoles() error = %v", err)
	}

	bootstrapConn, err := pgx.Connect(ctx, bootstrapURL)
	if err != nil {
		t.Fatalf("connect disposable database for migration login: %v", err)
	}
	migrationIdentifier := pgx.Identifier{migrationRoleName}.Sanitize()
	if _, err := bootstrapConn.Exec(ctx,
		"CREATE ROLE "+migrationIdentifier+
			" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE"+
			" NOREPLICATION NOBYPASSRLS PASSWORD '"+migrationPassword+"'"+
			"; GRANT "+MigratorRole+" TO "+migrationIdentifier,
	); err != nil {
		bootstrapConn.Close(ctx)
		t.Fatalf("create non-super migration login: %v", err)
	}
	bootstrapConn.Close(ctx)

	migrationURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		migrationRoleName,
		migrationPassword,
	)
	migrator, err := NewMigrator(migrationURL)
	if err != nil {
		t.Fatalf("NewMigrator() error = %v", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		t.Fatalf("migration Up() error = %v", err)
	}
	assertMigrationVersion(t, migrator, 5)
	if err := BootstrapRoles(ctx, cfg); err != nil {
		t.Fatalf("post-migration idempotent BootstrapRoles() error = %v", err)
	}
	if err := BootstrapRoles(ctx, mainLikeConfig); err != nil {
		t.Fatalf("repeat BootstrapRoles(main-like) error = %v", err)
	}

	ownerConn, err := pgx.Connect(ctx, bootstrapURL)
	if err != nil {
		t.Fatalf("connect migrated database as owner: %v", err)
	}
	defer ownerConn.Close(ctx)

	assertPrivateSchema(t, ctx, ownerConn)
	rotatedIdentifier := pgx.Identifier{rotatedMigrationRoleName}.Sanitize()
	if _, err := ownerConn.Exec(ctx,
		"CREATE ROLE "+rotatedIdentifier+
			" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE"+
			" NOREPLICATION NOBYPASSRLS PASSWORD '"+rotatedMigrationPassword+"'"+
			"; GRANT "+MigratorRole+" TO "+rotatedIdentifier,
	); err != nil {
		t.Fatalf("create rotated non-super migration login: %v", err)
	}
	assertSingleMembership(
		t,
		ctx,
		ownerConn,
		rotatedMigrationRoleName,
		MigratorRole,
	)
	rotatedURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		rotatedMigrationRoleName,
		rotatedMigrationPassword,
	)
	rotatedMigrator, err := NewMigrator(rotatedURL)
	if err != nil {
		t.Fatalf("NewMigrator(rotated login) error = %v", err)
	}
	assertMigrationVersion(t, rotatedMigrator, 5)
	if err := rotatedMigrator.DownOne(); err != nil {
		t.Fatalf("rotated migration DownOne() error = %v", err)
	}
	assertMigrationVersion(t, rotatedMigrator, 4)
	if err := rotatedMigrator.Up(); err != nil {
		t.Fatalf("rotated migration Up() error = %v", err)
	}
	assertMigrationVersion(t, rotatedMigrator, 5)
	if err := rotatedMigrator.Close(); err != nil {
		t.Fatalf("close rotated migrator: %v", err)
	}

	assertRoleSecurity(t, ctx, ownerConn, migrationRoleName, runtimeRoleName, adminRoleName)
	assertSingleMembership(
		t,
		ctx,
		ownerConn,
		mainLikeRuntimeRoleName,
		RuntimeRole,
	)
	assertSingleMembership(
		t,
		ctx,
		ownerConn,
		mainLikeAdminRoleName,
		AdminRuntimeRole,
	)
	assertAuthorizationDatabaseGrants(
		t,
		ctx,
		ownerConn,
		RuntimeRole,
		runtimeRoleName,
		databaseName,
	)
	assertAuthorizationDatabaseGrants(
		t,
		ctx,
		ownerConn,
		RuntimeRole,
		mainLikeRuntimeRoleName,
		mainLikeDatabaseName,
	)
	assertAuthorizationDatabaseGrants(
		t,
		ctx,
		ownerConn,
		AdminRuntimeRole,
		adminRoleName,
		databaseName,
	)
	mainLikeRuntimeConn, err := pgx.Connect(ctx, mainLikeRuntimeURL)
	if err != nil {
		t.Fatalf("reconnect main-like runtime after disposable bootstrap: %v", err)
	}
	var mainLikeProductCount int
	if err := mainLikeRuntimeConn.QueryRow(
		ctx,
		"SELECT count(*) FROM tauco_app.products",
	).Scan(&mainLikeProductCount); err != nil {
		mainLikeRuntimeConn.Close(ctx)
		t.Fatalf("read main-like database after disposable bootstrap: %v", err)
	}
	mainLikeRuntimeConn.Close(ctx)
	assertPublishedRevisionIntegrity(t, ctx, ownerConn)
	assertActivityLogAppendOnly(t, ctx, ownerConn)
	assertAdminCMSFoundation(t, ctx, ownerConn)

	runtimeConn, err := pgx.Connect(ctx, runtimeURL)
	if err != nil {
		t.Fatalf("connect with real runtime login: %v", err)
	}
	assertRuntimePrivileges(t, ctx, runtimeConn)
	runtimeConn.Close(ctx)

	adminConn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect with admin runtime login: %v", err)
	}
	assertAdminPrivileges(t, ctx, adminConn)
	adminConn.Close(ctx)

	if err := migrator.DownAll(); err != nil {
		t.Fatalf("migration DownAll() error = %v", err)
	}
	assertMigrationVersion(t, migrator, 0)
	assertDomainTablesAbsent(t, ctx, ownerConn)

	if err := migrator.Up(); err != nil {
		t.Fatalf("second migration Up() error = %v", err)
	}
	assertMigrationVersion(t, migrator, 5)
	if err := migrator.DownAll(); err != nil {
		t.Fatalf("second migration DownAll() error = %v", err)
	}
}

func assertPrivateSchema(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	var publicCount int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (
      'pages', 'page_revisions', 'products', 'product_revisions',
      'media_assets', 'media_variants', 'contact_messages',
      'background_jobs', 'activity_logs'
  )`).Scan(&publicCount); err != nil {
		t.Fatalf("count public app tables: %v", err)
	}
	if publicCount != 0 {
		t.Fatalf("public app table count = %d, want 0", publicCount)
	}

	var appCount int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM information_schema.tables
WHERE table_schema = 'tauco_app'
  AND table_name IN (
      'pages', 'page_revisions', 'products', 'product_revisions',
      'media_assets', 'media_variants', 'contact_messages',
      'background_jobs', 'activity_logs'
  )`).Scan(&appCount); err != nil {
		t.Fatalf("count private app tables: %v", err)
	}
	if appCount != 9 {
		t.Fatalf("Phase 1B private app table count = %d, want 9", appCount)
	}

	var metadataCount int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM information_schema.tables
WHERE table_schema = 'tauco_app'
  AND table_name = 'schema_migrations'`).Scan(&metadataCount); err != nil {
		t.Fatalf("count migration metadata tables: %v", err)
	}
	if metadataCount != 1 {
		t.Fatalf("private schema_migrations count = %d, want 1", metadataCount)
	}
	var metadataOwner string
	if err := conn.QueryRow(ctx, `
SELECT tableowner
FROM pg_tables
WHERE schemaname = 'tauco_app'
  AND tablename = 'schema_migrations'`).Scan(&metadataOwner); err != nil {
		t.Fatalf("read migration metadata owner: %v", err)
	}
	if metadataOwner != MigratorRole {
		t.Fatalf("schema_migrations owner = %q, want %q", metadataOwner, MigratorRole)
	}

	var maxAttemptsDefault string
	if err := conn.QueryRow(ctx, `
SELECT column_default
FROM information_schema.columns
WHERE table_schema = 'tauco_app'
  AND table_name = 'background_jobs'
  AND column_name = 'max_attempts'`).Scan(&maxAttemptsDefault); err != nil {
		t.Fatalf("read background job max_attempts default: %v", err)
	}
	if maxAttemptsDefault != "8" {
		t.Fatalf("background job max_attempts default = %q, want 8", maxAttemptsDefault)
	}
}

func assertSingleMembership(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	memberName string,
	wantGrantedRole string,
) {
	t.Helper()

	var (
		grantedRole string
		adminOption bool
	)
	if err := conn.QueryRow(ctx, `
SELECT granted.rolname, membership.admin_option
FROM pg_auth_members AS membership
JOIN pg_roles AS member ON member.oid = membership.member
JOIN pg_roles AS granted ON granted.oid = membership.roleid
WHERE member.rolname = $1`, memberName).Scan(&grantedRole, &adminOption); err != nil {
		t.Fatalf("read membership for %s: %v", memberName, err)
	}
	if grantedRole != wantGrantedRole || adminOption {
		t.Fatalf(
			"membership for %s = role %s admin=%t, want %s admin=false",
			memberName,
			grantedRole,
			adminOption,
			wantGrantedRole,
		)
	}
}

func assertRoleSecurity(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	migrationLogin string,
	runtimeLogin string,
	adminLogin string,
) {
	t.Helper()

	for _, roleName := range []string{
		MigratorRole,
		RuntimeRole,
		AdminRuntimeRole,
		migrationLogin,
		runtimeLogin,
		adminLogin,
	} {
		var (
			canLogin    bool
			superuser   bool
			createDB    bool
			createRole  bool
			replication bool
			bypassRLS   bool
		)
		if err := conn.QueryRow(ctx, `
SELECT
    rolcanlogin,
    rolsuper,
    rolcreatedb,
    rolcreaterole,
    rolreplication,
    rolbypassrls
FROM pg_roles
WHERE rolname = $1`, roleName).Scan(
			&canLogin,
			&superuser,
			&createDB,
			&createRole,
			&replication,
			&bypassRLS,
		); err != nil {
			t.Fatalf("inspect role %s: %v", roleName, err)
		}
		if (roleName == runtimeLogin || roleName == migrationLogin || roleName == adminLogin) && !canLogin {
			t.Errorf("database login %s cannot login", roleName)
		}
		if roleName != runtimeLogin && roleName != migrationLogin && roleName != adminLogin && canLogin {
			t.Errorf("authorization role %s unexpectedly has LOGIN", roleName)
		}
		if superuser || createDB || createRole || replication || bypassRLS {
			t.Errorf("role %s has elevated attributes", roleName)
		}
	}

	type membership struct {
		member      string
		grantedRole string
		adminOption bool
	}
	rows, err := conn.Query(ctx, `
SELECT member.rolname, granted.rolname, membership.admin_option
FROM pg_auth_members AS membership
JOIN pg_roles AS member ON member.oid = membership.member
JOIN pg_roles AS granted ON granted.oid = membership.roleid
WHERE member.rolname IN ($1, $2, $3)
ORDER BY member.rolname, granted.rolname`, migrationLogin, runtimeLogin, adminLogin)
	if err != nil {
		t.Fatalf("query database role memberships: %v", err)
	}
	defer rows.Close()
	var memberships []membership
	for rows.Next() {
		var value membership
		if err := rows.Scan(&value.member, &value.grantedRole, &value.adminOption); err != nil {
			t.Fatalf("scan database role membership: %v", err)
		}
		memberships = append(memberships, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate database role memberships: %v", err)
	}
	if len(memberships) != 3 {
		t.Fatalf("database login memberships = %+v, want exactly three", memberships)
	}
	for _, value := range memberships {
		if value.adminOption {
			t.Errorf("membership %+v unexpectedly has admin option", value)
		}
		switch value.member {
		case migrationLogin:
			if value.grantedRole != MigratorRole {
				t.Errorf("migration login membership = %+v", value)
			}
		case runtimeLogin:
			if value.grantedRole != RuntimeRole {
				t.Errorf("runtime login membership = %+v", value)
			}
		case adminLogin:
			if value.grantedRole != AdminRuntimeRole {
				t.Errorf("admin login membership = %+v", value)
			}
		default:
			t.Errorf("unexpected membership = %+v", value)
		}
	}

	var nestedAuthorizationMemberships int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM pg_auth_members AS membership
JOIN pg_roles AS member ON member.oid = membership.member
WHERE member.rolname IN ($1, $2, $3)`, MigratorRole, RuntimeRole, AdminRuntimeRole).Scan(
		&nestedAuthorizationMemberships,
	); err != nil {
		t.Fatalf("count nested authorization memberships: %v", err)
	}
	if nestedAuthorizationMemberships != 0 {
		t.Fatalf(
			"nested authorization memberships = %d, want 0",
			nestedAuthorizationMemberships,
		)
	}
}

func assertAuthorizationDatabaseGrants(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	authorizationRole string,
	runtimeLogin string,
	databaseName string,
) {
	t.Helper()

	var authorizationRoleGrants int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM pg_database AS database
CROSS JOIN LATERAL aclexplode(database.datacl) AS acl
JOIN pg_roles AS role ON role.oid = acl.grantee
WHERE role.rolname = $1`, authorizationRole).Scan(&authorizationRoleGrants); err != nil {
		t.Fatalf("count runtime authorization database grants: %v", err)
	}
	if authorizationRoleGrants != 0 {
		t.Fatalf(
			"runtime authorization database grants = %d, want 0",
			authorizationRoleGrants,
		)
	}

	var (
		loginGrantCount int
		validGrantCount int
	)
	if err := conn.QueryRow(ctx, `
SELECT
    count(*),
    count(*) FILTER (
        WHERE database.datname = $2
          AND acl.privilege_type = 'CONNECT'
          AND NOT acl.is_grantable
    )
FROM pg_database AS database
CROSS JOIN LATERAL aclexplode(database.datacl) AS acl
JOIN pg_roles AS role ON role.oid = acl.grantee
WHERE role.rolname = $1`, runtimeLogin, databaseName).Scan(
		&loginGrantCount,
		&validGrantCount,
	); err != nil {
		t.Fatalf("inspect concrete runtime database grant: %v", err)
	}
	if loginGrantCount != 1 || validGrantCount != 1 {
		t.Fatalf(
			"runtime login database grants = %d valid=%d, want 1 valid=1",
			loginGrantCount,
			validGrantCount,
		)
	}
}

func assertRuntimePrivileges(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	var searchPath string
	if err := conn.QueryRow(ctx, "SHOW search_path").Scan(&searchPath); err != nil {
		t.Fatalf("read runtime search_path: %v", err)
	}
	if searchPath != "tauco_app, pg_catalog" {
		t.Fatalf("runtime search_path = %q", searchPath)
	}

	var count int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM tauco_app.products").Scan(&count); err != nil {
		t.Fatalf("runtime SELECT published storage: %v", err)
	}

	assertPostgresCode(t, "runtime create table", "42501", func() error {
		_, err := conn.Exec(ctx, "CREATE TABLE tauco_app.runtime_forbidden (id integer)")
		return err
	})
	assertPostgresCode(t, "runtime alter table", "42501", func() error {
		_, err := conn.Exec(ctx, "ALTER TABLE tauco_app.products ADD COLUMN forbidden integer")
		return err
	})
	assertPostgresCode(t, "runtime create temporary table", "42501", func() error {
		_, err := conn.Exec(ctx, "CREATE TEMPORARY TABLE runtime_temp_forbidden (id integer)")
		return err
	})
	assertPostgresCode(t, "runtime insert revision", "42501", func() error {
		_, err := conn.Exec(ctx, `
INSERT INTO tauco_app.page_revisions (
    id, page_id, revision_number, status, schema_version,
    content_json, content_checksum
) VALUES (
    '019bfc80-0000-7000-8000-000000009991',
    '019bfc80-0000-7000-8000-000000009992',
    1, 'draft', 1, '{}'::jsonb, repeat('a', 64)
)`)
		return err
	})

	// search_path is not a security boundary. Even if a session changes it,
	// schema ACLs still prevent public-schema or application-schema DDL.
	if _, err := conn.Exec(ctx, "SET search_path TO public, pg_catalog"); err != nil {
		t.Fatalf("change runtime search_path for privilege probe: %v", err)
	}
	assertPostgresCode(t, "runtime public create after search_path change", "42501", func() error {
		_, err := conn.Exec(ctx, "CREATE TABLE runtime_public_forbidden (id integer)")
		return err
	})

	_, err := conn.Exec(ctx, `
INSERT INTO tauco_app.contact_messages (
    id,
    idempotency_key_hash,
    request_payload_hash,
    name,
    email,
    phone,
    subject,
    message,
    privacy_consent,
    privacy_notice_version,
    consent_at,
    retention_delete_at,
    created_at
) VALUES (
    '019bfc80-0000-7000-8000-000000009901',
    repeat('a', 64),
    repeat('b', 64),
    'Integration Test',
    'integration@example.test',
    NULL,
    'Pertanyaan umum',
    'Pesan integration test memiliki panjang yang valid.',
    true,
    'phase-1b',
    statement_timestamp(),
    statement_timestamp() + interval '11 months',
    statement_timestamp()
)`)
	if err != nil {
		t.Fatalf("runtime insert contact message: %v", err)
	}
}

func assertAdminPrivileges(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	var searchPath string
	if err := conn.QueryRow(ctx, "SHOW search_path").Scan(&searchPath); err != nil {
		t.Fatalf("read admin search_path: %v", err)
	}
	if searchPath != "tauco_app, pg_catalog" {
		t.Fatalf("admin search_path = %q", searchPath)
	}

	var permissionCount int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM tauco_app.permissions").Scan(&permissionCount); err != nil {
		t.Fatalf("admin SELECT permissions: %v", err)
	}
	if permissionCount != 12 {
		t.Fatalf("admin permission count = %d, want 12", permissionCount)
	}

	if _, err := conn.Exec(ctx, `
INSERT INTO tauco_app.admin_users (id, email, password_hash)
VALUES (
    '019bfc80-0000-7000-8000-000000009701',
    'admin-privilege@example.test',
    '$argon2id$v=19$m=65536,t=3,p=2$abcdefghijklmnop$abcdefghijklmnopabcdefghijklmnopabcdefghijklmnop'
)`); err != nil {
		t.Fatalf("admin INSERT own user storage: %v", err)
	}
	if _, err := conn.Exec(ctx, `
UPDATE tauco_app.contact_messages
SET status = 'read'
WHERE id = '019bfc80-0000-7000-8000-000000009901'`); err != nil {
		t.Fatalf("admin UPDATE inbox status: %v", err)
	}

	assertPostgresCode(t, "admin create table", "42501", func() error {
		_, err := conn.Exec(ctx, "CREATE TABLE tauco_app.admin_forbidden (id integer)")
		return err
	})
	assertPostgresCode(t, "admin delete inbox", "42501", func() error {
		_, err := conn.Exec(ctx, "DELETE FROM tauco_app.contact_messages")
		return err
	})
	assertPostgresCode(t, "admin mutate permission catalog", "42501", func() error {
		_, err := conn.Exec(ctx, `
INSERT INTO tauco_app.permissions (id, key, description)
VALUES ('019bfc80-0000-7000-8000-000000009702', 'forbidden.write', 'Forbidden')`)
		return err
	})
}

func assertAdminCMSFoundation(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	var relationCount int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM information_schema.tables
WHERE table_schema = 'tauco_app'
  AND table_name IN (
      'admin_users', 'roles', 'permissions', 'user_roles', 'role_permissions',
      'admin_sessions', 'admin_refresh_tokens', 'mfa_credentials',
      'mfa_recovery_codes', 'page_revision_media', 'product_revision_media'
  )`).Scan(&relationCount); err != nil {
		t.Fatalf("count C1 foundation tables: %v", err)
	}
	if relationCount != 11 {
		t.Fatalf("C1 foundation table count = %d, want 11", relationCount)
	}

	var seededPermissionCount int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM tauco_app.role_permissions AS mapping
JOIN tauco_app.roles AS role ON role.id = mapping.role_id
WHERE role.key = 'super_admin'`).Scan(&seededPermissionCount); err != nil {
		t.Fatalf("count seeded super-admin permissions: %v", err)
	}
	if seededPermissionCount != 12 {
		t.Fatalf("super-admin permission count = %d, want 12", seededPermissionCount)
	}

	if _, err := conn.Exec(ctx, `
SET ROLE tauco_migrator;
INSERT INTO tauco_app.admin_users (id, email, password_hash)
VALUES (
    '019bfc80-0000-7000-8000-000000009601',
    'revision-owner@example.test',
    '$argon2id$v=19$m=65536,t=3,p=2$abcdefghijklmnop$abcdefghijklmnopabcdefghijklmnopabcdefghijklmnop'
);
INSERT INTO tauco_app.pages (id, key)
VALUES ('019bfc80-0000-7000-8000-000000009602', 'products');
INSERT INTO tauco_app.page_revisions (
    id, page_id, revision_number, status, schema_version,
    content_json, content_checksum, created_by
) VALUES (
    '019bfc80-0000-7000-8000-000000009603',
    '019bfc80-0000-7000-8000-000000009602',
    1, 'draft', 1, '{}'::jsonb, repeat('d', 64),
    '019bfc80-0000-7000-8000-000000009601'
);
RESET ROLE`); err != nil {
		t.Fatalf("insert C1 immutable revision fixture: %v", err)
	}

	assertPostgresCode(t, "draft revision update", "55000", func() error {
		_, err := conn.Exec(ctx, `
SET ROLE tauco_migrator;
UPDATE tauco_app.page_revisions
SET content_json = '{"mutated":true}'::jsonb
WHERE id = '019bfc80-0000-7000-8000-000000009603'`)
		return err
	})
	assertPostgresCode(t, "draft revision delete", "55000", func() error {
		_, err := conn.Exec(ctx, `
SET ROLE tauco_migrator;
DELETE FROM tauco_app.page_revisions
WHERE id = '019bfc80-0000-7000-8000-000000009603'`)
		return err
	})
}

func assertPublishedRevisionIntegrity(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin published revision fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET ROLE tauco_migrator"); err != nil {
		t.Fatalf("set migrator role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO tauco_app.pages (id, key)
VALUES
    ('019bfc80-0000-7000-8000-000000009001', 'home'),
    ('019bfc80-0000-7000-8000-000000009002', 'about'),
    ('019bfc80-0000-7000-8000-000000009003', 'tauco-guide');
INSERT INTO tauco_app.page_revisions (
    id, page_id, revision_number, status, schema_version,
    content_json, content_checksum, published_at
) VALUES
    (
        '019bfc80-0000-7000-8000-000000009011',
        '019bfc80-0000-7000-8000-000000009001',
        1, 'published', 1, '{}'::jsonb, repeat('a', 64), statement_timestamp()
    ),
    (
        '019bfc80-0000-7000-8000-000000009012',
        '019bfc80-0000-7000-8000-000000009002',
        1, 'published', 1, '{}'::jsonb, repeat('b', 64), statement_timestamp()
    ),
    (
        '019bfc80-0000-7000-8000-000000009013',
        '019bfc80-0000-7000-8000-000000009003',
        1, 'draft', 1, '{}'::jsonb, repeat('c', 64), NULL
    );
UPDATE tauco_app.pages
SET published_revision_id = '019bfc80-0000-7000-8000-000000009011'
WHERE id = '019bfc80-0000-7000-8000-000000009001'`); err != nil {
		t.Fatalf("insert published revision fixture: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit valid published pointer: %v", err)
	}

	assertPostgresCode(t, "published revision update", "55000", func() error {
		_, err := conn.Exec(ctx, `
SET ROLE tauco_migrator;
UPDATE tauco_app.page_revisions
SET content_json = '{"mutated":true}'::jsonb
WHERE id = '019bfc80-0000-7000-8000-000000009011';
RESET ROLE`)
		return err
	})

	crossTx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin cross-owner pointer transaction: %v", err)
	}
	defer crossTx.Rollback(ctx)
	if _, err := crossTx.Exec(ctx, `
SET ROLE tauco_migrator;
UPDATE tauco_app.pages
SET published_revision_id = '019bfc80-0000-7000-8000-000000009012'
WHERE id = '019bfc80-0000-7000-8000-000000009001'`); err != nil {
		t.Fatalf("stage cross-owner pointer: %v", err)
	}
	assertPostgresCode(t, "cross-owner published pointer", "23503", func() error {
		return crossTx.Commit(ctx)
	})

	draftTx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin draft pointer transaction: %v", err)
	}
	defer draftTx.Rollback(ctx)
	if _, err := draftTx.Exec(ctx, `
SET ROLE tauco_migrator;
UPDATE tauco_app.pages
SET published_revision_id = '019bfc80-0000-7000-8000-000000009013'
WHERE id = '019bfc80-0000-7000-8000-000000009003'`); err != nil {
		t.Fatalf("stage same-owner draft pointer: %v", err)
	}
	assertPostgresCode(t, "same-owner draft pointer", "23514", func() error {
		return draftTx.Commit(ctx)
	})
}

func assertActivityLogAppendOnly(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	if _, err := conn.Exec(ctx, `
SET ROLE tauco_migrator;
INSERT INTO tauco_app.activity_logs (
    id, event_type, entity_type, actor_type, metadata_json
) VALUES (
    '019bfc80-0000-7000-8000-000000009101',
    'migration.test',
    'migration',
    'system',
    '{}'::jsonb
);
RESET ROLE`); err != nil {
		t.Fatalf("insert activity log fixture: %v", err)
	}
	assertPostgresCode(t, "activity log update", "55000", func() error {
		_, err := conn.Exec(ctx, `
SET ROLE tauco_migrator;
UPDATE tauco_app.activity_logs
SET metadata_json = '{"mutated":true}'::jsonb
WHERE id = '019bfc80-0000-7000-8000-000000009101';
RESET ROLE`)
		return err
	})
}

func assertDomainTablesAbsent(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	var count int
	if err := conn.QueryRow(ctx, `
SELECT count(*)
FROM information_schema.tables
WHERE table_schema = 'tauco_app'
  AND table_name <> 'schema_migrations'`).Scan(&count); err != nil {
		t.Fatalf("count tables after down: %v", err)
	}
	if count != 0 {
		t.Fatalf("domain table count after down = %d, want 0", count)
	}
}

func assertMigrationVersion(t *testing.T, migrator *Migrator, want uint) {
	t.Helper()

	version, dirty, err := migrator.Version()
	if err != nil {
		t.Fatalf("migration Version() error = %v", err)
	}
	if version != want || dirty {
		t.Fatalf("migration version = %d dirty=%t, want %d false", version, dirty, want)
	}
}

func assertPostgresCode(t *testing.T, name, wantCode string, operation func() error) {
	t.Helper()

	err := operation()
	if err == nil {
		t.Fatalf("%s error = nil, want PostgreSQL code %s", name, wantCode)
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		t.Fatalf("%s error = %T %v, want *pgconn.PgError", name, err, err)
	}
	if postgresError.Code != wantCode {
		t.Fatalf("%s PostgreSQL code = %s, want %s: %v", name, postgresError.Code, wantCode, err)
	}
}

func replaceDatabaseAndUser(
	t *testing.T,
	rawURL,
	databaseName,
	username,
	password string,
) string {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	parsed.Path = "/" + databaseName
	parsed.RawPath = ""
	if username != "" {
		parsed.User = url.UserPassword(username, password)
	}
	query := parsed.Query()
	query.Del("search_path")
	query.Del("x-migrations-table")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
