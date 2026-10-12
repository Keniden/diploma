// Package middleware provides HTTP middleware for the observability server.
//
// Only the request-id plumbing is ported; tracing/metrics middleware is not
// available in the skeleton. TODO(LASTMILE-092): port HTTPTracing and
// HTTPMetrics once tracing/metrics are allowed in go.mod.
package middleware

import "net/http"

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares in order: the first middleware is the
// outermost.
func Chain(mws ...Middleware) Middleware {
	return func(handler http.Handler) http.Handler {
		for _, mw := range mws {
			handler = mw(handler)
		}
		return handler
	}
}