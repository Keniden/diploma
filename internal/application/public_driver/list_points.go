package publicdriver

import (
	"context"
	"fmt"

	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/queries/driver/showcase"
)

// ListPointsRequest carries the cursor pagination of the showcase point page.
type ListPointsRequest struct {
	PageSize  int64
	PageToken int64
}

// ListPoints returns a cursor-paginated page of showcase points.
func (a *Application) ListPoints(
	ctx context.Context,
	req ListPointsRequest,
) ([]showcase.ShowcasePoint, primitives.PaginationResponse[int32], error) {
	page, err := primitives.NewPage(req.PageSize, req.PageToken)
	if err != nil {
		return nil, primitives.PaginationResponse[int32]{}, lmterrors.NewBadRequestError(err.Error())
	}

	points, resp, err := a.showcaseStore.ListPoints(ctx, page)
	if err != nil {
		return nil, primitives.PaginationResponse[int32]{}, fmt.Errorf("list points: %w", err)
	}

	return points, resp, nil
}