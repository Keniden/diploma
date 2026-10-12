package primitives

import "time"

// Now returns the current UTC time truncated to microseconds, the precision
// used by the persistence layer.
func Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}