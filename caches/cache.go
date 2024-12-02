package caches

import "time"

var (
	DefaultExpiredDuration = 24 * time.Hour

	DefaultExpiredTaskTimer = 10 * time.Minute
)
