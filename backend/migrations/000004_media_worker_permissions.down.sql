SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

REVOKE INSERT, UPDATE ON tauco_app.media_assets FROM tauco_runtime;
REVOKE INSERT, UPDATE ON tauco_app.media_variants FROM tauco_runtime;

RESET ROLE;

