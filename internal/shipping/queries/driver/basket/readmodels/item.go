package readmodels

import (
	"time"

	"github.com/google/uuid"
)

type BasketItem struct {
	ItemID    uuid.UUID
	RouteID   int64
	RouteName string
	CreatedAt time.Time
	OfficeID  int64
}
