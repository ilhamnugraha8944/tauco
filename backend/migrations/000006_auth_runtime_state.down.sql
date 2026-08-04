SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

DROP INDEX IF EXISTS tauco_app.mfa_credentials_active_idx;

ALTER TABLE tauco_app.mfa_credentials
    DROP CONSTRAINT mfa_credentials_time_order,
    DROP COLUMN revoked_at,
    ADD CONSTRAINT mfa_credentials_time_order CHECK (
        updated_at >= created_at
        AND (enabled_at IS NULL OR enabled_at >= created_at)
    );

ALTER TABLE tauco_app.admin_sessions
    DROP CONSTRAINT admin_sessions_authentication_level_allowed,
    DROP COLUMN authentication_level;

RESET ROLE;
