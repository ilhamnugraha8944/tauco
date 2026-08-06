SET ROLE tauco_migrator;
SET search_path TO tauco_app, pg_catalog;

CREATE FUNCTION tauco_app.tauco_purge_expired_contact_messages(
    purge_before timestamptz,
    purge_limit integer
)
RETURNS bigint
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, tauco_app
AS $$
DECLARE
    deleted_count bigint;
BEGIN
    IF purge_before IS NULL OR purge_before > statement_timestamp() THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'purge cutoff must not be null or in the future';
    END IF;

    IF purge_limit IS NULL OR purge_limit < 1 OR purge_limit > 1000 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'purge limit must be between 1 and 1000';
    END IF;

    WITH candidates AS (
        SELECT id
        FROM tauco_app.contact_messages
        WHERE retention_delete_at <= purge_before
        ORDER BY retention_delete_at, id
        LIMIT purge_limit
        FOR UPDATE SKIP LOCKED
    ), deleted AS (
        DELETE FROM tauco_app.contact_messages AS message
        USING candidates
        WHERE message.id = candidates.id
        RETURNING 1
    )
    SELECT count(*) INTO deleted_count FROM deleted;

    RETURN deleted_count;
END
$$;

REVOKE ALL ON FUNCTION tauco_app.tauco_purge_expired_contact_messages(timestamptz, integer)
FROM PUBLIC;

GRANT EXECUTE ON FUNCTION tauco_app.tauco_purge_expired_contact_messages(timestamptz, integer)
TO tauco_runtime;

RESET ROLE;
