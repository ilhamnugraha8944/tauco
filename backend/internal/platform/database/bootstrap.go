package database

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/pbkdf2"
)

const (
	scramIterations = 4096
	scramSaltBytes  = 16
)

// BootstrapRoles provisions the fixed authorization roles, private schema, and
// a dedicated runtime LOGIN. It is intended for local/owned PostgreSQL only.
// Managed environments may pre-provision this contract and set
// MIGRATION_BOOTSTRAP_ROLES=false.
func BootstrapRoles(ctx context.Context, cfg MigrationConfig) error {
	if !cfg.BootstrapRoles {
		return nil
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	runtime, err := parseDatabaseURL("DATABASE_URL", cfg.RuntimeURL, true)
	if err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, cfg.MigrationURL)
	if err != nil {
		return fmt.Errorf("connect for database role bootstrap: %w", err)
	}
	defer conn.Close(ctx)

	if err := createAuthorizationRoles(ctx, conn); err != nil {
		return err
	}
	if err := revokeRuntimeAuthorizationDatabaseGrants(ctx, conn); err != nil {
		return err
	}
	if err := assertRuntimeAuthorizationRoleSafe(ctx, conn); err != nil {
		return err
	}
	if err := grantMigratorMembership(ctx, conn); err != nil {
		return err
	}
	if err := createPrivateSchema(ctx, conn); err != nil {
		return err
	}
	if err := hardenLocalDatabasePrivileges(ctx, conn, runtime.database); err != nil {
		return err
	}
	if err := assertRuntimeAuthorizationMembersSafe(
		ctx,
		conn,
		runtime.username,
		runtime.database,
	); err != nil {
		return err
	}
	if err := createRuntimeLogin(ctx, conn, runtime); err != nil {
		return err
	}
	return nil
}

func assertRuntimeAuthorizationMembersSafe(
	ctx context.Context,
	conn *pgx.Conn,
	expectedRuntimeLogin string,
	databaseName string,
) error {
	var unsafeMember bool
	if err := conn.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM pg_auth_members AS membership
    JOIN pg_roles AS granted ON granted.oid = membership.roleid
    JOIN pg_roles AS member ON member.oid = membership.member
    WHERE granted.rolname = $1
      AND member.rolname <> $2
      AND (
          membership.admin_option
          OR NOT member.rolcanlogin
          OR member.rolsuper
          OR member.rolcreatedb
          OR member.rolcreaterole
          OR member.rolreplication
          OR member.rolbypassrls
          OR has_database_privilege(member.oid, $3, 'CONNECT')
          OR EXISTS (
              SELECT 1
              FROM pg_stat_activity AS activity
              WHERE activity.datname = $3
                AND activity.usename = member.rolname
                AND activity.pid <> pg_backend_pid()
          )
      )
)`, RuntimeRole, expectedRuntimeLogin, databaseName).Scan(&unsafeMember); err != nil {
		return fmt.Errorf("inspect runtime authorization members: %w", err)
	}
	if unsafeMember {
		return fmt.Errorf(
			"refuse to bootstrap database %q with an unsafe %s member",
			databaseName,
			RuntimeRole,
		)
	}
	return nil
}

func createAuthorizationRoles(ctx context.Context, conn *pgx.Conn) error {
	if err := assertRoleSafeForBootstrap(ctx, conn, MigratorRole, false); err != nil {
		return err
	}
	if err := assertRoleSafeForBootstrap(ctx, conn, RuntimeRole, false); err != nil {
		return err
	}

	statement := `
DO $bootstrap$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tauco_migrator') THEN
        CREATE ROLE tauco_migrator
            NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
            NOREPLICATION NOBYPASSRLS INHERIT;
        COMMENT ON ROLE tauco_migrator IS
            'Tauco schema owner; authentication is provisioned separately';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tauco_runtime') THEN
        CREATE ROLE tauco_runtime
            NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
            NOREPLICATION NOBYPASSRLS INHERIT;
        COMMENT ON ROLE tauco_runtime IS
            'Tauco least-privilege application authorization role';
    END IF;
