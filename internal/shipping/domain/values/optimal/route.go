package optimal

import (
	"errors"

	"last-mile-task/internal/shipping/domain/values"
)

// Route optimized
type Route struct {
	Distance int64 // Total optimal distance
	Duration int64 // Total optimal duration

	// Optimized points in sequence order
	//
	// Contains all points including source and target
	Points []Point
}

func (r *Route) MiddlePoints() ([]Point, error) {
	if len(r.Points) < 2 {
		return nil, errors.New("route is too short")
	}

	return r.Points[1 : len(r.Points)-1], nil
}

// Point optimized
type Point struct {
	ID       values.PointID // ID reference to actual point
	SeqNum   int32          // SeqNum Sequence Number
	Distance int64          // Distance from previous point to current
	Duration int64          // Duration from previous point to current
}
