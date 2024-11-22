DELETE FROM condcache WHERE expired_at < NOW();
