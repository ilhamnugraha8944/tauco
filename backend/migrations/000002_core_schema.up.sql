SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

CREATE TABLE tauco_app.pages (
    id uuid PRIMARY KEY,
    key text NOT NULL,
    published_revision_id uuid,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT pages_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT pages_key_allowed
        CHECK (key IN ('home', 'about', 'tauco-guide', 'products')),
    CONSTRAINT pages_key_unique UNIQUE (key),
    CONSTRAINT pages_updated_after_created
        CHECK (updated_at >= created_at)
);

CREATE TABLE tauco_app.page_revisions (
    id uuid PRIMARY KEY,
    page_id uuid NOT NULL,
    revision_number integer NOT NULL,
    status text NOT NULL,
    schema_version integer NOT NULL,
    content_json jsonb NOT NULL,
    content_checksum text NOT NULL,
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    published_at timestamptz,
    CONSTRAINT page_revisions_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT page_revisions_created_by_uuid_v7
        CHECK (
            created_by IS NULL
            OR created_by::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        ),
    CONSTRAINT page_revisions_page_fk
        FOREIGN KEY (page_id) REFERENCES tauco_app.pages (id) ON DELETE CASCADE,
    CONSTRAINT page_revisions_number_positive
        CHECK (revision_number > 0),
    CONSTRAINT page_revisions_status_allowed
        CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT page_revisions_schema_version_positive
        CHECK (schema_version > 0),
    CONSTRAINT page_revisions_content_object
        CHECK (jsonb_typeof(content_json) = 'object'),
    CONSTRAINT page_revisions_checksum_sha256
        CHECK (content_checksum ~ '^[0-9a-f]{64}$'),
    CONSTRAINT page_revisions_publication_state
        CHECK (
            (status = 'published' AND published_at IS NOT NULL)
            OR (status IN ('draft', 'archived') AND published_at IS NULL)
        ),
    CONSTRAINT page_revisions_published_after_created
        CHECK (published_at IS NULL OR published_at >= created_at),
    CONSTRAINT page_revisions_number_unique
        UNIQUE (page_id, revision_number),
    CONSTRAINT page_revisions_owner_identity_unique
        UNIQUE (page_id, id)
);

