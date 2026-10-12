package showcase

import (
	"context"

	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/queries/driver/showcase/readmodels"
)

type (
	ShowcasePoint      = readmodels.ShowcasePoint
	ShowcaseRoute      = readmodels.ShowcaseRoute
	ShowcaseRoutePoint = readmodels.ShowcaseRoutePoint
)

type ShowcaseStore interface {
	ListPoints(ctx context.Context, page primitives.Page[int64]) ([]ShowcasePoint, primitives.PaginationResponse[int32], error)
	ListRoutesByPoint(ctx context.Context, page primitives.Page[int64], pointID int64) ([]ShowcaseRoute, primitives.PaginationResponse[int64], error)
}