END
$bootstrap$`
	if _, err := conn.Exec(ctx, statement); err != nil {
		return fmt.Errorf("provision fixed database authorization roles: %w", err)
	}

	// Reassert security attributes even when roles were pre-existing.
	if _, err := conn.Exec(ctx, `
ALTER ROLE tauco_migrator
    WITH NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
         NOREPLICATION NOBYPASSRLS INHERIT;
ALTER ROLE tauco_runtime
    WITH NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
         NOREPLICATION NOBYPASSRLS INHERIT`); err != nil {
		return fmt.Errorf("harden fixed database authorization roles: %w", err)
	}
	return nil
}

func revokeRuntimeAuthorizationDatabaseGrants(
	ctx context.Context,
	conn *pgx.Conn,
) error {
	rows, err := conn.Query(ctx, `
SELECT format('REVOKE ALL ON DATABASE %I FROM tauco_runtime', database.datname)
FROM pg_database AS database
CROSS JOIN LATERAL aclexplode(database.datacl) AS acl
JOIN pg_roles AS role ON role.oid = acl.grantee
WHERE role.rolname = 'tauco_runtime'
GROUP BY database.datname`)
	if err != nil {
		return fmt.Errorf("inspect runtime authorization database grants: %w", err)
	}

	var statements []string
	for rows.Next() {
		var statement string
		if err := rows.Scan(&statement); err != nil {
			rows.Close()
			return fmt.Errorf("read runtime authorization database revoke: %w", err)
		}
		statements = append(statements, statement)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read runtime authorization database grants: %w", err)
	}
	rows.Close()

	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("revoke runtime authorization database grant: %w", err)
		}
	}
	return nil
}

func assertRuntimeAuthorizationRoleSafe(ctx context.Context, conn *pgx.Conn) error {
	var ownsObjects bool
	if err := conn.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM pg_roles AS role
    WHERE role.rolname = $1
      AND (
          EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner = role.oid)
          OR EXISTS (SELECT 1 FROM pg_class WHERE relowner = role.oid)
          OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner = role.oid)
          OR EXISTS (SELECT 1 FROM pg_default_acl WHERE defaclrole = role.oid)
          OR EXISTS (SELECT 1 FROM pg_database WHERE datdba = role.oid)
      )
)`, RuntimeRole).Scan(&ownsObjects); err != nil {
		return fmt.Errorf("inspect runtime authorization role ownership: %w", err)
	}
	if ownsObjects {
		return fmt.Errorf("refuse to reuse %s because it owns database objects", RuntimeRole)
	}

	var unexpectedGrant bool
	if err := conn.QueryRow(ctx, `
WITH runtime_role AS (
    SELECT oid
    FROM pg_roles
    WHERE rolname = $1
)
SELECT
    EXISTS (
        SELECT 1
        FROM pg_database AS database
        CROSS JOIN LATERAL aclexplode(database.datacl) AS acl
        CROSS JOIN runtime_role
        WHERE acl.grantee = runtime_role.oid
    )
    OR EXISTS (
        SELECT 1
        FROM pg_namespace AS namespace
        CROSS JOIN LATERAL aclexplode(namespace.nspacl) AS acl
        CROSS JOIN runtime_role
        WHERE acl.grantee = runtime_role.oid
          AND NOT (
              namespace.nspname = 'tauco_app'
              AND acl.privilege_type = 'USAGE'
              AND NOT acl.is_grantable
          )
    )
    OR EXISTS (
        SELECT 1
        FROM pg_class AS relation
        JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
        CROSS JOIN LATERAL aclexplode(relation.relacl) AS acl
        CROSS JOIN runtime_role
        WHERE acl.grantee = runtime_role.oid
          AND NOT (
              namespace.nspname = 'tauco_app'
              AND NOT acl.is_grantable
              AND (
                  (
                      relation.relname IN (
                          'pages', 'page_revisions', 'products',
                          'product_revisions', 'media_assets', 'media_variants'
                      )
                      AND acl.privilege_type = 'SELECT'
                  )
                  OR (
                      relation.relname = 'contact_messages'
                      AND acl.privilege_type IN ('SELECT', 'INSERT')
                  )
                  OR (
                      relation.relname = 'background_jobs'
                      AND acl.privilege_type IN ('SELECT', 'INSERT', 'UPDATE')
                  )
                  OR (
                      relation.relname = 'activity_logs'
                      AND acl.privilege_type IN ('SELECT', 'INSERT')
                  )
              )
          )
    )
    OR EXISTS (
        SELECT 1
        FROM pg_proc AS procedure
        JOIN pg_namespace AS namespace ON namespace.oid = procedure.pronamespace
        CROSS JOIN LATERAL aclexplode(procedure.proacl) AS acl
        CROSS JOIN runtime_role
        WHERE acl.grantee = runtime_role.oid
          AND NOT (
              namespace.nspname = 'tauco_app'
              AND procedure.proname IN (
                  'tauco_set_updated_at',
                  'tauco_reject_published_revision_mutation',
                  'tauco_assert_page_published_revision',
                  'tauco_assert_product_published_revision',
                  'tauco_reject_activity_log_mutation'
              )
              AND procedure.pronargs = 0
              AND procedure.prorettype = 'trigger'::regtype
              AND acl.privilege_type = 'EXECUTE'
              AND NOT acl.is_grantable
          )
    )`, RuntimeRole).Scan(&unexpectedGrant); err != nil {
		return fmt.Errorf("inspect runtime authorization role grants: %w", err)
	}
	if unexpectedGrant {
		return fmt.Errorf("refuse to reuse %s with unexpected direct grants", RuntimeRole)
	}
	return nil
}

