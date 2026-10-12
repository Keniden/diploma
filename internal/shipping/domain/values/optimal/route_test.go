package optimal_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values/optimal"
)

func TestRoute_MiddlePoints(t *testing.T) {
	r := optimal.Route{
		Distance: 100,
		Duration: 200,
		Points: []optimal.Point{
			{ID: 1, SeqNum: 1, Distance: 0, Duration: 0},
			{ID: 2, SeqNum: 2, Distance: 10, Duration: 5},
			{ID: 3, SeqNum: 3, Distance: 15, Duration: 8},
			{ID: 4, SeqNum: 4, Distance: 20, Duration: 10},
		},
	}

	middle, err := r.MiddlePoints()
	if err != nil {
		t.Fatal(err)
	}

	if len(middle) != 2 {
		t.Fatalf("expected 2 middle points, got %d", len(middle))
	}
	if middle[0].ID != 2 || middle[1].ID != 3 {
		t.Fatal("middle points have wrong IDs")
	}
}

func TestRoute_MiddlePoints_TooShort(t *testing.T) {
	r := optimal.Route{
		Points: []optimal.Point{
			{ID: 1, SeqNum: 1},
		},
	}

	_, err := r.MiddlePoints()
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_MiddlePoints_MinimalValid(t *testing.T) {
	r := optimal.Route{
		Points: []optimal.Point{
			{ID: 1, SeqNum: 1, Distance: 0, Duration: 0},
			{ID: 2, SeqNum: 2, Distance: 10, Duration: 5},
		},
	}

	middle, err := r.MiddlePoints()
	if err != nil {
		t.Fatal(err)
	}

	if len(middle) != 0 {
		t.Fatalf("expected 0 middle points for 2-point route, got %d", len(middle))
	}
}
