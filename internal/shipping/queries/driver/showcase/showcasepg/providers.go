package showcasepg

import (
	"context"
	"fmt"

	"last-mile-task/internal/primitives"
)

// Office is a deterministic slice-stub office (source point of the showcase).
//
// TODO(LASTMILE-075): the real office client (ListOffices/ListPointsV2) is not
// ported — the showcase reads a fixed set of offices so the slice keeps real
// substitution points once the client lands.
type Office struct {
	ID       int64
	Name     string
	Address  string
	Location primitives.Coordinates
}

// Route is a deterministic slice-stub route (routecar) of the showcase.
type Route struct {
	ID       int64
	Name     string
	OfficeID int64
	Points   []RoutePoint
}

// RoutePoint is a deterministic slice-stub route point of a showcase route.
type RoutePoint struct {
	PointID  int64
	SeqNum   int32
	Name     string
	Address  string
	Location primitives.Coordinates
}

// OfficeProvider exposes the showcase source points.
type OfficeProvider interface {
	ListOffices(ctx context.Context) ([]Office, error)
}

// RouteProvider exposes the showcase routes and a by-id lookup shared with the
// basket adapter (route name / office for basket items).
type RouteProvider interface {
	ListRoutes(ctx context.Context) ([]Route, error)
	GetRouteByID(ctx context.Context, id int64) (Route, error)
}

// CargoProvider exposes the package count per point of a route.
type CargoProvider interface {
	CountPackagesByPointID(ctx context.Context, pointID int64) (int32, error)
}

// sliceOfficeProvider is a deterministic in-memory office stub.
type sliceOfficeProvider struct {
	offices []Office
}

// NewOfficeProvider builds the office stub from a fixed slice. When offices is
// nil the default deterministic showcase office set is used.
func NewOfficeProvider(offices []Office) OfficeProvider {
	if offices == nil {
		offices = defaultOffices()
	}
	return &sliceOfficeProvider{offices: offices}
}

func (p *sliceOfficeProvider) ListOffices(context.Context) ([]Office, error) {
	return p.offices, nil
}

// sliceRouteProvider is a deterministic in-memory route stub.
type sliceRouteProvider struct {
	routes []Route
}

// NewRouteProvider builds the route stub from a fixed slice. When routes is nil
// the default deterministic showcase route set is used.
func NewRouteProvider(routes []Route) RouteProvider {
	if routes == nil {
		routes = defaultRoutes()
	}
	return &sliceRouteProvider{routes: routes}
}

func (p *sliceRouteProvider) ListRoutes(context.Context) ([]Route, error) {
	return p.routes, nil
}

func (p *sliceRouteProvider) GetRouteByID(_ context.Context, id int64) (Route, error) {
	for _, r := range p.routes {
		if r.ID == id {
			return r, nil
		}
	}
	return Route{}, fmt.Errorf("route %d not found", id)
}

// sliceCargoProvider is a deterministic package-count stub.
//
// TODO(LASTMILE-075): the real cargo (ListPackagesToLoad) and package-content
// (GetPackagesContent) clients are not ported — the count is derived from the
// point id so the slice stays deterministic.
type sliceCargoProvider struct{}

// NewCargoProvider builds the deterministic cargo stub.
func NewCargoProvider() CargoProvider {
	return &sliceCargoProvider{}
}

func (p *sliceCargoProvider) CountPackagesByPointID(_ context.Context, pointID int64) (int32, error) {
	return int32(10 + pointID%50), nil
}

func defaultOffices() []Office {
	return []Office{
		{ID: 1, Name: "Склад №1", Address: "г. Москва, ул. Пример, 1", Location: mustCoords(55.7501, 37.6156)},
		{ID: 2, Name: "Склад №2", Address: "г. Москва, ул. Пример, 2", Location: mustCoords(55.7602, 37.6256)},
		{ID: 3, Name: "Склад №3", Address: "г. Москва, ул. Пример, 3", Location: mustCoords(55.7703, 37.6356)},
		{ID: 4, Name: "Склад №4", Address: "г. Москва, ул. Пример, 4", Location: mustCoords(55.7804, 37.6456)},
		{ID: 5, Name: "Склад №5", Address: "г. Москва, ул. Пример, 5", Location: mustCoords(55.7905, 37.6556)},
	}
}

func defaultRoutes() []Route {
	return []Route{
		{
			ID:       101,
			Name:     "Маршрут №101",
			OfficeID: 1,
			Points: []RoutePoint{
				{PointID: 1, SeqNum: 1, Name: "Склад №1", Address: "г. Москва, ул. Пример, 1", Location: mustCoords(55.7501, 37.6156)},
				{PointID: 2, SeqNum: 2, Name: "Склад №2", Address: "г. Москва, ул. Пример, 2", Location: mustCoords(55.7602, 37.6256)},
			},
		},
		{
			ID:       102,
			Name:     "Маршрут №102",
			OfficeID: 2,
			Points: []RoutePoint{
				{PointID: 2, SeqNum: 1, Name: "Склад №2", Address: "г. Москва, ул. Пример, 2", Location: mustCoords(55.7602, 37.6256)},
				{PointID: 3, SeqNum: 2, Name: "Склад №3", Address: "г. Москва, ул. Пример, 3", Location: mustCoords(55.7703, 37.6356)},
			},
		},
		{
			ID:       103,
			Name:     "Маршрут №103",
			OfficeID: 3,
			Points: []RoutePoint{
				{PointID: 3, SeqNum: 1, Name: "Склад №3", Address: "г. Москва, ул. Пример, 3", Location: mustCoords(55.7703, 37.6356)},
				{PointID: 4, SeqNum: 2, Name: "Склад №4", Address: "г. Москва, ул. Пример, 4", Location: mustCoords(55.7804, 37.6456)},
			},
		},
	}
}

func mustCoords(lat, lng float64) primitives.Coordinates {
	c, err := primitives.NewCoordinates(lat, lng)
	if err != nil {
		panic(fmt.Sprintf("showcasepg: invalid coordinates: %v", err))
	}
	return c
}