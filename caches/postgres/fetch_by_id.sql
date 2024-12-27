SELECT
    url,
    item
FROM
    condcache
WHERE
    url = $1 AND expired_at >= $2;
