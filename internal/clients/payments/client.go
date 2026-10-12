// Package payments is the seam to the external payments gRPC service:
// last-mile shipment accruals. The external SDK/proto is not ported, so a
// narrow PaymentsClient interface is used; the real client replaces this
// adapter once ported (see TODO(LASTMILE-099)).
package payments

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// LastMileShipmentAccrual is the accrual payload for a completed last-mile
// shipment. Field names and the JSON layout mirror the payments
// last_mile_shipment_accrual contract; they must be confirmed against the
// real client when it is ported.
//
// TODO(LASTMILE-099): switch to the generated payments client types
// (LastMileShipmentAccrual request) once the external proto/SDK is ported.
type LastMileShipmentAccrual struct {
	EventID    string    `json:"event_uuid"`
	ShipmentID uuid.UUID `json:"shipment_id"`
}

// PaymentsClient is the narrow surface the payments outbox reader needs to
// deliver a last-mile shipment accrual to the payments service.
type PaymentsClient interface {
	LastMileShipmentAccrual(ctx context.Context, accrual LastMileShipmentAccrual) error
}

// Client is the slice-scoped stand-in for the payments service.
//
// TODO(LASTMILE-099): replace with the client-backed adapter once the
// internal/clients/payments/v1 (generated proto) is ported. Calls are logged
// and discarded, keeping the delivery path green.
type Client struct{}

// NewClient builds the payments stub.
func NewClient() PaymentsClient { return &Client{} }

// LastMileShipmentAccrual logs the outbound call (client not ported).
func (c *Client) LastMileShipmentAccrual(_ context.Context, accrual LastMileShipmentAccrual) error {
	slog.Debug("stub payments: LastMileShipmentAccrual",
		"event_uuid", accrual.EventID,
		"shipment_id", accrual.ShipmentID.String(),
	)

	return nil
}