package readmodels

import "time"

type ShipmentReserve struct {
	ExpiresAt *time.Time
	UpdatedAt time.Time
}
