package values_test

import (
	"testing"

	"last-mile-task/internal/documents/domain/values"
)

func TestDocumentTypeValues(t *testing.T) {
	tests := []struct {
		name string
		got  values.DocumentType
		want int16
	}{
		{name: "unspecified defaults to 0", got: values.DocumentTypeUnspecified, want: 0},
		{name: "waybill forward is 1", got: values.DocumentTypeWaybillForward, want: 1},
		{name: "waybill return is 2", got: values.DocumentTypeWaybillReturn, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int16(tt.got) != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, int16(tt.got))
			}
		})
	}
}

func TestDocumentTypeIsValid(t *testing.T) {
	if values.DocumentTypeWaybillForward == values.DocumentTypeWaybillReturn {
		t.Fatal("forward and return must be distinct")
	}
	if values.DocumentTypeUnspecified == values.DocumentTypeWaybillForward {
		t.Fatal("unspecified must not collide")
	}
}

func TestIDAliases(t *testing.T) {
	id := values.NewID()
	if id == values.NilID {
		t.Fatal("expected non-nil id")
	}
}