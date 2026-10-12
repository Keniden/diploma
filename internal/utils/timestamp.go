package utils

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TimestampOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}
