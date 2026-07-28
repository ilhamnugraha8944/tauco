SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

ALTER TABLE tauco_app.products
    DROP CONSTRAINT IF EXISTS products_published_revision_same_owner_fk;
ALTER TABLE tauco_app.pages
    DROP CONSTRAINT IF EXISTS pages_published_revision_same_owner_fk;

DROP TABLE IF EXISTS tauco_app.activity_logs;
DROP TABLE IF EXISTS tauco_app.background_jobs;
DROP TABLE IF EXISTS tauco_app.contact_messages;
DROP TABLE IF EXISTS tauco_app.media_variants;
DROP TABLE IF EXISTS tauco_app.media_assets;
DROP TABLE IF EXISTS tauco_app.product_revisions;
DROP TABLE IF EXISTS tauco_app.products;
DROP TABLE IF EXISTS tauco_app.page_revisions;
DROP TABLE IF EXISTS tauco_app.pages;

RESET ROLE;