func grantMigratorMembership(ctx context.Context, conn *pgx.Conn) error {
	var statement string
	if err := conn.QueryRow(ctx, `
SELECT format(
    'GRANT tauco_migrator TO %I',
    current_user
)`).Scan(&statement); err != nil {
		return fmt.Errorf("prepare migrator membership: %w", err)
	}
	if _, err := conn.Exec(ctx, statement); err != nil {
		return fmt.Errorf("grant migrator membership: %w", err)
	}
	return nil
}

func hardenLocalDatabasePrivileges(
	ctx context.Context,
	conn *pgx.Conn,
	databaseName string,
) error {
	databaseIdentifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := conn.Exec(ctx,
		"REVOKE CONNECT, TEMPORARY ON DATABASE "+databaseIdentifier+" FROM PUBLIC"+
			"; REVOKE ALL ON DATABASE "+databaseIdentifier+" FROM "+RuntimeRole+
			"; GRANT CONNECT, TEMPORARY ON DATABASE "+databaseIdentifier+" TO "+MigratorRole,
	); err != nil {
		return fmt.Errorf("harden local database privileges: %w", err)
	}
	return nil
}

func createPrivateSchema(ctx context.Context, conn *pgx.Conn) error {
	if _, err := conn.Exec(ctx, `
CREATE SCHEMA IF NOT EXISTS tauco_app AUTHORIZATION tauco_migrator;
ALTER SCHEMA tauco_app OWNER TO tauco_migrator;
REVOKE ALL ON SCHEMA tauco_app FROM PUBLIC;
REVOKE ALL ON SCHEMA tauco_app FROM tauco_runtime;
REVOKE ALL ON SCHEMA public FROM tauco_runtime;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA tauco_app TO tauco_migrator;
GRANT USAGE ON SCHEMA tauco_app TO tauco_runtime`); err != nil {
		return fmt.Errorf("provision private application schema: %w", err)
	}
	return nil
}

