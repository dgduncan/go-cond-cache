package caches

import "time"

var (
	// DefaultExpiredDuration the default expired duration
	DefaultExpiredDuration = 24 * time.Hour

	// DefaultExpiredTaskTimer is the default duration of the expired task timer
	DefaultExpiredTaskTimer = 10 * time.Minute
)
