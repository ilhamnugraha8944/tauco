SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

GRANT INSERT, UPDATE ON tauco_app.media_assets TO tauco_runtime;
GRANT INSERT, UPDATE ON tauco_app.media_variants TO tauco_runtime;

RESET ROLE;

