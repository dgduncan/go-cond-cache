SELECT
    url
from
    condcache
where
    expired_at <= (now () at time zone 'utc');