func createRuntimeLogin(ctx context.Context, conn *pgx.Conn, runtime databaseEndpoint) error {
	identifier := pgx.Identifier{runtime.username}.Sanitize()
	verifier, err := newSCRAMVerifier(runtime.password)
	if err != nil {
		return fmt.Errorf("derive runtime role verifier: %w", err)
	}

	var exists bool
	if err := conn.QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)",
		runtime.username,
	).Scan(&exists); err != nil {
		return fmt.Errorf("inspect runtime login role: %w", err)
	}
	if !exists {
		if _, err := conn.Exec(ctx, "CREATE ROLE "+identifier); err != nil {
			return fmt.Errorf("create runtime login role: %w", err)
		}
	} else if err := assertRoleSafeForBootstrap(ctx, conn, runtime.username, true); err != nil {
		return err
	} else if err := assertRuntimeRoleHasNoUnexpectedAccess(
		ctx,
		conn,
		runtime,
	); err != nil {
		return err
	}

	// The verifier contains only the fixed SCRAM alphabet and is never logged.
	if _, err := conn.Exec(ctx,
		"ALTER ROLE "+identifier+
			" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE"+
			" NOREPLICATION NOBYPASSRLS INHERIT PASSWORD '"+verifier+"'",
	); err != nil {
		return fmt.Errorf("harden runtime login role: %w", err)
	}
	if _, err := conn.Exec(ctx,
		"REVOKE "+MigratorRole+" FROM "+identifier+
			"; REVOKE ADMIN OPTION FOR "+RuntimeRole+" FROM "+identifier+
			"; GRANT "+RuntimeRole+" TO "+identifier,
	); err != nil {
		return fmt.Errorf("grant runtime authorization membership: %w", err)
	}

	databaseIdentifier := pgx.Identifier{runtime.database}.Sanitize()
	if _, err := conn.Exec(ctx,
		"REVOKE ALL ON DATABASE "+databaseIdentifier+" FROM "+identifier+
			"; GRANT CONNECT ON DATABASE "+databaseIdentifier+" TO "+identifier,
	); err != nil {
		return fmt.Errorf("grant runtime login database connection: %w", err)
	}
	if _, err := conn.Exec(ctx,
		"ALTER ROLE "+identifier+" IN DATABASE "+databaseIdentifier+
			" SET search_path TO "+ApplicationSchema+", pg_catalog",
	); err != nil {
		return fmt.Errorf("set runtime role search path: %w", err)
	}

	if _, err := conn.Exec(ctx,
		"REVOKE ALL ON SCHEMA "+ApplicationSchema+" FROM "+identifier,
	); err != nil {
		return fmt.Errorf("remove direct runtime schema grants: %w", err)
	}
	return nil
}

