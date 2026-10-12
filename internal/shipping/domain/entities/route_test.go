package entities_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/optimal"
)

func TestNewRouteFromTemplate(t *testing.T) {
	template := values.RouteTemplate{
		RouteTemplateID: 100,
		Distance:        5000,
		Duration:        3600,
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: 1, SeqNum: 1, Distance: 0},
			{RoutePointTemplateID: 2, SeqNum: 2, Distance: 1000},
			{RoutePointTemplateID: 3, SeqNum: 3, Distance: 2000},
		},
	}

	route, err := entities.NewRouteFromTemplate(template)
	if err != nil {
		t.Fatal(err)
	}

	if route.ID == values.NilID {
		t.Fatal("expected non-nil route ID")
	}
	if route.TemplateID != template.RouteTemplateID {
		t.Fatalf("expected template ID %d, got %d", template.RouteTemplateID, route.TemplateID)
	}
	if route.Distance != template.Distance {
		t.Fatalf("expected distance %d, got %d", template.Distance, route.Distance)
	}
	if route.Duration != template.Duration {
		t.Fatalf("expected duration %d, got %d", template.Duration, route.Duration)
	}
	if len(route.Points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(route.Points))
	}
}

func TestNewRouteFromTemplate_TooShort(t *testing.T) {
	tests := []struct {
		name   string
		points []values.RoutePointTemplate
	}{
		{"empty", nil},
		{"one point", []values.RoutePointTemplate{{RoutePointTemplateID: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewRouteFromTemplate(values.RouteTemplate{
				Points: tt.points,
			})
			if err == nil {
				t.Fatal("expected error for too-short route")
			}
		})
	}
}

func TestNewRouteFromTemplate_TwoPoints(t *testing.T) {
	template := values.RouteTemplate{
		RouteTemplateID: 100,
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: 1, SeqNum: 1, Distance: 0},
			{RoutePointTemplateID: 2, SeqNum: 2, Distance: 1000},
		},
	}

	route, err := entities.NewRouteFromTemplate(template)
	if err != nil {
		t.Fatal(err)
	}
	if len(route.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(route.Points))
	}
}

func TestRoute_Source(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 10, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 20, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 30, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	source, err := r.Source()
	if err != nil {
		t.Fatal(err)
	}
	if source.PointID != 10 {
		t.Fatalf("expected source point ID 10, got %d", source.PointID)
	}
}

func TestRoute_Target(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 10, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 20, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 30, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	target, err := r.Target()
	if err != nil {
		t.Fatal(err)
	}
	if target.PointID != 30 {
		t.Fatalf("expected target point ID 30, got %d", target.PointID)
	}
}

func TestRoute_Source_TooShort(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 10},
		},
	}

	_, err := r.Source()
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_Target_TooShort(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 10},
		},
	}

	_, err := r.Target()
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_AssignToShipment(t *testing.T) {
	r := &entities.Route{}
	id := values.NewID()

	r.AssignToShipment(id)

	if r.ShipmentID != id {
		t.Fatalf("expected shipment ID %v, got %v", id, r.ShipmentID)
	}
}

func TestRoute_MiddlePoints(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 2},
			{ID: values.NewID(), PointID: 4, SeqNum: entities.SourceSeqNum + 3},
		},
	}

	middle, err := r.MiddlePoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(middle) != 2 {
		t.Fatalf("expected 2 middle points, got %d", len(middle))
	}
	if middle[0].PointID != 2 || middle[1].PointID != 3 {
		t.Fatal("middle points have wrong IDs")
	}
}

func TestRoute_MiddlePoints_TwoPoints(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
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

func TestRoute_MiddlePoints_TooShort(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1},
		},
	}

	_, err := r.MiddlePoints()
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_RoutePointsForLoading(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	points, err := r.RoutePointsForLoading()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points for loading (all except last), got %d", len(points))
	}
	if points[0].PointID != 1 || points[1].PointID != 2 {
		t.Fatal("loading points have wrong IDs")
	}
}

func TestRoute_RoutePointsForLoading_TwoPoints(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
		},
	}

	points, err := r.RoutePointsForLoading()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].PointID != 1 {
		t.Fatal("expected only source for loading on 2-point route")
	}
}

func TestRoute_RoutePointsForLoading_TooShort(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1},
		},
	}

	_, err := r.RoutePointsForLoading()
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_RoutePointsForUnloading(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	points, err := r.RoutePointsForUnloading()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points for unloading (all except first), got %d", len(points))
	}
	if points[0].PointID != 2 || points[1].PointID != 3 {
		t.Fatal("unloading points have wrong IDs")
	}
}

func TestRoute_RoutePointsForUnloading_TwoPoints(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
		},
	}

	points, err := r.RoutePointsForUnloading()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].PointID != 2 {
		t.Fatal("expected only target for unloading on 2-point route")
	}
}

func TestRoute_RoutePointsForUnloading_TooShort(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1},
		},
	}

	_, err := r.RoutePointsForUnloading()
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_SetDistance(t *testing.T) {
	r := &entities.Route{}
	r.SetDistance(999)
	if r.Distance != 999 {
		t.Fatalf("expected distance 999, got %d", r.Distance)
	}
}

func TestRoute_SetDuration(t *testing.T) {
	r := &entities.Route{}
	r.SetDuration(888)
	if r.Duration != 888 {
		t.Fatalf("expected duration 888, got %d", r.Duration)
	}
}

