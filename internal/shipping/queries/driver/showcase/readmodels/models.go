package readmodels

import (
	"last-mile-task/internal/primitives"
)

type ShowcasePoint struct {
	ID       int64
	Name     string
	Address  string
	Location primitives.Coordinates
}

type ShowcaseRoute struct {
	ID          int64
	Name        string
	RoutePoints []ShowcaseRoutePoint
}

type ShowcaseRoutePoint struct {
	PointID       int64
	SeqNum        int32
	Name          string
	Address       string
	Location      primitives.Coordinates
	CountPackages int32
}
