SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

CREATE FUNCTION tauco_app.tauco_set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, tauco_app
AS $$
BEGIN
    NEW.updated_at := statement_timestamp();
    RETURN NEW;
END
$$;

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

CREATE FUNCTION tauco_app.tauco_assert_page_published_revision()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, tauco_app
AS $$
BEGIN
    IF NEW.published_revision_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM tauco_app.page_revisions AS revision
        WHERE revision.id = NEW.published_revision_id
          AND revision.page_id = NEW.id
          AND revision.status = 'published'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'page published revision must be published and belong to the same page';
    END IF;

    RETURN NEW;
END
$$;

CREATE FUNCTION tauco_app.tauco_assert_product_published_revision()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, tauco_app
AS $$
BEGIN
    IF NEW.published_revision_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM tauco_app.product_revisions AS revision
        WHERE revision.id = NEW.published_revision_id
          AND revision.product_id = NEW.id
          AND revision.status = 'published'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'product published revision must be published and belong to the same product';
    END IF;

    RETURN NEW;
END
$$;

CREATE FUNCTION tauco_app.tauco_reject_activity_log_mutation()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, tauco_app
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'activity logs are append-only';
END
$$;

CREATE TRIGGER pages_set_updated_at
    BEFORE UPDATE ON tauco_app.pages
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE TRIGGER products_set_updated_at
    BEFORE UPDATE ON tauco_app.products
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE TRIGGER media_assets_set_updated_at
    BEFORE UPDATE ON tauco_app.media_assets
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE TRIGGER contact_messages_set_updated_at
    BEFORE UPDATE ON tauco_app.contact_messages
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE TRIGGER background_jobs_set_updated_at
    BEFORE UPDATE ON tauco_app.background_jobs
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_set_updated_at();

CREATE TRIGGER page_revisions_reject_published_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.page_revisions
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_published_revision_mutation();

CREATE TRIGGER product_revisions_reject_published_mutation
    BEFORE UPDATE OR DELETE ON tauco_app.product_revisions
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_reject_published_revision_mutation();

CREATE CONSTRAINT TRIGGER pages_z_assert_published_revision
    AFTER INSERT OR UPDATE ON tauco_app.pages
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_assert_page_published_revision();

CREATE CONSTRAINT TRIGGER products_z_assert_published_revision
    AFTER INSERT OR UPDATE ON tauco_app.products
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
    EXECUTE FUNCTION tauco_app.tauco_assert_product_published_revision();

CREATE TRIGGER activity_logs_reject_mutation
    BEFORE UPDATE OR DELETE OR TRUNCATE ON tauco_app.activity_logs
    FOR EACH STATEMENT
    EXECUTE FUNCTION tauco_app.tauco_reject_activity_log_mutation();

REVOKE ALL ON FUNCTION tauco_app.tauco_set_updated_at() FROM PUBLIC;
REVOKE ALL ON FUNCTION tauco_app.tauco_reject_published_revision_mutation() FROM PUBLIC;
REVOKE ALL ON FUNCTION tauco_app.tauco_assert_page_published_revision() FROM PUBLIC;
REVOKE ALL ON FUNCTION tauco_app.tauco_assert_product_published_revision() FROM PUBLIC;
REVOKE ALL ON FUNCTION tauco_app.tauco_reject_activity_log_mutation() FROM PUBLIC;

GRANT EXECUTE ON FUNCTION tauco_app.tauco_set_updated_at() TO tauco_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_reject_published_revision_mutation() TO tauco_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_assert_page_published_revision() TO tauco_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_assert_product_published_revision() TO tauco_runtime;
GRANT EXECUTE ON FUNCTION tauco_app.tauco_reject_activity_log_mutation() TO tauco_runtime;

RESET ROLE;
