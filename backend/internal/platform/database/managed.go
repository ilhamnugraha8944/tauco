package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	ManagedMigrationLogin = "tauco_migration_login"
	ManagedRuntimeLogin   = "tauco_public_login"
	ManagedAdminLogin     = "tauco_admin_login"
)

type ManagedPasswords struct {
	Migration string
	Runtime   string
	Admin     string
}

func ProvisionManagedRoles(ctx context.Context, rawURL string, passwords ManagedPasswords) error {
	owner, err := parseDatabaseURL("PROVISION_DATABASE_URL", strings.TrimSpace(rawURL), true)
	if err != nil {
		return err
	}
	if owner.port != "5432" || !secureSSLMode(owner.sslMode) {
		return &ConfigError{Field: "PROVISION_DATABASE_URL", Reason: "must use TLS and the session/direct port 5432"}
	}
	for field, password := range map[string]string{
		"migration password": passwords.Migration,
		"runtime password":   passwords.Runtime,
		"admin password":     passwords.Admin,
	} {
		if !isBootstrapPassword(password) {
			return &ConfigError{Field: field, Reason: "must be 16-256 printable ASCII characters"}
		}
	}

	conn, err := pgx.Connect(ctx, rawURL)
	if err != nil {
		return fmt.Errorf("connect for managed database provisioning: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	if err := createAuthorizationRoles(ctx, conn); err != nil {
		return err
	}
	if err := assertRuntimeAuthorizationRoleSafe(ctx, conn); err != nil {
		return err
	}
	if err := createManagedPrivateSchema(ctx, conn); err != nil {
		return err
	}

	logins := []struct {
		name, password, role string
	}{
		{ManagedMigrationLogin, passwords.Migration, MigratorRole},
		{ManagedRuntimeLogin, passwords.Runtime, RuntimeRole},
		{ManagedAdminLogin, passwords.Admin, AdminRuntimeRole},
	}
	for _, login := range logins {
		endpoint := databaseEndpoint{username: login.name, password: login.password, database: owner.database}
		if err := assertAuthorizationMembersSafe(ctx, conn, login.role, login.name, owner.database); err != nil {
			return err
		}
		if err := createRuntimeLogin(ctx, conn, endpoint, login.role); err != nil {
			return err
		}
	}
	return nil
}

func createManagedPrivateSchema(ctx context.Context, conn *pgx.Conn) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin managed schema provisioning: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentUser, currentDatabase string
	if err := tx.QueryRow(ctx, "SELECT current_user, current_database()").Scan(&currentUser, &currentDatabase); err != nil {
		return fmt.Errorf("read managed provisioning role: %w", err)
	}
	currentUserIdentifier := pgx.Identifier{currentUser}.Sanitize()
	currentDatabaseIdentifier := pgx.Identifier{currentDatabase}.Sanitize()
	if _, err := tx.Exec(ctx, "GRANT "+MigratorRole+" TO "+currentUserIdentifier); err != nil {
		return fmt.Errorf("temporarily grant managed schema ownership role: %w", err)
	}
	if _, err := tx.Exec(ctx, "GRANT CREATE ON DATABASE "+currentDatabaseIdentifier+" TO "+MigratorRole); err != nil {
		return fmt.Errorf("temporarily grant managed schema create privilege: %w", err)
	}

	_, err = tx.Exec(ctx, `
CREATE SCHEMA IF NOT EXISTS tauco_app AUTHORIZATION tauco_migrator;
ALTER SCHEMA tauco_app OWNER TO tauco_migrator;
REVOKE ALL ON SCHEMA tauco_app FROM PUBLIC;
REVOKE ALL ON SCHEMA tauco_app FROM tauco_runtime;
REVOKE ALL ON SCHEMA tauco_app FROM tauco_admin_runtime;
GRANT USAGE, CREATE ON SCHEMA tauco_app TO tauco_migrator;
GRANT USAGE ON SCHEMA tauco_app TO tauco_runtime;
GRANT USAGE ON SCHEMA tauco_app TO tauco_admin_runtime`)
	if err != nil {
		return fmt.Errorf("provision managed private application schema: %w", err)
	}
	if _, err := tx.Exec(ctx, "REVOKE CREATE ON DATABASE "+currentDatabaseIdentifier+" FROM "+MigratorRole); err != nil {
		return fmt.Errorf("remove temporary managed schema create privilege: %w", err)
	}
	if _, err := tx.Exec(ctx, "REVOKE "+MigratorRole+" FROM "+currentUserIdentifier); err != nil {
		return fmt.Errorf("remove temporary managed schema ownership role: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit managed schema provisioning: %w", err)
	}
	return nil
}
