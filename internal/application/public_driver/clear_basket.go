package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ClearBasket removes every item of the driver's basket. An already empty
// basket is not an error.
func (a *Application) ClearBasket(ctx context.Context) error {
	driver, err := a.driversProvider.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("get driver from context: %w", err)
	}

	if err := a.basketStore.ClearBasket(ctx, uuid.UUID(driver.ID)); err != nil {
		return fmt.Errorf("clear basket: %w", err)
	}

	return nil
}