ALTER TABLE tauco_app.pages
    ADD CONSTRAINT pages_published_revision_same_owner_fk
    FOREIGN KEY (id, published_revision_id)
    REFERENCES tauco_app.page_revisions (page_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE tauco_app.products (
    id uuid PRIMARY KEY,
    slug text NOT NULL,
    sku text,
    sort_order integer NOT NULL DEFAULT 0,
    published_revision_id uuid,
    first_published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT products_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT products_slug_format
        CHECK (
            char_length(slug) BETWEEN 1 AND 80
            AND slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
        ),
    CONSTRAINT products_slug_unique UNIQUE (slug),
    CONSTRAINT products_sku_canonical
        CHECK (
            sku IS NULL
            OR (
                char_length(sku) BETWEEN 1 AND 80
                AND btrim(sku) = sku
            )
        ),
    CONSTRAINT products_sort_order_nonnegative
        CHECK (sort_order >= 0),
    CONSTRAINT products_first_published_after_created
        CHECK (first_published_at IS NULL OR first_published_at >= created_at),
    CONSTRAINT products_updated_after_created
        CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX products_sku_unique_when_present
    ON tauco_app.products (sku)
    WHERE sku IS NOT NULL;

CREATE TABLE tauco_app.product_revisions (
    id uuid PRIMARY KEY,
    product_id uuid NOT NULL,
    revision_number integer NOT NULL,
    status text NOT NULL,
    schema_version integer NOT NULL,
    content_json jsonb NOT NULL,
    content_checksum text NOT NULL,
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    published_at timestamptz,
    CONSTRAINT product_revisions_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT product_revisions_created_by_uuid_v7
        CHECK (
            created_by IS NULL
            OR created_by::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        ),
    CONSTRAINT product_revisions_product_fk
        FOREIGN KEY (product_id) REFERENCES tauco_app.products (id) ON DELETE CASCADE,
    CONSTRAINT product_revisions_number_positive
        CHECK (revision_number > 0),
    CONSTRAINT product_revisions_status_allowed
        CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT product_revisions_schema_version_positive
        CHECK (schema_version > 0),
    CONSTRAINT product_revisions_content_object
        CHECK (jsonb_typeof(content_json) = 'object'),
    CONSTRAINT product_revisions_checksum_sha256
        CHECK (content_checksum ~ '^[0-9a-f]{64}$'),
    CONSTRAINT product_revisions_publication_state
        CHECK (
            (status = 'published' AND published_at IS NOT NULL)
            OR (status IN ('draft', 'archived') AND published_at IS NULL)
        ),
    CONSTRAINT product_revisions_published_after_created
        CHECK (published_at IS NULL OR published_at >= created_at),
    CONSTRAINT product_revisions_number_unique
        UNIQUE (product_id, revision_number),
    CONSTRAINT product_revisions_owner_identity_unique
        UNIQUE (product_id, id)
);

ALTER TABLE tauco_app.products
    ADD CONSTRAINT products_published_revision_same_owner_fk
    FOREIGN KEY (id, published_revision_id)
    REFERENCES tauco_app.product_revisions (product_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE tauco_app.media_assets (
    id uuid PRIMARY KEY,
    status text NOT NULL,
    original_object_key text NOT NULL,
    original_mime_type text NOT NULL,
    original_width integer NOT NULL,
    original_height integer NOT NULL,
    original_bytes bigint NOT NULL,
    sha256 text NOT NULL,
    alt_text text NOT NULL,
    decorative boolean NOT NULL DEFAULT false,
    last_error_code text,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT media_assets_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT media_assets_status_allowed
        CHECK (status IN ('processing', 'ready', 'failed')),
    CONSTRAINT media_assets_object_key_unique UNIQUE (original_object_key),
    CONSTRAINT media_assets_object_key_canonical
        CHECK (
            char_length(original_object_key) BETWEEN 1 AND 1024
            AND btrim(original_object_key) = original_object_key
        ),
    CONSTRAINT media_assets_mime_type_image
        CHECK (
            char_length(original_mime_type) BETWEEN 7 AND 100
            AND original_mime_type ~ '^image/[a-z0-9.+-]+$'
        ),
    CONSTRAINT media_assets_dimensions_positive
        CHECK (original_width > 0 AND original_height > 0),
    CONSTRAINT media_assets_bytes_positive CHECK (original_bytes > 0),
    CONSTRAINT media_assets_sha256_format CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT media_assets_alt_text_semantics
        CHECK (
            (decorative AND alt_text = '')
            OR (
                NOT decorative
                AND char_length(alt_text) BETWEEN 1 AND 300
                AND btrim(alt_text) = alt_text
            )
        ),
    CONSTRAINT media_assets_error_code_format
        CHECK (
            last_error_code IS NULL
            OR (
                char_length(last_error_code) BETWEEN 3 AND 80
                AND last_error_code ~ '^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$'
            )
        ),
    CONSTRAINT media_assets_updated_after_created
        CHECK (updated_at >= created_at)
);

CREATE TABLE tauco_app.media_variants (
    id uuid PRIMARY KEY,
    media_asset_id uuid NOT NULL,
    width integer NOT NULL,
    height integer NOT NULL,
    format text NOT NULL,
    object_key text NOT NULL,
    mime_type text NOT NULL,
    bytes bigint NOT NULL,
    sha256 text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT media_variants_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT media_variants_asset_fk
        FOREIGN KEY (media_asset_id) REFERENCES tauco_app.media_assets (id) ON DELETE CASCADE,
    CONSTRAINT media_variants_dimensions_positive
        CHECK (width > 0 AND height > 0),
    CONSTRAINT media_variants_format_webp CHECK (format = 'webp'),
    CONSTRAINT media_variants_object_key_unique UNIQUE (object_key),
    CONSTRAINT media_variants_object_key_canonical
        CHECK (
            char_length(object_key) BETWEEN 1 AND 1024
            AND btrim(object_key) = object_key
        ),
    CONSTRAINT media_variants_mime_webp CHECK (mime_type = 'image/webp'),
    CONSTRAINT media_variants_bytes_positive CHECK (bytes > 0),
    CONSTRAINT media_variants_sha256_format CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT media_variants_dimension_unique
        UNIQUE (media_asset_id, width, format)
);

CREATE TABLE tauco_app.contact_messages (
    id uuid PRIMARY KEY,
    idempotency_key_hash text NOT NULL,
    request_payload_hash text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    phone text,
    subject text NOT NULL,
    message text NOT NULL,
    status text NOT NULL DEFAULT 'unread',
    privacy_consent boolean NOT NULL,
    privacy_notice_version text NOT NULL,
    consent_at timestamptz NOT NULL,
    retention_delete_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT contact_messages_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT contact_messages_idempotency_hash_unique
        UNIQUE (idempotency_key_hash),
    CONSTRAINT contact_messages_idempotency_hash_sha256
        CHECK (idempotency_key_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT contact_messages_payload_hash_sha256
        CHECK (request_payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT contact_messages_name_canonical
        CHECK (
            char_length(name) BETWEEN 2 AND 100
            AND btrim(name) = name
        ),
    CONSTRAINT contact_messages_email_canonical
        CHECK (
            char_length(email) BETWEEN 3 AND 160
            AND btrim(email) = email
            AND email ~ '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'
        ),
    CONSTRAINT contact_messages_phone_canonical
        CHECK (
            phone IS NULL
            OR (
                char_length(phone) BETWEEN 7 AND 30
                AND btrim(phone) = phone
                AND phone ~ '^[+0-9()-][+0-9() -]{5,28}[+0-9()-]$'
            )
        ),
    CONSTRAINT contact_messages_subject_allowed
        CHECK (
            subject IN (
                'Informasi produk',
                'Kerja sama dan distribusi',
                'Pertanyaan umum'
            )
        ),
    CONSTRAINT contact_messages_message_canonical
        CHECK (
            char_length(message) BETWEEN 20 AND 2000
            AND btrim(message) = message
        ),
    CONSTRAINT contact_messages_status_allowed
        CHECK (status IN ('unread', 'read', 'archived')),
    CONSTRAINT contact_messages_privacy_consent_true
        CHECK (privacy_consent),
    CONSTRAINT contact_messages_notice_version_canonical
        CHECK (
            char_length(privacy_notice_version) BETWEEN 1 AND 50
            AND privacy_notice_version ~ '^[a-z0-9]+([._-][a-z0-9]+)*$'
        ),
    CONSTRAINT contact_messages_consent_not_future
        CHECK (consent_at <= created_at),
    CONSTRAINT contact_messages_retention_window
        CHECK (
            retention_delete_at > consent_at
            AND retention_delete_at <= consent_at + interval '12 months'
        ),
    CONSTRAINT contact_messages_updated_after_created
        CHECK (updated_at >= created_at)
);

CREATE TABLE tauco_app.background_jobs (
    id uuid PRIMARY KEY,
    kind text NOT NULL,
    payload_json jsonb NOT NULL,
    idempotency_key text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    priority smallint NOT NULL DEFAULT 0,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 8,
    run_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    locked_at timestamptz,
    lock_owner text,
    lease_expires_at timestamptz,
    last_error_code text,
    last_error_message text,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    completed_at timestamptz,
    dead_at timestamptz,
    CONSTRAINT background_jobs_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT background_jobs_kind_format
        CHECK (
            char_length(kind) BETWEEN 3 AND 80
            AND kind ~ '^[a-z][a-z0-9]*([._-][a-z0-9]+)*$'
        ),
    CONSTRAINT background_jobs_payload_object
        CHECK (jsonb_typeof(payload_json) = 'object'),
    CONSTRAINT background_jobs_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT background_jobs_idempotency_key_canonical
        CHECK (
            char_length(idempotency_key) BETWEEN 16 AND 200
            AND btrim(idempotency_key) = idempotency_key
        ),
    CONSTRAINT background_jobs_status_allowed
        CHECK (status IN ('pending', 'running', 'succeeded', 'retry', 'dead')),
    CONSTRAINT background_jobs_priority_bounded
        CHECK (priority BETWEEN -1000 AND 1000),
    CONSTRAINT background_jobs_attempts_valid
        CHECK (
            attempts >= 0
            AND max_attempts BETWEEN 1 AND 100
            AND attempts <= max_attempts
        ),
    CONSTRAINT background_jobs_lock_state
        CHECK (
            (
                status = 'running'
                AND locked_at IS NOT NULL
                AND lock_owner IS NOT NULL
                AND char_length(lock_owner) BETWEEN 1 AND 128
                AND btrim(lock_owner) = lock_owner
                AND lease_expires_at IS NOT NULL
                AND lease_expires_at > locked_at
            )
            OR (
                status <> 'running'
                AND locked_at IS NULL
                AND lock_owner IS NULL
                AND lease_expires_at IS NULL
            )
        ),
    CONSTRAINT background_jobs_error_code_format
        CHECK (
            last_error_code IS NULL
            OR (
                char_length(last_error_code) BETWEEN 3 AND 80
                AND last_error_code ~ '^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$'
            )
        ),
    CONSTRAINT background_jobs_error_message_bounded
        CHECK (
            last_error_message IS NULL
            OR (
                char_length(last_error_message) BETWEEN 1 AND 500
                AND btrim(last_error_message) = last_error_message
            )
        ),
    CONSTRAINT background_jobs_completion_state
        CHECK (
            (status = 'succeeded' AND completed_at IS NOT NULL)
            OR (status <> 'succeeded' AND completed_at IS NULL)
        ),
    CONSTRAINT background_jobs_dead_state
        CHECK (
            (status = 'dead' AND dead_at IS NOT NULL)
            OR (status <> 'dead' AND dead_at IS NULL)
        ),
    CONSTRAINT background_jobs_terminal_after_created
        CHECK (
            (completed_at IS NULL OR completed_at >= created_at)
            AND (dead_at IS NULL OR dead_at >= created_at)
        ),
    CONSTRAINT background_jobs_updated_after_created
        CHECK (updated_at >= created_at)
);

COMMENT ON COLUMN tauco_app.background_jobs.payload_json IS
    'Allowlisted entity identifiers only; PII is forbidden by application validation';
COMMENT ON COLUMN tauco_app.background_jobs.last_error_message IS
    'Sanitized bounded diagnostic; secrets and PII are forbidden';

CREATE TABLE tauco_app.activity_logs (
    id uuid PRIMARY KEY,
    event_type text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid,
    actor_type text NOT NULL,
    actor_id uuid,
    metadata_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    request_id text,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    CONSTRAINT activity_logs_id_uuid_v7
        CHECK (id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
    CONSTRAINT activity_logs_event_type_format
        CHECK (
            char_length(event_type) BETWEEN 3 AND 100
            AND event_type ~ '^[a-z][a-z0-9]*([._-][a-z0-9]+)*$'
        ),
    CONSTRAINT activity_logs_entity_type_format
        CHECK (
            char_length(entity_type) BETWEEN 3 AND 80
            AND entity_type ~ '^[a-z][a-z0-9]*([._-][a-z0-9]+)*$'
        ),
    CONSTRAINT activity_logs_entity_id_uuid_v7
        CHECK (
            entity_id IS NULL
            OR entity_id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        ),
    CONSTRAINT activity_logs_actor_type_allowed
        CHECK (actor_type IN ('system', 'visitor', 'admin')),
    CONSTRAINT activity_logs_actor_id_uuid_v7
        CHECK (
            actor_id IS NULL
            OR actor_id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        ),
    CONSTRAINT activity_logs_metadata_object
        CHECK (jsonb_typeof(metadata_json) = 'object'),
    CONSTRAINT activity_logs_request_id_format
        CHECK (
            request_id IS NULL
            OR (
                char_length(request_id) BETWEEN 1 AND 128
                AND request_id ~ '^[A-Za-z0-9._:-]+$'
            )
        )
);

COMMENT ON COLUMN tauco_app.activity_logs.metadata_json IS
    'Allowlisted non-PII metadata only';

CREATE INDEX page_revisions_page_status_revision_idx
    ON tauco_app.page_revisions (page_id, status, revision_number DESC);
CREATE INDEX product_revisions_product_status_revision_idx
    ON tauco_app.product_revisions (product_id, status, revision_number DESC);
CREATE INDEX products_published_catalog_idx
    ON tauco_app.products (sort_order, id)
    WHERE published_revision_id IS NOT NULL;
CREATE INDEX media_assets_status_created_idx
    ON tauco_app.media_assets (status, created_at, id);
CREATE INDEX media_variants_asset_idx
    ON tauco_app.media_variants (media_asset_id, width);
CREATE INDEX contact_messages_inbox_idx
    ON tauco_app.contact_messages (status, created_at DESC, id);
CREATE INDEX contact_messages_retention_idx
    ON tauco_app.contact_messages (retention_delete_at, id);
CREATE INDEX background_jobs_claim_idx
    ON tauco_app.background_jobs (priority DESC, run_at, id)
    WHERE status IN ('pending', 'retry');
CREATE INDEX background_jobs_expired_lease_idx
    ON tauco_app.background_jobs (lease_expires_at, id)
    WHERE status = 'running';
CREATE INDEX activity_logs_entity_idx
    ON tauco_app.activity_logs (entity_type, entity_id, created_at DESC);
CREATE INDEX activity_logs_request_idx
    ON tauco_app.activity_logs (request_id)
    WHERE request_id IS NOT NULL;
CREATE INDEX activity_logs_created_brin_idx
    ON tauco_app.activity_logs USING brin (created_at);

REVOKE ALL ON
    tauco_app.pages,
    tauco_app.page_revisions,
    tauco_app.products,
    tauco_app.product_revisions,
    tauco_app.media_assets,
    tauco_app.media_variants,
    tauco_app.contact_messages,
    tauco_app.background_jobs,
    tauco_app.activity_logs
FROM PUBLIC;

REVOKE ALL ON
    tauco_app.pages,
    tauco_app.page_revisions,
    tauco_app.products,
    tauco_app.product_revisions,
    tauco_app.media_assets,
    tauco_app.media_variants,
    tauco_app.contact_messages,
    tauco_app.background_jobs,
    tauco_app.activity_logs
FROM tauco_runtime;

GRANT SELECT ON
    tauco_app.pages,
    tauco_app.page_revisions,
    tauco_app.products,
    tauco_app.product_revisions,
    tauco_app.media_assets,
    tauco_app.media_variants
TO tauco_runtime;

GRANT SELECT, INSERT ON tauco_app.contact_messages TO tauco_runtime;
GRANT SELECT, INSERT, UPDATE ON tauco_app.background_jobs TO tauco_runtime;
GRANT SELECT, INSERT ON tauco_app.activity_logs TO tauco_runtime;

RESET ROLE;
