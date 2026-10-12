package middleware

import (
	"net/http"

	"last-mile-task/internal/platform/requestid"
)

// HTTPRequestID injects the incoming X-Request-Id header into the request
// context so the logging handler can attach it.
func HTTPRequestID() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r.WithContext(requestid.InjectFromRequest(r.Context(), r)))
		})
	}
}