func assertRuntimeRoleHasNoUnexpectedAccess(
	ctx context.Context,
	conn *pgx.Conn,
	runtime databaseEndpoint,
) error {
	roleName := runtime.username
	rows, err := conn.Query(ctx, `
SELECT granted.rolname
FROM pg_auth_members AS membership
JOIN pg_roles AS member ON member.oid = membership.member
JOIN pg_roles AS granted ON granted.oid = membership.roleid
WHERE member.rolname = $1
  AND granted.rolname <> $2`, roleName, RuntimeRole)
	if err != nil {
		return fmt.Errorf("inspect runtime login memberships: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf(
			"refuse to mutate runtime role %q with unexpected memberships",
			roleName,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read runtime login memberships: %w", err)
	}
	rows.Close()

	var ownsObjects bool
	if err := conn.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM pg_roles AS role
    WHERE role.rolname = $1
      AND (
          EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner = role.oid)
          OR EXISTS (SELECT 1 FROM pg_class WHERE relowner = role.oid)
          OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner = role.oid)
          OR EXISTS (SELECT 1 FROM pg_default_acl WHERE defaclrole = role.oid)
          OR EXISTS (SELECT 1 FROM pg_database WHERE datdba = role.oid)
      )
)`, roleName).Scan(&ownsObjects); err != nil {
		return fmt.Errorf("inspect runtime login ownership: %w", err)
	}
	if ownsObjects {
		return fmt.Errorf("refuse to mutate runtime role %q that owns database objects", roleName)
	}

	var hasDirectGrants bool
	if err := conn.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM pg_roles AS role
    WHERE role.rolname = $1
      AND (
          EXISTS (
              SELECT 1
              FROM pg_namespace AS namespace
              CROSS JOIN LATERAL aclexplode(namespace.nspacl) AS acl
              WHERE acl.grantee = role.oid
          )
          OR EXISTS (
              SELECT 1
              FROM pg_class AS relation
              CROSS JOIN LATERAL aclexplode(relation.relacl) AS acl
              WHERE acl.grantee = role.oid
          )
          OR EXISTS (
              SELECT 1
              FROM pg_proc AS procedure
              CROSS JOIN LATERAL aclexplode(procedure.proacl) AS acl
              WHERE acl.grantee = role.oid
          )
          OR EXISTS (
              SELECT 1
              FROM pg_database AS database
              CROSS JOIN LATERAL aclexplode(database.datacl) AS acl
              WHERE acl.grantee = role.oid
                AND NOT (
                    database.datname = $2
                    AND acl.privilege_type = 'CONNECT'
                    AND NOT acl.is_grantable
                )
          )
      )
)`, roleName, runtime.database).Scan(&hasDirectGrants); err != nil {
		return fmt.Errorf("inspect runtime login direct grants: %w", err)
	}
	if hasDirectGrants {
		return fmt.Errorf("refuse to mutate runtime role %q with direct grants", roleName)
	}
	return nil
}

func assertRoleSafeForBootstrap(
	ctx context.Context,
	conn *pgx.Conn,
	roleName string,
	allowLogin bool,
) error {
	var (
		exists      bool
		canLogin    bool
		superuser   bool
		createDB    bool
		createRole  bool
		replication bool
		bypassRLS   bool
	)
	err := conn.QueryRow(ctx, `
SELECT
    true,
    rolcanlogin,
    rolsuper,
    rolcreatedb,
    rolcreaterole,
    rolreplication,
    rolbypassrls
FROM pg_roles
WHERE rolname = $1`, roleName).Scan(
		&exists,
		&canLogin,
		&superuser,
		&createDB,
		&createRole,
		&replication,
		&bypassRLS,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return fmt.Errorf("inspect role before bootstrap: %w", err)
	}
	if !exists {
		return nil
	}
	if superuser || createDB || createRole || replication || bypassRLS {
		return fmt.Errorf("refuse to mutate privileged database role %q", roleName)
	}
	if canLogin && !allowLogin {
		return fmt.Errorf("refuse to convert LOGIN role %q into an authorization role", roleName)
	}
	if !allowLogin {
		var inheritedMembership bool
		if err := conn.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM pg_auth_members AS membership
    JOIN pg_roles AS member ON member.oid = membership.member
    WHERE member.rolname = $1
)`, roleName).Scan(&inheritedMembership); err != nil {
			return fmt.Errorf("inspect authorization role memberships: %w", err)
		}
		if inheritedMembership {
			return fmt.Errorf(
				"refuse to reuse authorization role %q with inherited memberships",
				roleName,
			)
		}
	}
	return nil
}

func newSCRAMVerifier(password string) (string, error) {
	salt := make([]byte, scramSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	saltedPassword := pbkdf2.Key(
		[]byte(password),
		salt,
		scramIterations,
		sha256.Size,
		sha256.New,
	)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacSHA256(saltedPassword, []byte("Server Key"))

	encode := base64.StdEncoding.EncodeToString
	return fmt.Sprintf(
		"SCRAM-SHA-256$%d:%s$%s:%s",
		scramIterations,
		encode(salt),
		encode(storedKey[:]),
		encode(serverKey),
	), nil
}

func hmacSHA256(key, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write(data)
	return hash.Sum(nil)
}
