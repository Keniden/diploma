//go:build e2e

// End-to-end test of the payments accrual outbox at-least-once delivery
// (ARCHITECTURE.md §23 flow 8): a payments.last_mile_shipment_accrual message
// is persisted, its delivery fails once (mock payments receiver), the reader
// retries and eventually completes the message, and the same message id is
// reused (no duplicates).
//
// Run: go test -tags e2e -run 'TestE2EPaymentsOutboxDeliveryAtLeastOnce' -v ./internal/platform/application
package application_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	paymentsclient "last-mile-task/internal/clients/payments"
	"last-mile-task/internal/platform/database"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	"last-mile-task/pkg/outbox"
)

// failNextPaymentsClient is a mock payments receiver that accepts
// payments.last_mile_shipment_accrual and fails the first N calls (then
// succeeds), simulating an external payments outage that recovers.
type failNextPaymentsClient struct {
	mu       sync.Mutex
	fail     int
	accruals []paymentsclient.LastMileShipmentAccrual
}

func (c *failNextPaymentsClient) LastMileShipmentAccrual(_ context.Context, accrual paymentsclient.LastMileShipmentAccrual) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.accruals = append(c.accruals, accrual)
	if c.fail > 0 {
		c.fail--
		return errForTest("payments down")
	}
	return nil
}

func (c *failNextPaymentsClient) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.accruals)
}

func (c *failNextPaymentsClient) call(i int) paymentsclient.LastMileShipmentAccrual {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.accruals[i]
}

// TestE2EPaymentsOutboxDeliveryAtLeastOnce proves the at-least-once delivery
// of a payments.last_mile_shipment_accrual message to the payments receiver:
// a fail-once client is retried by the reader until it succeeds (COMPLETED),
// the message is delivered at least twice, and no duplicate row is created.
func TestE2EPaymentsOutboxDeliveryAtLeastOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	repo, admin := testDocumentsRepo(t, ctx, "last_mile_e2e_payments")
	shipmentID := uuid.New()

	accrual := paymentsclient.LastMileShipmentAccrual{
		EventID:    uuid.NewString(),
		ShipmentID: shipmentID,
	}
	payload, err := json.Marshal(accrual)
	if err != nil {
		t.Fatalf("marshal accrual: %v", err)
	}

	if _, err := repo.Pool.Exec(ctx,
		`INSERT INTO lmt.outbox (id, type, payload, metadata, attempt, scheduled_at, status, err)
		 VALUES ($1, $2, $3, '{}', 0, NOW() - interval '1 minute', 'TODO', '')`,
		uuid.New(), payments_outbox.MessageTypeLastMileShipmentAccrual, payload,
	); err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	client := &failNextPaymentsClient{fail: 1}

	reader := outbox.NewReader(
		outbox.NewPostgresOutbox(database.NewOutboxAdapter(repo), "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var got paymentsclient.LastMileShipmentAccrual
			if err := json.Unmarshal(message.Payload, &got); err != nil {
				return err
			}
			return client.LastMileShipmentAccrual(ctx, got)
		}),
		outbox.ReaderWithMessageType(payments_outbox.MessageTypeLastMileShipmentAccrual),
		outbox.ReaderWithInterval(10*time.Millisecond),
		outbox.ReaderWithDelayFunc(func(attempt int) time.Duration {
			return time.Duration(attempt) * time.Millisecond
		}),
	)
	reader.Start()

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		if err := admin.QueryRowContext(ctx,
			`SELECT status FROM lmt.outbox WHERE type = $1`,
			payments_outbox.MessageTypeLastMileShipmentAccrual,
		).Scan(&status); err == nil && status == "COMPLETED" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	stopCtx, stopCancel := context.WithTimeout(ctx, 2*time.Second)
	defer stopCancel()
	_ = reader.Stop(stopCtx)

	var status string
	if err := admin.QueryRowContext(ctx,
		`SELECT status FROM lmt.outbox WHERE type = $1`,
		payments_outbox.MessageTypeLastMileShipmentAccrual,
	).Scan(&status); err != nil {
		t.Fatalf("outbox status: %v", err)
	}
	if status != "COMPLETED" {
		t.Fatalf("expected outbox message COMPLETED, got %q", status)
	}

	// At-least-once: the fail-once receiver must see ≥2 attempts and, on the
	// final success, the same message id (idempotent, no duplicate delivery).
	if got := client.count(); got < 2 {
		t.Fatalf("expected at-least-two delivery attempts (failed then retried), got %d", got)
	}
	for i := 1; i < client.count(); i++ {
		if client.call(i).EventID != client.call(0).EventID {
			t.Fatalf("event id changed across retries: %q vs %q", client.call(0).EventID, client.call(i).EventID)
		}
	}

	// No duplicate outbox row; exactly one message, completed.
	assertOutboxCount(t, ctx, admin, payments_outbox.MessageTypeLastMileShipmentAccrual, "TODO", 0)
	assertOutboxCount(t, ctx, admin, payments_outbox.MessageTypeLastMileShipmentAccrual, "COMPLETED", 1)
}
