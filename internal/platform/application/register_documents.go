package application

import (
	docservices "last-mile-task/internal/documents/domain/services"
	documentspg "last-mile-task/internal/documents/infrastructure/adapters/postgres"
	"last-mile-task/internal/platform/database"
	docs_outbox "last-mile-task/internal/platform/outbox/documents"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/infrastructure/eventdeps"
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
)

// registerDocuments wires the documents vertical slice into the synchronous
// in-memory event bus: the documents handler subscribes to ShipmentStarted
// (forward waybill) and PointCompleted (conditionally the return waybill),
// building and persisting a Document and appending the
// documents.publish_after_start_shipment outbox message inside the same
// transaction as the triggering event.
func (a *Application) registerDocuments(
	bus *inmem.SyncEventBus,
	repo *database.Repository,
	outboxAdapter *database.OutboxAdapter,
	eventStore *eventdeps.Store,
) {
	documentsWriter := docs_outbox.NewWriter(outboxAdapter)
	documentsRepo := documentspg.NewRepository(repo)

	documentsEventHandler := handlers.NewDocumentsEventHandler(
		eventStore,
		eventdeps.NewOffices(),
		eventdeps.NewUsers(),
		docservices.NewDocumentBuilderService(),
		documentsRepo,
		documentsWriter,
	)

	bus.SubscribeShipmentEvent(events.ShipmentStarted,
		inmem.TypedHandler(documentsEventHandler.HandleShipmentStartedEvent),
	)
	bus.SubscribeRouteEvent(events.PointCompleted,
		inmem.TypedHandler(documentsEventHandler.HandlePointCompletedEvent),
	)
}