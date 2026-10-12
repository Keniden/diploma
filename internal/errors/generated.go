package errors

import "google.golang.org/grpc/codes"

// Generated-style constructors (errgen API shape). The skeleton keeps the
// error pool in errors.yaml; constructors below cover the slice flows.
//
// TODO(LASTMILE-014): production code is generated with errgen (cmd/errorgen)
// — do not hand-edit the generated package.

// NewBadRequestError builds a validation error (InvalidArgument/400).
func NewBadRequestError(message string) *Error {
	return New(message, "validation.invalid_argument", int32(codes.InvalidArgument), 400)
}

// NewNotFoundError builds an entity-not-found error (NotFound/404).
func NewNotFoundError(message string) *Error {
	return New(message, "resource.not_found", int32(codes.NotFound), 404)
}

// NewInternalError builds an internal error (Internal/500).
func NewInternalError(message string) *Error {
	return New(message, "internal", int32(codes.Internal), 500)
}

// NewFailedPreconditionError builds a state-transition error
// (FailedPrecondition/412).
func NewFailedPreconditionError(message string) *Error {
	return New(message, "shipping.invalid_status_transition", int32(codes.FailedPrecondition), 412)
}

// NewDocumentsNotFoundError builds a documents not-found error
// (NotFound/404). TODO(LASTMILE-014): errgen.
func NewDocumentsNotFoundError(message string) *Error {
	return New(message, "documents_document_not_found", int32(codes.NotFound), 404)
}

// NewDocumentsInvalidError builds a documents invalid-data error
// (FailedPrecondition/412). TODO(LASTMILE-014): errgen.
func NewDocumentsInvalidError(message string) *Error {
	return New(message, "documents_invalid_document", int32(codes.FailedPrecondition), 412)
}

// NewDocumentsDeliveryError builds a documents delivery error
// (Internal/500). TODO(LASTMILE-014): errgen.
func NewDocumentsDeliveryError(message string) *Error {
	return New(message, "documents_delivery_failed", int32(codes.Internal), 500)
}