package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	lmterrors "last-mile-task/internal/errors"
)

// DeleteItem removes an item of the driver's basket. A foreign or missing item
// maps to NotFound without changing the basket.
func (a *Application) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	if itemID == uuid.Nil {
		return lmterrors.NewBadRequestError("item id is required")
	}

	driver, err := a.driversProvider.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("get driver from context: %w", err)
	}

	if err := a.basketStore.DeleteItem(ctx, uuid.UUID(driver.ID), itemID); err != nil {
		return fmt.Errorf("delete basket item: %w", err)
	}

	return nil
}