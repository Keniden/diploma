//go:build e2e

package application_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
	docservices "last-mile-task/internal/documents/domain/services"
	documentspg "last-mile-task/internal/documents/infrastructure/adapters/postgres"
	docs_outbox "last-mile-task/internal/platform/outbox/documents"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/infrastructure/eventdeps"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	"last-mile-task/pkg/outbox"
)

// testDocumentsRepo opens a fresh migrated test database and returns a
// database.Repository plus an admin handle for assertions.
func testDocumentsRepo(t *testing.T, ctx context.Context, dbName string) (*database.Repository, *sql.DB) {
	t.Helper()

	dsn := envOr("E2E_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/"+dbName+"?sslmode=disable")
	migrationsDir := envOr("E2E_MIGRATIONS_DIR", "../../../migrations")

	ensureDB(t, ctx, dsn, dbName)

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	freshMigrate(t, ctx, admin, migrationsDir)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return database.NewRepository(pool), admin
}

func seedDocumentShipment(
	t *testing.T,
	ctx context.Context,
	base *database.Repository,
	shipmentID uuid.UUID,
) (source, dest uuid.UUID) {
	t.Helper()

	source, dest = uuid.New(), uuid.New()
	routeID := uuid.New()

	seed := []string{
		`INSERT INTO lmt.shipment (shipment_id, shipment_short_id, shipment_status, driver_id, car_license_plate)
		 VALUES ('` + shipmentID.String() + `', 900001, 4, '` + uuid.New().String() + `', 'E777AA77')`,
		`INSERT INTO lmt.route (route_id, shipment_id, routecar_id)
		 VALUES ('` + routeID.String() + `', '` + shipmentID.String() + `', 1)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + source.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1001, 1, 0)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + dest.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1002, 2, 0)`,
	}
	for _, stmt := range seed {
		if _, err := base.Pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("seed shipment: %v\n%s", err, stmt)
		}
	}

	return source, dest
}

func seedForwardPackage(
	t *testing.T,
	ctx context.Context,
	base *database.Repository,
	shipmentID, source, dest uuid.UUID,
) uuid.UUID {
	t.Helper()

	pkg := uuid.New()
	rows := []string{
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		 VALUES ('` + pkg.String() + `', '` + shipmentID.String() + `', 2, 42, 'BOX', false)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkg.String() + `', '` + source.String() + `', '` + shipmentID.String() + `', 1, 1)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkg.String() + `', '` + dest.String() + `', '` + shipmentID.String() + `', 2, 1)`,
	}
	for _, stmt := range rows {
		if _, err := base.Pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("seed forward package: %v\n%s", err, stmt)
		}
	}

	return pkg
}

func seedReturnPackageAt(
	t *testing.T,
	ctx context.Context,
	base *database.Repository,
	shipmentID, loadPointID, unloadPointID uuid.UUID,
) uuid.UUID {
	t.Helper()

	pkg := uuid.New()
	rows := []string{
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		 VALUES ('` + pkg.String() + `', '` + shipmentID.String() + `', 2, 43, 'BOX', true)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkg.String() + `', '` + loadPointID.String() + `', '` + shipmentID.String() + `', 1, 2)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkg.String() + `', '` + unloadPointID.String() + `', '` + shipmentID.String() + `', 2, 1)`,
	}
	for _, stmt := range rows {
		if _, err := base.Pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("seed return package: %v\n%s", err, stmt)
		}
	}

	return pkg
}

// documentsHandler is the narrow surface the e2e drives (implemented by the
// unexported documentsEventHandler).
type documentsHandler interface {
	HandleShipmentStartedEvent(ctx context.Context, event events.ShipmentStartedEvent) error
	HandlePointCompletedEvent(ctx context.Context, event events.PointCompletedEvent) error
}

func newDocumentsHandler(repo *database.Repository) documentsHandler {
	store := eventdeps.NewStore(repo)
	return handlers.NewDocumentsEventHandler(
		store,
		eventdeps.NewOffices(),
		eventdeps.NewUsers(),
		docservices.NewDocumentBuilderService(),
		documentspg.NewRepository(repo),
		docs_outbox.NewWriter(database.NewOutboxAdapter(repo)),
	)
}

// TestE2EDocumentsForwardOnStart verifies that a ShipmentStarted event
// produces a forward waybill document (lmt.documents + lmt.document_package)
// and a documents.publish_after_start_shipment outbox message, all within the
// same transaction.
func TestE2EDocumentsForwardOnStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	repo, admin := testDocumentsRepo(t, ctx, "last_mile_e2e_docs")
	shipmentID := uuid.New()
	source, dest := seedDocumentShipment(t, ctx, repo, shipmentID)
	seedForwardPackage(t, ctx, repo, shipmentID, source, dest)

	h := newDocumentsHandler(repo)

	txCtx, err := repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := h.HandleShipmentStartedEvent(txCtx, events.ShipmentStartedEvent{ID: shipmentID}); err != nil {
		t.Fatalf("handle shipment started: %v", err)
	}
	if err := repo.Commit(txCtx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	assertDocsCount(t, ctx, admin, shipmentID, 1)
	assertDocPackages(t, ctx, admin, 1)
	assertDocumentsOutboxCount(t, ctx, admin, docs_outbox.MessageTypePublishAfterStartShipment, "TODO", 1)
}

