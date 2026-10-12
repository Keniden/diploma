package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/queries/driver/basket"
)

// ListItems returns the basket items of the driver resolved from the request
// context, in insertion order.
func (a *Application) ListItems(ctx context.Context) ([]basket.Item, error) {
	driver, err := a.driversProvider.GetDriverFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get driver from context: %w", err)
	}

	items, err := a.basketStore.ListItemsByDriverID(ctx, uuid.UUID(driver.ID))
	if err != nil {
		return nil, fmt.Errorf("list basket items by driver: %w", err)
	}

	return items, nil
}