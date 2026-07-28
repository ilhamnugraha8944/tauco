-- Cluster-global roles and the private schema are provisioned out-of-band by
-- cmd/migrate when MIGRATION_BOOTSTRAP_ROLES=true. A managed database may
-- pre-provision the same contract and disable local role creation.
SET search_path TO tauco_app, pg_catalog;

-- golang-migrate creates this table before version 1. Stable group ownership
-- lets non-super migration LOGIN credentials rotate safely.
ALTER TABLE tauco_app.schema_migrations OWNER TO tauco_migrator;

SET ROLE tauco_migrator;

REVOKE ALL ON tauco_app.schema_migrations FROM PUBLIC;
REVOKE ALL ON tauco_app.schema_migrations FROM tauco_runtime;

REVOKE ALL ON SCHEMA tauco_app FROM PUBLIC;
REVOKE ALL ON SCHEMA tauco_app FROM tauco_runtime;
REVOKE ALL ON SCHEMA tauco_app FROM tauco_migrator;

GRANT USAGE, CREATE ON SCHEMA tauco_app TO tauco_migrator;
GRANT USAGE ON SCHEMA tauco_app TO tauco_runtime;

RESET ROLE;
