package publicdriver

import (
	"context"
	"fmt"

	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/queries/driver/showcase"
)

// ListRoutesByPointRequest carries the source point and the cursor pagination
// of the showcase route page.
type ListRoutesByPointRequest struct {
	PointID   int64
	PageSize  int64
	PageToken int64
}

// ListRoutesByPoint returns a cursor-paginated page of routes serving a point.
func (a *Application) ListRoutesByPoint(
	ctx context.Context,
	req ListRoutesByPointRequest,
) ([]showcase.ShowcaseRoute, primitives.PaginationResponse[int64], error) {
	if req.PointID <= 0 {
		return nil, primitives.PaginationResponse[int64]{}, lmterrors.NewBadRequestError("point id is required")
	}

	page, err := primitives.NewPage(req.PageSize, req.PageToken)
	if err != nil {
		return nil, primitives.PaginationResponse[int64]{}, lmterrors.NewBadRequestError(err.Error())
	}

	routes, resp, err := a.showcaseStore.ListRoutesByPoint(ctx, page, req.PointID)
	if err != nil {
		return nil, primitives.PaginationResponse[int64]{}, fmt.Errorf("list routes by point: %w", err)
	}

	return routes, resp, nil
}