package utils

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTimestampOrNilNil(t *testing.T) {
	if got := TimestampOrNil(nil); got != nil {
		t.Fatalf("TimestampOrNil(nil) = %v, want nil", got)
	}
}

func TestTimestampOrNilZero(t *testing.T) {
	zero := time.Time{}
	if got := TimestampOrNil(&zero); got != nil {
		t.Fatalf("TimestampOrNil(zero) = %v, want nil", got)
	}
}

func TestTimestampOrNil(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	got := TimestampOrNil(&now)
	want := timestamppb.New(now)
	if !got.AsTime().Equal(want.AsTime()) {
		t.Fatalf("TimestampOrNil = %v, want %v", got, want)
	}
}
