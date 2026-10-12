package showcasepg

import (
	"context"
	"fmt"
	"sort"

	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/queries/driver/showcase"
)

// Store implements [showcase.ShowcaseStore] over deterministic slice-stubs of
// the external office/route/cargo services.
//
// TODO(LASTMILE-075): the real showcase reads office (ListOffices/ListPointsV2),
// route (ListRoutecars) and cargo (ListPackagesToLoad) clients; the skeleton
// reads fixed slice stubs so the showcase stays deterministic until the clients
// are ported.
type Store struct {
	offices OfficeProvider
	routes  RouteProvider
	cargo   CargoProvider
}

// New builds the showcase store over the deterministic providers.
func New(offices OfficeProvider, routes RouteProvider, cargo CargoProvider) *Store {
	return &Store{
		offices: offices,
		routes:  routes,
		cargo:   cargo,
	}
}

// ListPoints returns a cursor-paginated page of showcase points.
func (s *Store) ListPoints(
	ctx context.Context,
	page primitives.Page[int64],
) ([]showcase.ShowcasePoint, primitives.PaginationResponse[int32], error) {
	offices, err := s.offices.ListOffices(ctx)
	if err != nil {
		return nil, primitives.PaginationResponse[int32]{}, fmt.Errorf("list offices: %w", err)
	}
	// order by id asc so the cursor pagination is deterministic.
	sort.Slice(offices, func(i, j int) bool { return offices[i].ID < offices[j].ID })

	points := make([]showcase.ShowcasePoint, 0, len(offices))
	for _, o := range offices {
		points = append(points, showcase.ShowcasePoint{
			ID:       o.ID,
			Name:     o.Name,
			Address:  o.Address,
			Location: o.Location,
		})
	}

	items, lastID, hasMore := paginatePoints(points, page)
	resp := primitives.PaginationResponse[int32]{
		Page:    primitives.Page[int32]{Size: page.Size, Token: int32(lastID)},
		HasMore: hasMore,
	}
	if hasMore {
		next := primitives.Page[int32]{Size: page.Size, Token: int32(lastID)}
		resp.NextPage = &next
	}

	return items, resp, nil
}

// ListRoutesByPoint returns a cursor-paginated page of routes serving the given
// point. An unknown point maps to NotFound.
func (s *Store) ListRoutesByPoint(
	ctx context.Context,
	page primitives.Page[int64],
	pointID int64,
) ([]showcase.ShowcaseRoute, primitives.PaginationResponse[int64], error) {
	if err := s.pointExists(ctx, pointID); err != nil {
		return nil, primitives.PaginationResponse[int64]{}, err
	}

	routes, err := s.routes.ListRoutes(ctx)
	if err != nil {
		return nil, primitives.PaginationResponse[int64]{}, fmt.Errorf("list routes: %w", err)
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].ID < routes[j].ID })

	filtered := make([]showcase.ShowcaseRoute, 0, len(routes))
	for _, r := range routes {
		if pointAppears(r.Points, pointID) {
			mapped, err := s.mapRoute(ctx, r)
			if err != nil {
				return nil, primitives.PaginationResponse[int64]{}, err
			}
			filtered = append(filtered, mapped)
		}
	}

	items, lastID, hasMore := paginateRoutes(filtered, page)
	resp := primitives.PaginationResponse[int64]{
		Page:    primitives.Page[int64]{Size: page.Size, Token: lastID},
		HasMore: hasMore,
	}
	if hasMore {
		next := primitives.Page[int64]{Size: page.Size, Token: lastID}
		resp.NextPage = &next
	}

	return items, resp, nil
}

func (s *Store) mapRoute(ctx context.Context, r Route) (showcase.ShowcaseRoute, error) {
	out := showcase.ShowcaseRoute{
		ID:          r.ID,
		Name:        r.Name,
		RoutePoints: make([]showcase.ShowcaseRoutePoint, 0, len(r.Points)),
	}
	for _, p := range r.Points {
		count, err := s.cargo.CountPackagesByPointID(ctx, p.PointID)
		if err != nil {
			return showcase.ShowcaseRoute{}, fmt.Errorf("count packages by point %d: %w", p.PointID, err)
		}
		out.RoutePoints = append(out.RoutePoints, showcase.ShowcaseRoutePoint{
			PointID:       p.PointID,
			SeqNum:        p.SeqNum,
			Name:          p.Name,
			Address:       p.Address,
			Location:      p.Location,
			CountPackages: count,
		})
	}
	return out, nil
}

func (s *Store) pointExists(ctx context.Context, pointID int64) error {
	offices, err := s.offices.ListOffices(ctx)
	if err != nil {
		return fmt.Errorf("list offices: %w", err)
	}
	for _, o := range offices {
		if o.ID == pointID {
			return nil
		}
	}
	return lmterrors.NewNotFoundError(fmt.Sprintf("point %d not found", pointID))
}

func paginatePoints(
	points []showcase.ShowcasePoint,
	page primitives.Page[int64],
) ([]showcase.ShowcasePoint, int64, bool) {
	if len(points) == 0 {
		return points, 0, false
	}
	start := cursorStart(points, page.Token, func(p showcase.ShowcasePoint) int64 { return p.ID })
	end := clampEnd(start, int(page.Size), len(points))
	if start >= len(points) {
		return points[start:start], 0, false
	}
	return points[start:end], points[end-1].ID, end < len(points)
}

func paginateRoutes(
	routes []showcase.ShowcaseRoute,
	page primitives.Page[int64],
) ([]showcase.ShowcaseRoute, int64, bool) {
	if len(routes) == 0 {
		return routes, 0, false
	}
	start := cursorStart(routes, page.Token, func(r showcase.ShowcaseRoute) int64 { return r.ID })
	end := clampEnd(start, int(page.Size), len(routes))
	if start >= len(routes) {
		return routes[start:start], 0, false
	}
	return routes[start:end], routes[end-1].ID, end < len(routes)
}

func cursorStart[T any](items []T, token int64, id func(T) int64) int {
	if token <= 0 {
		return 0
	}
	idx := sort.Search(len(items), func(i int) bool { return id(items[i]) > token })
	return idx
}

func clampEnd(start, size, length int) int {
	end := start + size
	if end > length {
		return length
	}
	return end
}

func pointAppears(points []RoutePoint, pointID int64) bool {
	for _, p := range points {
		if p.PointID == pointID {
			return true
		}
	}
	return false
}