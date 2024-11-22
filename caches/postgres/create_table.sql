CREATE TABLE IF NOT EXISTS condcache (
    url text,
    item bytea,
    created_at timestamp default current_timestamp,
    updated_at timestamp,
    expired_at timestamp,
    PRIMARY KEY(url)
);
