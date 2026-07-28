SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

DROP TRIGGER IF EXISTS activity_logs_reject_mutation ON tauco_app.activity_logs;
DROP TRIGGER IF EXISTS products_z_assert_published_revision ON tauco_app.products;
DROP TRIGGER IF EXISTS pages_z_assert_published_revision ON tauco_app.pages;
DROP TRIGGER IF EXISTS product_revisions_reject_published_mutation ON tauco_app.product_revisions;
DROP TRIGGER IF EXISTS page_revisions_reject_published_mutation ON tauco_app.page_revisions;
DROP TRIGGER IF EXISTS background_jobs_set_updated_at ON tauco_app.background_jobs;
DROP TRIGGER IF EXISTS contact_messages_set_updated_at ON tauco_app.contact_messages;
DROP TRIGGER IF EXISTS media_assets_set_updated_at ON tauco_app.media_assets;
DROP TRIGGER IF EXISTS products_set_updated_at ON tauco_app.products;
DROP TRIGGER IF EXISTS pages_set_updated_at ON tauco_app.pages;

DROP FUNCTION IF EXISTS tauco_app.tauco_reject_activity_log_mutation();
DROP FUNCTION IF EXISTS tauco_app.tauco_assert_product_published_revision();
DROP FUNCTION IF EXISTS tauco_app.tauco_assert_page_published_revision();
DROP FUNCTION IF EXISTS tauco_app.tauco_reject_published_revision_mutation();
DROP FUNCTION IF EXISTS tauco_app.tauco_set_updated_at();

RESET ROLE;
