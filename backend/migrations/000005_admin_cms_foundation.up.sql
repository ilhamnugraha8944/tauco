SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

CREATE TABLE tauco_app.admin_users (
    id uuid PRIMARY KEY,
    email text NOT NULL,
    password_hash text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    last_login_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT admin_users_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT admin_users_email_canonical
        CHECK (
            char_length(email) BETWEEN 3 AND 160
            AND email = lower(email)
            AND btrim(email) = email
            AND email ~ '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
        ),
    CONSTRAINT admin_users_email_unique UNIQUE (email),
    CONSTRAINT admin_users_password_argon2id
        CHECK (
            char_length(password_hash) BETWEEN 60 AND 512
            AND password_hash LIKE '$argon2id$%'
        ),
    CONSTRAINT admin_users_status_allowed CHECK (status IN ('active', 'disabled')),
    CONSTRAINT admin_users_last_login_after_created
        CHECK (last_login_at IS NULL OR last_login_at >= created_at),
    CONSTRAINT admin_users_updated_after_created CHECK (updated_at >= created_at)
);

CREATE TABLE tauco_app.roles (
    id uuid PRIMARY KEY,
    key text NOT NULL,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT roles_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT roles_key_format
        CHECK (char_length(key) BETWEEN 3 AND 80 AND key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT roles_key_unique UNIQUE (key),
    CONSTRAINT roles_description_canonical
        CHECK (char_length(description) BETWEEN 3 AND 200 AND btrim(description) = description)
);

CREATE TABLE tauco_app.permissions (
    id uuid PRIMARY KEY,
    key text NOT NULL,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT permissions_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT permissions_key_format
        CHECK (char_length(key) BETWEEN 3 AND 100 AND key ~ '^[a-z][a-z0-9]*([.:_-][a-z0-9]+)*$'),
    CONSTRAINT permissions_key_unique UNIQUE (key),
    CONSTRAINT permissions_description_canonical
        CHECK (char_length(description) BETWEEN 3 AND 200 AND btrim(description) = description)
);

CREATE TABLE tauco_app.user_roles (
    admin_user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (admin_user_id, role_id),
    CONSTRAINT user_roles_user_fk
        FOREIGN KEY (admin_user_id) REFERENCES tauco_app.admin_users (id) ON DELETE CASCADE,
    CONSTRAINT user_roles_role_fk
        FOREIGN KEY (role_id) REFERENCES tauco_app.roles (id) ON DELETE RESTRICT
);

CREATE TABLE tauco_app.role_permissions (
    role_id uuid NOT NULL,
    permission_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT role_permissions_role_fk
        FOREIGN KEY (role_id) REFERENCES tauco_app.roles (id) ON DELETE CASCADE,
    CONSTRAINT role_permissions_permission_fk
        FOREIGN KEY (permission_id) REFERENCES tauco_app.permissions (id) ON DELETE RESTRICT
);

INSERT INTO tauco_app.roles (id, key, description) VALUES
    ('019cf000-0000-7000-8000-000000000001', 'super_admin', 'Akses penuh CMS Phase 1C');

INSERT INTO tauco_app.permissions (id, key, description) VALUES
    ('019cf000-0000-7000-8000-000000000101', 'account.manage', 'Kelola akun admin sendiri'),
    ('019cf000-0000-7000-8000-000000000102', 'content.read', 'Baca konten CMS'),
    ('019cf000-0000-7000-8000-000000000103', 'content.write', 'Buat draft konten'),
    ('019cf000-0000-7000-8000-000000000104', 'content.publish', 'Publikasikan konten'),
    ('019cf000-0000-7000-8000-000000000105', 'product.read', 'Baca produk CMS'),
    ('019cf000-0000-7000-8000-000000000106', 'product.write', 'Kelola draft produk'),
    ('019cf000-0000-7000-8000-000000000107', 'product.publish', 'Publikasikan produk'),
    ('019cf000-0000-7000-8000-000000000108', 'media.read', 'Baca pustaka media'),
    ('019cf000-0000-7000-8000-000000000109', 'media.write', 'Unggah dan retry media'),
    ('019cf000-0000-7000-8000-000000000110', 'inbox.read', 'Baca inbox kontak'),
    ('019cf000-0000-7000-8000-000000000111', 'inbox.write', 'Ubah status inbox kontak'),
    ('019cf000-0000-7000-8000-000000000112', 'activity.read', 'Baca activity log');

INSERT INTO tauco_app.role_permissions (role_id, permission_id)
SELECT '019cf000-0000-7000-8000-000000000001'::uuid, id
FROM tauco_app.permissions;

CREATE TABLE tauco_app.admin_sessions (
    id uuid PRIMARY KEY,
    admin_user_id uuid NOT NULL,
    status text NOT NULL DEFAULT 'active',
    csrf_token_hash text NOT NULL,
    user_agent_hash text,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    last_used_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    revoke_reason text,
    CONSTRAINT admin_sessions_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT admin_sessions_user_fk
        FOREIGN KEY (admin_user_id) REFERENCES tauco_app.admin_users (id) ON DELETE CASCADE,
    CONSTRAINT admin_sessions_status_allowed CHECK (status IN ('active', 'revoked', 'expired')),
    CONSTRAINT admin_sessions_csrf_hash_sha256 CHECK (csrf_token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT admin_sessions_user_agent_hash_sha256
        CHECK (user_agent_hash IS NULL OR user_agent_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT admin_sessions_time_order
        CHECK (last_used_at >= created_at AND expires_at > created_at),
    CONSTRAINT admin_sessions_revocation_state
        CHECK (
            (status = 'revoked' AND revoked_at IS NOT NULL AND revoke_reason IS NOT NULL)
            OR (status <> 'revoked' AND revoked_at IS NULL AND revoke_reason IS NULL)
        ),
    CONSTRAINT admin_sessions_revoke_reason_format
        CHECK (
            revoke_reason IS NULL
            OR (char_length(revoke_reason) BETWEEN 3 AND 80 AND revoke_reason ~ '^[A-Z][A-Z0-9_]*$')
        )
);

CREATE TABLE tauco_app.admin_refresh_tokens (
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL,
    token_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    revoked_at timestamptz,
    CONSTRAINT admin_refresh_tokens_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT admin_refresh_tokens_session_fk
        FOREIGN KEY (session_id) REFERENCES tauco_app.admin_sessions (id) ON DELETE CASCADE,
    CONSTRAINT admin_refresh_tokens_hash_sha256 CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT admin_refresh_tokens_hash_unique UNIQUE (token_hash),
    CONSTRAINT admin_refresh_tokens_time_order
        CHECK (
            expires_at > created_at
            AND (used_at IS NULL OR used_at >= created_at)
            AND (revoked_at IS NULL OR revoked_at >= created_at)
        )
);

CREATE TABLE tauco_app.mfa_credentials (
    admin_user_id uuid PRIMARY KEY,
    encrypted_secret bytea NOT NULL,
    encryption_key_id text NOT NULL,
    nonce bytea NOT NULL,
    last_used_step bigint,
    enabled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT mfa_credentials_user_fk
        FOREIGN KEY (admin_user_id) REFERENCES tauco_app.admin_users (id) ON DELETE CASCADE,
    CONSTRAINT mfa_credentials_secret_nonempty CHECK (octet_length(encrypted_secret) BETWEEN 16 AND 512),
    CONSTRAINT mfa_credentials_key_id_canonical
        CHECK (
            char_length(encryption_key_id) BETWEEN 1 AND 80
            AND encryption_key_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]*$'
        ),
    CONSTRAINT mfa_credentials_nonce_length CHECK (octet_length(nonce) = 12),
    CONSTRAINT mfa_credentials_last_step_nonnegative CHECK (last_used_step IS NULL OR last_used_step >= 0),
    CONSTRAINT mfa_credentials_time_order
        CHECK (
            updated_at >= created_at
            AND (enabled_at IS NULL OR enabled_at >= created_at)
        )
);

CREATE TABLE tauco_app.mfa_recovery_codes (
    id uuid PRIMARY KEY,
    admin_user_id uuid NOT NULL,
    code_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    used_at timestamptz,
	revoked_at timestamptz,
    CONSTRAINT mfa_recovery_codes_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT mfa_recovery_codes_user_fk
        FOREIGN KEY (admin_user_id) REFERENCES tauco_app.admin_users (id) ON DELETE CASCADE,
    CONSTRAINT mfa_recovery_codes_hash_sha256 CHECK (code_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT mfa_recovery_codes_hash_unique UNIQUE (admin_user_id, code_hash),
	CONSTRAINT mfa_recovery_codes_time_order CHECK (
		(used_at IS NULL OR used_at >= created_at)
		AND (revoked_at IS NULL OR revoked_at >= created_at)
		AND NOT (used_at IS NOT NULL AND revoked_at IS NOT NULL)
	)
);

CREATE TABLE tauco_app.page_revision_media (
    page_revision_id uuid NOT NULL,
    media_asset_id uuid NOT NULL,
    field_path text NOT NULL,
    position smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (page_revision_id, field_path, position),
    CONSTRAINT page_revision_media_revision_fk
        FOREIGN KEY (page_revision_id) REFERENCES tauco_app.page_revisions (id) ON DELETE CASCADE,
    CONSTRAINT page_revision_media_asset_fk
        FOREIGN KEY (media_asset_id) REFERENCES tauco_app.media_assets (id) ON DELETE RESTRICT,
    CONSTRAINT page_revision_media_field_path_format
        CHECK (
            char_length(field_path) BETWEEN 1 AND 200
            AND field_path ~ '^/[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$'
        ),
    CONSTRAINT page_revision_media_position_bounded CHECK (position BETWEEN 0 AND 100)
);

CREATE TABLE tauco_app.product_revision_media (
    product_revision_id uuid NOT NULL,
    media_asset_id uuid NOT NULL,
    field_path text NOT NULL,
    position smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    PRIMARY KEY (product_revision_id, field_path, position),
    CONSTRAINT product_revision_media_revision_fk
        FOREIGN KEY (product_revision_id) REFERENCES tauco_app.product_revisions (id) ON DELETE CASCADE,
    CONSTRAINT product_revision_media_asset_fk
        FOREIGN KEY (media_asset_id) REFERENCES tauco_app.media_assets (id) ON DELETE RESTRICT,
    CONSTRAINT product_revision_media_field_path_format
        CHECK (
            char_length(field_path) BETWEEN 1 AND 200
            AND field_path ~ '^/[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$'
        ),
    CONSTRAINT product_revision_media_position_bounded CHECK (position BETWEEN 0 AND 100)
);

ALTER TABLE tauco_app.products
    ADD COLUMN archived_at timestamptz,
    ADD CONSTRAINT products_archived_after_created
        CHECK (archived_at IS NULL OR archived_at >= created_at);

ALTER TABLE tauco_app.page_revisions
    ADD CONSTRAINT page_revisions_created_by_admin_fk
        FOREIGN KEY (created_by) REFERENCES tauco_app.admin_users (id) ON DELETE SET NULL;

ALTER TABLE tauco_app.product_revisions
    ADD CONSTRAINT product_revisions_created_by_admin_fk
        FOREIGN KEY (created_by) REFERENCES tauco_app.admin_users (id) ON DELETE SET NULL;

DROP TRIGGER page_revisions_reject_published_mutation ON tauco_app.page_revisions;
DROP TRIGGER product_revisions_reject_published_mutation ON tauco_app.product_revisions;
DROP FUNCTION tauco_app.tauco_reject_published_revision_mutation();

CREATE FUNCTION tauco_app.tauco_reject_revision_mutation()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, tauco_app
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'content revisions are immutable';
END
$$;

CREATE TRIGGER page_revisions_reject_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.page_revisions
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_revision_mutation();

CREATE TRIGGER product_revisions_reject_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.product_revisions
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_revision_mutation();

CREATE TRIGGER page_revision_media_reject_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.page_revision_media
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_revision_mutation();

CREATE TRIGGER product_revision_media_reject_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.product_revision_media
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_revision_mutation();

CREATE TRIGGER admin_users_set_updated_at
    BEFORE UPDATE ON tauco_app.admin_users
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE TRIGGER mfa_credentials_set_updated_at
    BEFORE UPDATE ON tauco_app.mfa_credentials
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE INDEX admin_sessions_user_status_idx
    ON tauco_app.admin_sessions (admin_user_id, status, expires_at DESC);
CREATE INDEX admin_refresh_tokens_session_idx
    ON tauco_app.admin_refresh_tokens (session_id, expires_at DESC);
CREATE INDEX mfa_recovery_codes_user_unused_idx
    ON tauco_app.mfa_recovery_codes (admin_user_id, created_at)
    WHERE used_at IS NULL AND revoked_at IS NULL;
CREATE INDEX page_revision_media_asset_idx
    ON tauco_app.page_revision_media (media_asset_id, page_revision_id);
CREATE INDEX product_revision_media_asset_idx
    ON tauco_app.product_revision_media (media_asset_id, product_revision_id);
CREATE INDEX products_admin_catalog_idx
    ON tauco_app.products (archived_at, sort_order, id);

REVOKE ALL ON
    tauco_app.admin_users,
    tauco_app.roles,
    tauco_app.permissions,
    tauco_app.user_roles,
    tauco_app.role_permissions,
    tauco_app.admin_sessions,
    tauco_app.admin_refresh_tokens,
    tauco_app.mfa_credentials,
    tauco_app.mfa_recovery_codes,
    tauco_app.page_revision_media,
    tauco_app.product_revision_media
FROM PUBLIC, tauco_runtime, tauco_admin_runtime;

GRANT SELECT, INSERT, UPDATE ON
    tauco_app.admin_users,
    tauco_app.admin_sessions,
    tauco_app.admin_refresh_tokens,
    tauco_app.mfa_credentials,
    tauco_app.mfa_recovery_codes
TO tauco_admin_runtime;

GRANT SELECT ON tauco_app.roles, tauco_app.permissions TO tauco_admin_runtime;
GRANT SELECT, INSERT ON tauco_app.user_roles, tauco_app.role_permissions TO tauco_admin_runtime;
GRANT SELECT, INSERT, UPDATE ON
    tauco_app.pages,
    tauco_app.products,
    tauco_app.media_assets,
    tauco_app.media_variants,
    tauco_app.contact_messages,
    tauco_app.background_jobs
TO tauco_admin_runtime;
GRANT SELECT, INSERT ON
    tauco_app.page_revisions,
    tauco_app.product_revisions,
    tauco_app.page_revision_media,
    tauco_app.product_revision_media,
    tauco_app.activity_logs
TO tauco_admin_runtime;

REVOKE ALL ON FUNCTION tauco_app.tauco_reject_revision_mutation() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_set_updated_at() TO tauco_admin_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_reject_revision_mutation() TO tauco_admin_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_assert_page_published_revision() TO tauco_admin_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_assert_product_published_revision() TO tauco_admin_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_reject_activity_log_mutation() TO tauco_admin_runtime;

RESET ROLE;
