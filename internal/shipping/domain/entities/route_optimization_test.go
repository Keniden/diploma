package entities_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/optimal"
)

func TestRoute_ApplyOptimization(t *testing.T) {
	route := &entities.Route{
		ID: values.NewID(),
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 4, SeqNum: entities.SourceSeqNum + 2},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 3},
		},
	}

	opt := optimal.Route{
		Distance: 10,
		Duration: 20,
		Points: []optimal.Point{
			{ID: 1, SeqNum: entities.SourceSeqNum, Distance: 0, Duration: 0},     // source
			{ID: 2, SeqNum: entities.SourceSeqNum + 1, Distance: 1, Duration: 2}, // middle
			{ID: 4, SeqNum: entities.SourceSeqNum + 2, Distance: 2, Duration: 4}, // middle
			{ID: 3, SeqNum: entities.SourceSeqNum + 3, Distance: 1, Duration: 2}, // target
		},
	}

	if err := route.ApplyOptimization(opt); err != nil {
		t.Fatal(err)
	}
	if route.Distance != 10 || route.Duration != 20 {
		t.Fatalf("expected distance/duration applied")
	}
}
