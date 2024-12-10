BEGIN;
UPDATE condcache SET expired_at = $2, updated_at = $3 WHERE url = $1;
COMMIT;
