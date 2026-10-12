package observability

import "sync/atomic"

// readinessCheck is a readiness probe refreshed in the background with a
// fixed interval.
type readinessCheck struct {
	name   string
	status *atomic.Bool
}