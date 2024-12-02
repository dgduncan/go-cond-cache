CREATE TABLE IF NOT EXISTS condcache (
    url text,
    item bytea,
    created_at timestamp default (now () at time zone 'utc'),
    updated_at timestamp,
    expired_at timestamp,
    PRIMARY KEY (url)
);
