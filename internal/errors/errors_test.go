package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	lmerrors "last-mile-task/internal/errors"
)

func TestNew_AllAccessors(t *testing.T) {
	inner := stderrors.New("inner failure")
	got := lmerrors.New("boom", "shipping.invalid_status_transition", 9, 412).
		WithDetails("details").
		WithAttributes(lmerrors.Attributes{"pkg": "a", "point": 7}).
		WithRequestID("req-1").
		WithCorrelationID("corr-1").
		Wrap(inner)

	require.Equal(t, "boom", got.Message())
	require.Equal(t, "boom: inner failure", got.Error())
	require.Equal(t, "shipping.invalid_status_transition", got.Code())
	require.Equal(t, "details", got.Details())
	require.Equal(t, int32(9), got.ProtoCode())
	require.Equal(t, 412, got.HTTPCode())
	require.Equal(t, "req-1", got.RequestID())
	require.Equal(t, "corr-1", got.CorrelationID())
	require.Equal(t, lmerrors.Attributes{"pkg": "a", "point": 7}, got.Attributes())
	require.Equal(t, inner, got.Unwrap())
}

func TestReason(t *testing.T) {
	got := lmerrors.New("boom", "validation.invalid_argument", 3, 400)
	require.Equal(t, "lmt.validation.invalid_argument", got.Reason())
}

func TestNamespaceFor(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"valid.unknown", "lmt.validation"},
		{"not_found.shipment", "lmt.resource"},
		{"internal", "lmt"},
		{"shipping.invalid_status_transition", "lmt"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, lmerrors.NamespaceFor(tt.code), "code %q", tt.code)
	}
}

func TestGRPCStatus(t *testing.T) {
	got := lmerrors.New("boom", "shipping.invalid_status_transition", int32(codes.FailedPrecondition), 412)

	st := got.GRPCStatus()
	require.Equal(t, codes.FailedPrecondition, st.Code())
	require.Equal(t, "boom", st.Message())
}

func TestFromProto(t *testing.T) {
	st := status.New(codes.NotFound, "shipment gone")

	got := lmerrors.FromProto(st.Err())
	require.Equal(t, int32(codes.NotFound), got.ProtoCode())
	require.Equal(t, "shipment gone", got.Message())

	plain := lmerrors.FromProto(stderrors.New("raw"))
	require.Equal(t, int32(codes.Internal), plain.ProtoCode())
}

func TestMapToGRPC(t *testing.T) {
	require.Equal(t, codes.OK, lmerrors.MapToGRPC(nil).Code())

	typed := lmerrors.New("boom", "shipping.invalid_status_transition", int32(codes.FailedPrecondition), 412)
	require.Equal(t, codes.FailedPrecondition, lmerrors.MapToGRPC(typed).Code())

	for _, raw := range []error{stderrors.New("raw"), stderrors.New("wrapped: boom")} {
		require.Equal(t, codes.Internal, lmerrors.MapToGRPC(raw).Code())
	}
}

func TestIs(t *testing.T) {
	typed := lmerrors.New("boom", "shipping.invalid_status_transition", 9, 412)
	require.True(t, lmerrors.Is(typed, "shipping.invalid_status_transition"))
	require.False(t, lmerrors.Is(typed, "internal"))

	plain := stderrors.New("raw")
	require.False(t, lmerrors.Is(plain, "internal"))
}

func TestEnrichRequest(t *testing.T) {
	typed := lmerrors.New("boom", "shipping.invalid_status_transition", 9, 412)

	got := lmerrors.EnrichRequest(typed, "req-1", "corr-1")
	var asErr *lmerrors.Error
	require.True(t, stderrors.As(got, &asErr))
	require.Equal(t, "req-1", asErr.RequestID())
	require.Equal(t, "corr-1", asErr.CorrelationID())

	// Present values must win.
	typed2 := lmerrors.New("boom2", "internal", 13, 500).WithRequestID("req-2")
	_ = lmerrors.EnrichRequest(typed2, "req-1", "corr-1")
	require.Equal(t, "req-2", typed2.RequestID())
	require.Equal(t, "corr-1", typed2.CorrelationID())
}

func TestWrapAndMessage(t *testing.T) {
	require.Nil(t, lmerrors.Wrap(nil))
	require.Empty(t, lmerrors.Message(nil))

	inner := stderrors.New("inner")
	wrapped := lmerrors.Wrap(inner)
	asErr, ok := wrapped.(*lmerrors.Error)
	require.True(t, ok)
	require.Equal(t, "internal", asErr.Code())
	require.Equal(t, int32(13), asErr.ProtoCode())
	require.Equal(t, inner, asErr.Unwrap())
	require.Equal(t, "inner: inner", lmerrors.Message(wrapped))
	require.Equal(t, "raw", lmerrors.Message(stderrors.New("raw")))
}

func TestGeneratedConstructors(t *testing.T) {
	bad := lmerrors.NewBadRequestError("invalid")
	require.Equal(t, int32(codes.InvalidArgument), bad.ProtoCode())
	require.Equal(t, 400, bad.HTTPCode())
	require.Equal(t, "validation.invalid_argument", bad.Code())

	notFound := lmerrors.NewNotFoundError("missing")
	require.Equal(t, int32(codes.NotFound), notFound.ProtoCode())
	require.Equal(t, 404, notFound.HTTPCode())

	internal := lmerrors.NewInternalError("internal")
	require.Equal(t, int32(codes.Internal), internal.ProtoCode())
	require.Equal(t, 500, internal.HTTPCode())

	precondition := lmerrors.NewFailedPreconditionError("state")
	require.Equal(t, int32(codes.FailedPrecondition), precondition.ProtoCode())
	require.Equal(t, 412, precondition.HTTPCode())
}

func TestDocumentsConstructors(t *testing.T) {
	notFound := lmerrors.NewDocumentsNotFoundError("missing")
	require.Equal(t, int32(codes.NotFound), notFound.ProtoCode())
	require.Equal(t, 404, notFound.HTTPCode())
	require.Equal(t, "documents_document_not_found", notFound.Code())

	invalid := lmerrors.NewDocumentsInvalidError("invalid")
	require.Equal(t, int32(codes.FailedPrecondition), invalid.ProtoCode())
	require.Equal(t, 412, invalid.HTTPCode())
	require.Equal(t, "documents_invalid_document", invalid.Code())

	delivery := lmerrors.NewDocumentsDeliveryError("delivery")
	require.Equal(t, int32(codes.Internal), delivery.ProtoCode())
	require.Equal(t, 500, delivery.HTTPCode())
	require.Equal(t, "documents_delivery_failed", delivery.Code())
}

func TestTranslate(t *testing.T) {
	typed := lmerrors.New("boom", "internal", 13, 500)
	require.Equal(t, "boom", lmerrors.Translate(typed, lmerrors.Locale("ru")))
}