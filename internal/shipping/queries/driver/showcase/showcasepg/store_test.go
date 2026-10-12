package showcasepg_test

import (
	"context"
	"testing"

	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/queries/driver/showcase/showcasepg"
)

func newStore() *showcasepg.Store {
	return showcasepg.New(
		showcasepg.NewOfficeProvider(nil),
		showcasepg.NewRouteProvider(nil),
		showcasepg.NewCargoProvider(),
	)
}

func TestListPoints_Pagination(t *testing.T) {
	ctx := context.Background()
	s := newStore()

	points, resp, err := s.ListPoints(ctx, primitives.Page[int64]{Size: 2})
	if err != nil {
		t.Fatalf("ListPoints: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points on first page, got %d", len(points))
	}
	if !resp.HasMore {
		t.Fatal("expected HasMore on the first page")
	}
	if resp.NextPage == nil {
		t.Fatal("expected a next page token")
	}
	if points[0].ID != 1 || points[1].ID != 2 {
		t.Fatalf("expected points 1,2 in order, got %d,%d", points[0].ID, points[1].ID)
	}
}

func TestListPoints_TokenContinues(t *testing.T) {
	ctx := context.Background()
	s := newStore()

	points, resp, _ := s.ListPoints(ctx, primitives.Page[int64]{Size: 2, Token: 2})
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].ID != 3 || points[1].ID != 4 {
		t.Fatalf("expected points 3,4 in order, got %d,%d", points[0].ID, points[1].ID)
	}
	if !resp.HasMore {
		t.Fatal("expected HasMore after the second page")
	}
}

func TestListPoints_Empty(t *testing.T) {
	ctx := context.Background()
	s := showcasepg.New(showcasepg.NewOfficeProvider([]showcasepg.Office{}), showcasepg.NewRouteProvider(nil), showcasepg.NewCargoProvider())

	points, _, err := s.ListPoints(ctx, primitives.Page[int64]{Size: 2})
	if err != nil {
		t.Fatalf("empty showcase must not error: %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("expected no points, got %d", len(points))
	}
}

func TestListPoints_LastPage(t *testing.T) {
	ctx := context.Background()
	s := newStore()

	// 4th page with size 2: only point 5 remains after token 4.
	points, resp, _ := s.ListPoints(ctx, primitives.Page[int64]{Size: 2, Token: 4})
	if len(points) != 1 {
		t.Fatalf("expected 1 point on the last page, got %d", len(points))
	}
	if points[0].ID != 5 {
		t.Fatalf("expected point 5, got %d", points[0].ID)
	}
	if resp.HasMore {
		t.Fatal("expected no more pages")
	}
	if resp.NextPage != nil {
		t.Fatal("expected no next page on the last page")
	}
}

func TestListRoutesByPoint_Success(t *testing.T) {
	ctx := context.Background()
	s := newStore()

	// Point 2 is served by routes 101 and 102.
	routes, resp, err := s.ListRoutesByPoint(ctx, primitives.Page[int64]{Size: 10}, 2)
	if err != nil {
		t.Fatalf("ListRoutesByPoint: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes for point 2, got %d", len(routes))
	}
	if routes[0].ID != 101 || routes[1].ID != 102 {
		t.Fatalf("expected routes 101,102, got %d,%d", routes[0].ID, routes[1].ID)
	}
	if resp.HasMore {
		t.Fatal("expected no more routes")
	}
	if len(routes[0].RoutePoints) != 2 {
		t.Fatalf("expected 2 route points on route 101, got %d", len(routes[0].RoutePoints))
	}
	if routes[0].RoutePoints[0].CountPackages <= 0 {
		t.Fatalf("expected a deterministic package count, got %d", routes[0].RoutePoints[0].CountPackages)
	}
}

func TestListRoutesByPoint_UnknownPoint(t *testing.T) {
	ctx := context.Background()
	s := newStore()

	_, _, err := s.ListRoutesByPoint(ctx, primitives.Page[int64]{Size: 10}, 999)
	if err == nil {
		t.Fatal("expected an error for an unknown point")
	}
}

func TestListRoutesByPoint_Pagination(t *testing.T) {
	ctx := context.Background()
	s := newStore()

	_, resp, err := s.ListRoutesByPoint(ctx, primitives.Page[int64]{Size: 1}, 2)
	if err != nil {
		t.Fatalf("ListRoutesByPoint: %v", err)
	}
	if !resp.HasMore {
		t.Fatal("expected HasMore with a page size of 1 and 2 routes")
	}
}