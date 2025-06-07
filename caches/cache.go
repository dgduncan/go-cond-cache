package caches

import (
	"net/http"
	"strings"
	"time"
)

var (
	// DefaultExpiredDuration the default expired duration.
	DefaultExpiredDuration = 24 * time.Hour

	// DefaultExpiredTaskTimer is the default duration of the expired task timer.
	DefaultExpiredTaskTimer = 10 * time.Minute
)

func Key(req http.Request) string {
	return strings.Join([]string{req.Method, "#", req.URL.String()}, "")
}
