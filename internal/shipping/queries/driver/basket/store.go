package basket

import (
	"context"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/queries/driver/basket/readmodels"
)

type Item = readmodels.BasketItem

type BasketItemsStore interface {
	ListItemsByDriverID(ctx context.Context, driverID uuid.UUID) ([]Item, error)
	AddItem(ctx context.Context, driverID uuid.UUID, routeID int64) error
	DeleteItem(ctx context.Context, driverID, itemID uuid.UUID) error
	ClearBasket(ctx context.Context, driverID uuid.UUID) error
}