// TestE2EDocumentsReturnConditional verifies that a PointCompleted event
// publishes a return waybill only when the point has loaded return packages.
func TestE2EDocumentsReturnConditional(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	repo, admin := testDocumentsRepo(t, ctx, "last_mile_e2e_docs_return")
	shipmentID := uuid.New()
	source, dest := seedDocumentShipment(t, ctx, repo, shipmentID)

	// No return packages at dest → nothing published.
	h := newDocumentsHandler(repo)
	if err := h.HandlePointCompletedEvent(ctx, events.PointCompletedEvent{ShipmentID: shipmentID, RoutePointID: dest}); err != nil {
		t.Fatalf("handle point completed (no return): %v", err)
	}
	assertDocsCount(t, ctx, admin, shipmentID, 0)
	assertDocumentsOutboxCount(t, ctx, admin, docs_outbox.MessageTypePublishAfterStartShipment, "TODO", 0)

	// Add a loaded return package at dest → return waybill published.
	seedReturnPackageAt(t, ctx, repo, shipmentID, dest, source)
	if err := h.HandlePointCompletedEvent(ctx, events.PointCompletedEvent{ShipmentID: shipmentID, RoutePointID: dest}); err != nil {
		t.Fatalf("handle point completed (with return): %v", err)
	}
	assertDocsCount(t, ctx, admin, shipmentID, 1)
	assertDocumentsOutboxCount(t, ctx, admin, docs_outbox.MessageTypePublishAfterStartShipment, "TODO", 1)

	var docType int
	if err := admin.QueryRowContext(ctx,
		`SELECT document_type FROM lmt.documents WHERE shipment_id = $1`, shipmentID,
	).Scan(&docType); err != nil {
		t.Fatalf("select document_type: %v", err)
	}
	if docType != 2 {
		t.Fatalf("expected RETURN document type 2, got %d", docType)
	}
}

// failNextClient is a cargo-documents mock that fails the first N calls.
type failNextClient struct {
	mu      sync.Mutex
	fail    int
	events  []cargodocuments.Event
}

func (c *failNextClient) CreateEventByType(_ context.Context, event cargodocuments.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, event)
	if c.fail > 0 {
		c.fail--
		return errForTest("cargo-documents down")
	}
	return nil
}

func (c *failNextClient) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

// TestE2EDocumentsOutboxDeliveryAtLeastOnce verifies at-least-once delivery
// of a documents.publish_after_start_shipment message to cargo-documents: on
// failure the message is rescheduled, on success it is completed.
func TestE2EDocumentsOutboxDeliveryAtLeastOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	repo, admin := testDocumentsRepo(t, ctx, "last_mile_e2e_docs_delivery")
	shipmentID := uuid.New()

	event := cargodocuments.Event{
		EventUUID:    uuid.NewString(),
		Type:         "waybill",
		ShipmentID:   shipmentID,
		DocumentType: 1,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if _, err := repo.Pool.Exec(ctx,
		`INSERT INTO lmt.outbox (id, type, payload, metadata, attempt, scheduled_at, status, err)
		 VALUES ($1, $2, $3, '{}', 0, NOW() - interval '1 minute', 'TODO', '')`,
		uuid.New(), docs_outbox.MessageTypePublishAfterStartShipment, payload,
	); err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	client := &failNextClient{fail: 1}

	reader := outbox.NewReader(
		outbox.NewPostgresOutbox(database.NewOutboxAdapter(repo), "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var decoded cargodocuments.Event
			if err := json.Unmarshal(message.Payload, &decoded); err != nil {
				return err
			}
			return client.CreateEventByType(ctx, decoded)
		}),
		outbox.ReaderWithMessageType(docs_outbox.MessageTypePublishAfterStartShipment),
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
			docs_outbox.MessageTypePublishAfterStartShipment,
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
		docs_outbox.MessageTypePublishAfterStartShipment,
	).Scan(&status); err != nil {
		t.Fatalf("outbox status: %v", err)
	}
	if status != "COMPLETED" {
		t.Fatalf("expected outbox message COMPLETED, got %q", status)
	}
	if got := client.count(); got < 2 {
		t.Fatalf("expected at-least-two delivery attempts (failed then retried), got %d", got)
	}
}

func assertDocsCount(t *testing.T, ctx context.Context, admin *sql.DB, shipmentID uuid.UUID, want int) {
	t.Helper()

	var got int
	if err := admin.QueryRowContext(ctx,
		`SELECT count(*) FROM lmt.documents WHERE shipment_id = $1`, shipmentID,
	).Scan(&got); err != nil {
		t.Fatalf("documents count: %v", err)
	}
	if got != want {
		t.Fatalf("documents count: expected %d, got %d", want, got)
	}
}

func assertDocPackages(t *testing.T, ctx context.Context, admin *sql.DB, want int) {
	t.Helper()

	var got int
	if err := admin.QueryRowContext(ctx,
		`SELECT count(*) FROM lmt.document_package`,
	).Scan(&got); err != nil {
		t.Fatalf("document_package count: %v", err)
	}
	if got != want {
		t.Fatalf("document_package count: expected %d, got %d", want, got)
	}
}

func assertDocumentsOutboxCount(t *testing.T, ctx context.Context, admin *sql.DB, messageType, status string, want int) {
	t.Helper()

	var got int
	if err := admin.QueryRowContext(ctx,
		`SELECT count(*) FROM lmt.outbox WHERE type = $1 AND status = $2`, messageType, status,
	).Scan(&got); err != nil {
		t.Fatalf("outbox count: %v", err)
	}
	if got != want {
		t.Fatalf("outbox %s/%s: expected %d rows, got %d", messageType, status, want, got)
	}
}

// errForTest is a plain test error.
func errForTest(msg string) error {
	return &testErr{msg: msg}
}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }