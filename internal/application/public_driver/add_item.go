package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	lmterrors "last-mile-task/internal/errors"
)

// AddItem adds a route to the driver's basket. An unknown/foreign route maps
// to NotFound; a duplicate is a no-op (idempotent per (driver_id, item_id)).
func (a *Application) AddItem(ctx context.Context, routeID int64) error {
	if routeID <= 0 {
		return lmterrors.NewBadRequestError("route id is required")
	}

	driver, err := a.driversProvider.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("get driver from context: %w", err)
	}

	if err := a.basketStore.AddItem(ctx, uuid.UUID(driver.ID), routeID); err != nil {
		return fmt.Errorf("add basket item: %w", err)
	}

	return nil
}