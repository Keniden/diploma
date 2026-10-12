package logging

import (
	"context"
	"log/slog"

	"last-mile-task/internal/platform/requestid"
)

// AddRequestID appends the request id attribute to the record when the
// context carries one.
func AddRequestID(ctx context.Context, r *slog.Record) {
	rid := requestid.Extract(ctx)

	if rid != "" {
		r.Add("request_id", rid)
	}
}