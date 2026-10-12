// Package errors defines the internal error machinery of the service.
//
// TODO(LASTMILE-014): production generates this code with errgen
// (cmd/errorgen) from errors.yaml — the generated code must not be edited.
// Skeleton version is hand-written to the same shape.
package errors

import (
	"context"
	"errors"
	"fmt"
)

// Attributes holds structured error details.
type Attributes map[string]any

// Error is the internal error carrying a stable code, transport codes and
// enrichment attributes.
type Error struct {
	message       string
	code          string
	details       string
	requestID     string
	correlationID string
	attrs         Attributes
	protocode     int32
	httpcode      int
	wrapped       error
}

// New builds an error with a stable code and gRPC/HTTP transport codes.
func New(message, code string, protocode int32, httpcode int) *Error {
	return &Error{
		message:   message,
		code:      code,
		protocode: protocode,
		httpcode:  httpcode,
	}
}

// Wrap adds the underlying error to an internal error.
func (e *Error) Wrap(err error) *Error {
	e.wrapped = err
	return e
}

// WithDetails enriches the error details.
func (e *Error) WithDetails(details string) *Error {
	e.details = details
	return e
}

// WithAttributes attaches structured attributes.
func (e *Error) WithAttributes(attrs Attributes) *Error {
	e.attrs = attrs
	return e
}

// WithRequestID attaches the request id for correlation.
func (e *Error) WithRequestID(requestID string) *Error {
	e.requestID = requestID
	return e
}

// WithCorrelationID attaches the correlation id.
func (e *Error) WithCorrelationID(correlationID string) *Error {
	e.correlationID = correlationID
	return e
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %v", e.message, e.wrapped)
	}
	return e.message
}

// Message returns the root message without the wrapped chain.
func (e *Error) Message() string { return e.message }

// Unwrap exposes the wrapped error.
func (e *Error) Unwrap() error { return e.wrapped }

// Code returns the stable business code.
func (e *Error) Code() string { return e.code }

// Details returns the optional details.
func (e *Error) Details() string { return e.details }

// RequestID returns the attached request id.
func (e *Error) RequestID() string { return e.requestID }

// CorrelationID returns the attached correlation id.
func (e *Error) CorrelationID() string { return e.correlationID }

// Attributes returns the structured attributes.
func (e *Error) Attributes() Attributes { return e.attrs }

// ProtoCode returns the gRPC status code.
func (e *Error) ProtoCode() int32 { return e.protocode }

// HTTPCode returns the HTTP status code.
func (e *Error) HTTPCode() int { return e.httpcode }

// Reason derives a human-readable reason from the code.
func (e *Error) Reason() string {
	return fmt.Sprintf("%s.%s", NamespaceFor(e.code), e.code)
}

// EnrichRequest attaches request metadata to the error, all present fields win.
func EnrichRequest(err error, requestID, correlationID string) error {
	var ie *Error
	if errors.As(err, &ie) {
		if ie.requestID == "" {
			ie.requestID = requestID
		}
		if ie.correlationID == "" {
			ie.correlationID = correlationID
		}
	}
	return err
}

// Is reports whether the error chain contains an error with the given code.
func Is(err error, code string) bool {
	var ie *Error
	if errors.As(err, &ie) {
		return ie.code == code
	}
	return false
}

// ContextKey is the key type for request-scoped values.
type ContextKey string

// WithRequestMetadata attaches request/correlation ids to the context.
func WithRequestMetadata(ctx context.Context, requestID, correlationID string) context.Context {
	return context.WithValue(ctx, ContextKey("request_id"), requestID)
}