func TestRoute_ApplyOptimization_MissingPoint(t *testing.T) {
	r := &entities.Route{
		ID: values.NewID(),
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	opt := optimal.Route{
		Points: []optimal.Point{
			{ID: 1, SeqNum: entities.SourceSeqNum, Distance: 0, Duration: 0},
			{ID: 999, SeqNum: entities.SourceSeqNum + 1, Distance: 1, Duration: 2},
			{ID: 3, SeqNum: entities.SourceSeqNum + 2, Distance: 1, Duration: 2},
		},
	}

	err := r.ApplyOptimization(opt)
	if err == nil {
		t.Fatal("expected error for missing route point in optimization")
	}
}

func TestRoute_ApplyOptimization_TooShort(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1},
		},
	}

	opt := optimal.Route{
		Points: []optimal.Point{
			{ID: 1},
		},
	}

	err := r.ApplyOptimization(opt)
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestRoute_StripPointsByLoadedPackages(t *testing.T) {
	rp1 := values.NewID()
	rp2 := values.NewID()
	rp3 := values.NewID()
	rp4 := values.NewID()

	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: rp1, PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: rp2, PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: rp3, PointID: 3, SeqNum: entities.SourceSeqNum + 2},
			{ID: rp4, PointID: 4, SeqNum: entities.SourceSeqNum + 3},
		},
	}

	loadedPkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   rp1,
		UnloadPointID: rp3,
		Direction:     values.PackageDemandDirectionForward,
	})
	_ = loadedPkg.Loaded()

	notLoadedPkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   rp2,
		UnloadPointID: rp4,
		Direction:     values.PackageDemandDirectionForward,
	})

	pkgs := []*entities.Package{loadedPkg, notLoadedPkg}

	err := r.StripPointsByLoadedPackages(pkgs)
	if err != nil {
		t.Fatal(err)
	}

	// Should preserve:
	// rp1 (load point for loadedPkg)
	// rp2 (load point for notLoadedPkg)
	// rp3 (unload point for loadedPkg - because the package is loaded)
	// rp4 should be removed (unload point for notLoadedPkg - because it is NOT loaded)

	if len(r.Points) != 4 {
		t.Fatalf("expected 4 preserved points, got %d", len(r.Points))
	}

	found := make(map[values.RoutePointID]bool)
	for _, rp := range r.Points {
		found[rp.ID] = true
	}
	if !found[rp1] || !found[rp2] || !found[rp3] || !found[rp4] {
		t.Fatal("missing expected route points")
	}
}

func TestRoute_StripPointsByLoadedPackages_EmptyPackages(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
		},
	}

	err := r.StripPointsByLoadedPackages(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Points) != 2 {
		t.Fatalf("expected 2 preserved points for empty packages (source + target), got %d", len(r.Points))
	}
}

func TestRoute_FilterPointFunc(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	// Remove the middle point
	r.FilterPointFunc(func(rp *entities.RoutePoint) bool {
		return rp.PointID == 2
	})

	if len(r.Points) != 2 {
		t.Fatalf("expected 2 points after filter, got %d", len(r.Points))
	}
	if r.Points[0].PointID != 1 || r.Points[1].PointID != 3 {
		t.Fatal("wrong points after filter")
	}
	// Sequence should be rebuilt
	if r.Points[0].SeqNum != entities.SourceSeqNum || r.Points[1].SeqNum != entities.SourceSeqNum+1 {
		t.Fatal("sequence not rebuilt after filter")
	}
}

func TestRoute_RebuildSequence(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: 42},
			{ID: values.NewID(), PointID: 2, SeqNum: 17},
			{ID: values.NewID(), PointID: 3, SeqNum: 99},
		},
	}

	r.RebuildSequence()

	if r.Points[0].SeqNum != entities.SourceSeqNum ||
		r.Points[1].SeqNum != entities.SourceSeqNum+1 ||
		r.Points[2].SeqNum != entities.SourceSeqNum+2 {
		t.Fatal("sequence not properly rebuilt")
	}
	// Original order should be preserved by SeqNum (sorted ascending)
	if r.Points[0].PointID != 2 || r.Points[1].PointID != 1 || r.Points[2].PointID != 3 {
		t.Fatal("points not sorted by original SeqNum")
	}
}

func TestRoute_FindPointByID(t *testing.T) {
	id := values.NewID()
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: id, PointID: 1},
			{ID: values.NewID(), PointID: 2},
		},
	}

	found, ok := r.FindPointByID(id)
	if !ok {
		t.Fatal("expected to find point by ID")
	}
	if found.PointID != 1 {
		t.Fatalf("expected point ID 1, got %d", found.PointID)
	}
}

func TestRoute_FindPointByID_NotFound(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1},
		},
	}

	_, ok := r.FindPointByID(values.NewID())
	if ok {
		t.Fatal("expected not found")
	}
}

func TestRoute_FindPointByID_Empty(t *testing.T) {
	r := &entities.Route{}

	_, ok := r.FindPointByID(values.NewID())
	if ok {
		t.Fatal("expected not found for empty route")
	}
}

func TestRoute_Tail(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 2, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 3, SeqNum: entities.SourceSeqNum + 2},
			{ID: values.NewID(), PointID: 4, SeqNum: entities.SourceSeqNum + 3},
		},
	}

	tail := r.Tail(entities.SourceSeqNum + 1) // after point 2
	if len(tail) != 2 {
		t.Fatalf("expected 2 tail points, got %d", len(tail))
	}
	if tail[0].PointID != 3 || tail[1].PointID != 4 {
		t.Fatal("tail has wrong points")
	}
}

func TestRoute_Tail_Empty(t *testing.T) {
	r := &entities.Route{
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 1, SeqNum: entities.SourceSeqNum},
		},
	}

	tail := r.Tail(entities.SourceSeqNum)
	if len(tail) != 0 {
		t.Fatalf("expected empty tail, got %d points", len(tail))
	}
}
