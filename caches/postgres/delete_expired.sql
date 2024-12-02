DELETE FROM condcache
WHERE
    expired_at < (now () at time zone 'utc');
