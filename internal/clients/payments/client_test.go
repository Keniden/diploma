package payments_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	paymentsclient "last-mile-task/internal/clients/payments"
)

func TestPaymentsClientInterfaceSatisfied(t *testing.T) {
	var _ paymentsclient.PaymentsClient = &paymentsclient.Client{}
}

func TestClientLastMileShipmentAccrualIsNoop(t *testing.T) {
	client := paymentsclient.NewClient()

	err := client.LastMileShipmentAccrual(context.Background(), paymentsclient.LastMileShipmentAccrual{
		EventID:    "evt-1",
		ShipmentID: uuid.New(),
	})

	if err != nil {
		t.Fatalf("expected stub to succeed, got %v", err)
	}
}

func TestAccrualJSONRoundTrip(t *testing.T) {
	accrual := paymentsclient.LastMileShipmentAccrual{
		EventID:    "evt-1",
		ShipmentID: uuid.New(),
	}

	// The writer marshals the accrual; the reader must be able to unmarshal it
	// back into the same shape.
	var decoded paymentsclient.LastMileShipmentAccrual
	data, err := json.Marshal(accrual)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	if decoded.EventID != accrual.EventID || decoded.ShipmentID != accrual.ShipmentID {
		t.Fatal("accrual fields must survive JSON round trip")
	}
}