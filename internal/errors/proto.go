package errors

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NamespaceFor resolves the error namespace by code prefix (errors.yaml
// grouping). Unknown codes map to the "lmt" root namespace.
func NamespaceFor(code string) string {
	switch {
	case strings.HasPrefix(code, "valid."):
		return "lmt.validation"
	case strings.HasPrefix(code, "not_found."):
		return "lmt.resource"
	default:
		return "lmt"
	}
}

// GRPCStatus implements the gRPC status interface, mapping the error into a
// gRPC status with the code carried by the internal error.
func (e *Error) GRPCStatus() *status.Status {
	return status.New(codes.Code(e.protocode), e.message)
}

// FromProto converts a gRPC status error back into an internal error.
func FromProto(err error) *Error {
	st, ok := status.FromError(err)
	if !ok {
		return New(err.Error(), "internal", int32(codes.Internal), 500)
	}
	return New(st.Message(), "internal", int32(st.Code()), 500)
}

// MapToGRPC converts any error into a gRPC status, falling back to Internal.
func MapToGRPC(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}
	var ie *Error
	if errors.As(err, &ie) {
		return status.New(codes.Code(ie.protocode), ie.message)
	}
	return status.New(codes.Internal, err.Error())
}