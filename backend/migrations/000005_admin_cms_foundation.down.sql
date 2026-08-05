SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

REVOKE ALL ON ALL TABLES IN SCHEMA tauco_app FROM tauco_admin_runtime;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA tauco_app FROM tauco_admin_runtime;

DROP TRIGGER IF EXISTS mfa_credentials_set_updated_at ON tauco_app.mfa_credentials;
DROP TRIGGER IF EXISTS admin_users_set_updated_at ON tauco_app.admin_users;
DROP TRIGGER IF EXISTS product_revision_media_reject_mutation ON tauco_app.product_revision_media;
DROP TRIGGER IF EXISTS page_revision_media_reject_mutation ON tauco_app.page_revision_media;
DROP TRIGGER IF EXISTS product_revisions_reject_mutation ON tauco_app.product_revisions;
DROP TRIGGER IF EXISTS page_revisions_reject_mutation ON tauco_app.page_revisions;
DROP FUNCTION IF EXISTS tauco_app.tauco_reject_revision_mutation();

CREATE FUNCTION tauco_app.tauco_reject_published_revision_mutation()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, tauco_app
AS $$
BEGIN
    IF OLD.status = 'published' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'published revisions are immutable';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER page_revisions_reject_published_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.page_revisions
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_published_revision_mutation();

CREATE TRIGGER product_revisions_reject_published_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.product_revisions
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_published_revision_mutation();

ALTER TABLE tauco_app.product_revisions
    DROP CONSTRAINT IF EXISTS product_revisions_created_by_admin_fk;
ALTER TABLE tauco_app.page_revisions
    DROP CONSTRAINT IF EXISTS page_revisions_created_by_admin_fk;
ALTER TABLE tauco_app.products
    DROP CONSTRAINT IF EXISTS products_archived_after_created,
    DROP COLUMN IF EXISTS archived_at;

DROP TABLE IF EXISTS tauco_app.product_revision_media;
DROP TABLE IF EXISTS tauco_app.page_revision_media;
DROP TABLE IF EXISTS tauco_app.mfa_recovery_codes;
DROP TABLE IF EXISTS tauco_app.mfa_credentials;
DROP TABLE IF EXISTS tauco_app.admin_refresh_tokens;
DROP TABLE IF EXISTS tauco_app.admin_sessions;
DROP TABLE IF EXISTS tauco_app.role_permissions;
DROP TABLE IF EXISTS tauco_app.user_roles;
DROP TABLE IF EXISTS tauco_app.permissions;
DROP TABLE IF EXISTS tauco_app.roles;
DROP TABLE IF EXISTS tauco_app.admin_users;

REVOKE ALL ON FUNCTION tauco_app.tauco_reject_published_revision_mutation() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_reject_published_revision_mutation() TO tauco_runtime;

RESET ROLE;
