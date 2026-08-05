SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

ALTER TABLE tauco_app.admin_sessions
    ADD COLUMN authentication_level text NOT NULL DEFAULT 'password',
    ADD CONSTRAINT admin_sessions_authentication_level_allowed
        CHECK (authentication_level IN ('password', 'mfa'));

ALTER TABLE tauco_app.mfa_credentials
    DROP CONSTRAINT mfa_credentials_time_order,
    ADD COLUMN revoked_at timestamptz,
    ADD CONSTRAINT mfa_credentials_time_order CHECK (
        updated_at >= created_at
        AND (enabled_at IS NULL OR enabled_at >= created_at)
        AND (revoked_at IS NULL OR revoked_at >= created_at)
        AND NOT (enabled_at IS NOT NULL AND revoked_at IS NOT NULL)
    );

CREATE INDEX mfa_credentials_active_idx
    ON tauco_app.mfa_credentials (admin_user_id)
    WHERE revoked_at IS NULL;

RESET ROLE;
