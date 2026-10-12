// Package requestid carries the request correlation id through the service:
// it is extracted from incoming gRPC metadata / HTTP headers, injected into
// the context and rendered by the logging layer.
package requestid

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

const (
	// Header is the wire-level header that carries the request id.
	Header = "X-Request-Id"
	// TraceAttributeName mirrors the attribute name used by the real repo's
	// tracing integration. TODO(LASTMILE-092): drop once tracing is ported.
	TraceAttributeName = "http.request.id"
)

type contextKey struct{}

// Inject stores the request id in the context.
func Inject(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// Extract returns the request id stored in the context (empty string when
// absent).
func Extract(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// FromRequest returns the request id from an HTTP request header.
func FromRequest(r *http.Request) string {
	return r.Header.Get(Header)
}

// InjectFromRequest stores the HTTP request id into the context.
func InjectFromRequest(ctx context.Context, r *http.Request) context.Context {
	return Inject(ctx, FromRequest(r))
}

// FromIncomingContext extracts the request id from incoming gRPC metadata
// and stores it in the context. The original context is returned when no
// metadata or no header is present.
func FromIncomingContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	if val := md.Get(Header); len(val) > 0 {
		ctx = Inject(ctx, val[0])
	}

	return ctx
}