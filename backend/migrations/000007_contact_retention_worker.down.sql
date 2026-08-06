SET ROLE tauco_migrator;

REVOKE EXECUTE ON FUNCTION tauco_app.tauco_purge_expired_contact_messages(timestamptz, integer)
FROM tauco_runtime;

DROP FUNCTION tauco_app.tauco_purge_expired_contact_messages(timestamptz, integer);

RESET ROLE;
