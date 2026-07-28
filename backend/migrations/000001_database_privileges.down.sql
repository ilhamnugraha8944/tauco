-- Security hardening is intentionally not relaxed on down. PUBLIC never gains
-- privileges on the application schema, and the out-of-band schema remains so
-- golang-migrate can retain its metadata table.
REVOKE ALL ON SCHEMA tauco_app FROM tauco_runtime;
REVOKE ALL ON SCHEMA tauco_app FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA tauco_app TO tauco_migrator;
