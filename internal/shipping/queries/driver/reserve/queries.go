// Package reserve contains the driver-facing reserve queries.
//
// TODO(LASTMILE-033): the real repository implements these queries as a
// concrete type embedding *database.Repository and building SQL via
// squirrel (github.com/Masterminds/squirrel). The skeleton has neither
// internal/platform/database nor squirrel in go.mod, so only the query
// surface is preserved here as an interface. Once the database layer and
// the SQL builder land, port the concrete implementation 1:1.
package reserve

import (
	"context"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/queries/driver/reserve/readmodels"
)

// ReserveStore is the query surface of the real repository's reserve Queries.
type ReserveStore interface {
	GetActiveReserveByShipmentID(ctx context.Context, shipmentID uuid.UUID) (*readmodels.ReserveInfo, error)
	GetExpiredActiveReserveByRoutePointID(ctx context.Context, shipmentID, routePointID uuid.UUID) (*readmodels.ReserveInfo, error)
	GetReserveByShipmentID(ctx context.Context, shipmentID uuid.UUID) (readmodels.ReserveInfo, error)
}
