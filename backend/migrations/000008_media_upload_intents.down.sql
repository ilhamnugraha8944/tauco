SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

REVOKE ALL ON tauco_app.media_upload_intents
FROM tauco_runtime, tauco_admin_runtime;

DROP TRIGGER IF EXISTS media_upload_intents_set_updated_at
ON tauco_app.media_upload_intents;
DROP TABLE IF EXISTS tauco_app.media_upload_intents;

RESET ROLE;
