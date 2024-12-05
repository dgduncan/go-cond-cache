BEGIN;
UPDATE condcache SET expired_at = $2 WHERE url = $1;
COMMIT;