SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

CREATE TABLE tauco_app.media_upload_intents (
    id uuid PRIMARY KEY,
    status text NOT NULL DEFAULT 'pending',
    quarantine_object_key text NOT NULL,
    expected_mime_type text NOT NULL,
    expected_bytes bigint NOT NULL,
    expected_sha256 text NOT NULL,
    alt_text text NOT NULL,
    decorative boolean NOT NULL DEFAULT false,
    created_by uuid NOT NULL,
    media_asset_id uuid,
    last_error_code text,
    expires_at timestamptz NOT NULL,
    queued_at timestamptz,
    completed_at timestamptz,
    failed_at timestamptz,
    expired_at timestamptz,
    cleanup_claimed_at timestamptz,
    quarantine_deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT media_upload_intents_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT media_upload_intents_status_allowed
        CHECK (status IN ('pending', 'queued', 'completed', 'failed', 'expired')),
    CONSTRAINT media_upload_intents_quarantine_key_unique
        UNIQUE (quarantine_object_key),
    CONSTRAINT media_upload_intents_quarantine_key_canonical
        CHECK (quarantine_object_key = 'quarantine/' || id::text),
    CONSTRAINT media_upload_intents_mime_allowed
        CHECK (expected_mime_type IN ('image/jpeg', 'image/png', 'image/webp')),
    CONSTRAINT media_upload_intents_bytes_bounded
        CHECK (expected_bytes BETWEEN 1 AND 10485760),
    CONSTRAINT media_upload_intents_sha256_format
        CHECK (expected_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT media_upload_intents_alt_text_semantics
        CHECK (
            (decorative AND alt_text = '')
            OR (
                NOT decorative
                AND char_length(alt_text) BETWEEN 1 AND 300
                AND btrim(alt_text) = alt_text
            )
        ),
    CONSTRAINT media_upload_intents_created_by_fk
        FOREIGN KEY (created_by) REFERENCES tauco_app.admin_users (id) ON DELETE RESTRICT,
    CONSTRAINT media_upload_intents_asset_fk
        FOREIGN KEY (media_asset_id) REFERENCES tauco_app.media_assets (id) ON DELETE RESTRICT,
    CONSTRAINT media_upload_intents_asset_unique UNIQUE (media_asset_id),
    CONSTRAINT media_upload_intents_error_code_format
        CHECK (
            last_error_code IS NULL
            OR (
                char_length(last_error_code) BETWEEN 3 AND 80
                AND last_error_code ~ '^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$'
            )
        ),
    CONSTRAINT media_upload_intents_expiry_window
        CHECK (
            expires_at > created_at
            AND expires_at <= created_at + interval '1 hour'
        ),
    CONSTRAINT media_upload_intents_lifecycle
        CHECK (
            (
                status = 'pending'
                AND queued_at IS NULL
                AND completed_at IS NULL
                AND failed_at IS NULL
                AND expired_at IS NULL
                AND media_asset_id IS NULL
                AND last_error_code IS NULL
            )
            OR (
                status = 'queued'
                AND queued_at IS NOT NULL
                AND completed_at IS NULL
                AND failed_at IS NULL
                AND expired_at IS NULL
                AND media_asset_id IS NULL
                AND last_error_code IS NULL
            )
            OR (
                status = 'completed'
                AND queued_at IS NOT NULL
                AND completed_at IS NOT NULL
                AND failed_at IS NULL
                AND expired_at IS NULL
                AND media_asset_id IS NOT NULL
                AND last_error_code IS NULL
            )
            OR (
                status = 'failed'
                AND queued_at IS NOT NULL
                AND completed_at IS NULL
                AND failed_at IS NOT NULL
                AND expired_at IS NULL
                AND media_asset_id IS NULL
                AND last_error_code IS NOT NULL
            )
            OR (
                status = 'expired'
                AND queued_at IS NULL
                AND completed_at IS NULL
                AND failed_at IS NULL
                AND expired_at IS NOT NULL
                AND media_asset_id IS NULL
                AND last_error_code IS NULL
            )
        ),
    CONSTRAINT media_upload_intents_timestamps_ordered
        CHECK (
            updated_at >= created_at
            AND (queued_at IS NULL OR queued_at >= created_at)
            AND (completed_at IS NULL OR completed_at >= queued_at)
            AND (failed_at IS NULL OR failed_at >= queued_at)
            AND (expired_at IS NULL OR expired_at >= created_at)
            AND (cleanup_claimed_at IS NULL OR cleanup_claimed_at >= created_at)
            AND (
                quarantine_deleted_at IS NULL
                OR quarantine_deleted_at >= cleanup_claimed_at
            )
        )
);

CREATE TRIGGER media_upload_intents_set_updated_at
    BEFORE UPDATE ON tauco_app.media_upload_intents
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE INDEX media_upload_intents_status_expiry_idx
    ON tauco_app.media_upload_intents (status, expires_at, id);
CREATE INDEX media_upload_intents_cleanup_idx
    ON tauco_app.media_upload_intents (expires_at, id)
    WHERE quarantine_deleted_at IS NULL;

REVOKE ALL ON tauco_app.media_upload_intents
FROM PUBLIC, tauco_runtime, tauco_admin_runtime;

GRANT SELECT, UPDATE ON tauco_app.media_upload_intents TO tauco_runtime;
GRANT SELECT, INSERT, UPDATE ON tauco_app.media_upload_intents TO tauco_admin_runtime;

RESET ROLE;
