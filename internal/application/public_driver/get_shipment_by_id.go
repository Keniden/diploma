package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	shipment "last-mile-task/internal/shipping/queries/driver/shipment"
)

// GetShipmentByID returns the detailed shipment (route, points, reserve).
func (a *Application) GetShipmentByID(ctx context.Context, id uuid.UUID) (*shipment.DetailedShipment, error) {
	s, err := a.shippingQueries.GetDetailedShipmentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get detailed shipment by id: %w", err)
	}

	// TODO(LASTMILE-084): the real use case enriches the shipment with office
	// week schedules (officeProvider.WeekSchedulesByPointIDs + mapWeekSchedules)
	// before returning it. The skeleton has no office client and the proto
	// contract carries no schedule graph — skip enrichment until the client
	// port is available.
	return &s, nil
}
