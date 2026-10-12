package application

import (
	"log/slog"

	publicdriver "last-mile-task/internal/application/public_driver"
	drivergrpc "last-mile-task/internal/application/public_driver/transport/grpc"
	"last-mile-task/internal/platform/database"
	platformgrpc "last-mile-task/internal/platform/grpc"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	shipping_application "last-mile-task/internal/shipping/application"
	shipping_demand "last-mile-task/internal/shipping/application/demand"
	shipping_scanner "last-mile-task/internal/shipping/application/scanner"
	"last-mile-task/internal/shipping/domain/events"
	shipping_policies "last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/infrastructure"
	pgdriver "last-mile-task/internal/shipping/infrastructure/driver/postgres"
	"last-mile-task/internal/shipping/infrastructure/eventdeps"
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	cargo_handlers "last-mile-task/internal/shipping/infrastructure/events/handlers/cargo"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
	basketpg "last-mile-task/internal/shipping/queries/driver/basket/basketpg"
	showcasepg "last-mile-task/internal/shipping/queries/driver/showcase/showcasepg"
	"last-mile-task/internal/shipping/queries/driver/shipment"
)

// CreateDriver wires the public driver gRPC server: aggregate repositories,
// the sync event bus with outbox handlers (cargo + WMS), the scanner and the
// package sync service, and the driver read queries.
//
// The wiring mirrors the real register_driver.go, narrowed to the load/
// unload slice: external clients (cargo-core, package-content, office, users,
// TSP, payments, documents) are replaced by slice-scoped stubs — see
// infrastructure.Provider and eventdeps.
func (a *Application) CreateDriver() (platformgrpc.Service, error) {
	repo, err := a.builder.Repository()
	if err != nil {
		return nil, err
	}
	a.repo = repo

	// Aggregate repository set shared by the application layer.
	entityRepository := infrastructure.NewEntityRepository(repo)

	// Synchronous in-process event bus: the scanner publishes package scan
	// events; handlers translate them into outbox messages within the same
	// transaction.
	shippingBus := inmem.NewSyncEventBus()

	// Outbox table access shared by the handlers and the event bus.
	outboxAdapter := database.NewOutboxAdapter(repo)
	wmsWriter := wms_outbox.NewWriter(outboxAdapter)
	paymentsWriter := payments_outbox.NewWriter(outboxAdapter)

	// Event handlers: package scans → cargo.packages.scanned + WMS loads.
	eventStore := eventdeps.NewStore(repo)
	warehousePublisher := warehouse.NewPublisher(
		eventStore,
		wmsWriter,
		eventdeps.NewOffices(),
		eventdeps.NewUsers(),
	)
	cargoPublisher := cargo_handlers.NewPublisher(outboxAdapter)
	packageEventHandler := handlers.NewPackageEventHandler(eventStore, warehousePublisher, cargoPublisher)
	shippingBus.SubscribePackageEvent(events.PackageScanned,
		inmem.TypedHandler(packageEventHandler.HandlePackageScannedEvent),
	)

	// Shipment lifecycle events → cargo-shipment (stub until the client is
	// ported, see infrastructure.CargoShipment). The completed event also
	// writes the payments.last_mile_shipment_accrual outbox message.
	shipmentEventHandler := handlers.NewShipmentEventHandler(infrastructure.NewCargoShipment(), paymentsWriter)
	shippingBus.SubscribeShipmentEvent(events.ShipmentCreated,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentCreatedEvent),
	)
	shippingBus.SubscribeShipmentEvent(events.ShipmentDriverAssign,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentDriverAssignEvent),
	)
	shippingBus.SubscribeShipmentEvent(events.ShipmentTransportAssign,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentTransportAssignEvent),
	)
	shippingBus.SubscribeShipmentEvent(events.ShipmentCompleted,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentCompletedEvent),
	)

	// Document generation: on ShipmentStarted (forward waybill) and
	// PointCompleted (conditional return waybill) the documents handler builds
	// and persists a Document and appends a documents.publish_after_start_shipment
	// outbox message inside the same transaction.
	a.registerDocuments(shippingBus, repo, outboxAdapter, eventStore)

	// Shipment creation services: policy over the write repositories + the
	// source-packages stub, assembler/progress/demand over the same repos.
	shipmentPolicy := shipping_policies.NewShipmentPolicy(
		infrastructure.NewShipmentPolicyRepository(
			entityRepository.ShipmentRepository,
			entityRepository.RouteRepository,
			entityRepository.PackageRepository,
			entityRepository.ReserveRepository,
			infrastructure.NewDemandProvider(),
		),
	)
	assemblerService := shipping_application.NewAssembler(
		entityRepository.ShipmentRepository,
		shipmentPolicy,
		entityRepository.RouteRepository,
		entityRepository.PackageRepository,
		shippingBus,
		repo,
	)
	progressService := shipping_application.NewShipmentProgressService(
		shipmentPolicy,
		entityRepository.ShipmentRepository,
		shippingBus,
		repo,
	)
	routePointService := shipping_application.NewRoutePointService(
		entityRepository.RouteRepository,
		entityRepository.PackageRepository,
		shippingBus,
		progressService,
		repo,
	)
	demandService := shipping_demand.NewDemandService(
		entityRepository.RouteRepository,
		entityRepository.PackageRepository,
		infrastructure.NewDemandProvider(),
		repo,
	)

	// Application services.
	scannerService := shipping_scanner.NewPackageScanner(
		entityRepository.PackageRepository,
		shippingBus,
		repo,
	)
	packageSyncService := shipping_application.NewPackageSyncService(
		entityRepository.ShipmentRepository,
		entityRepository.RouteRepository,
		entityRepository.PackageRepository,
		infrastructure.NewProvider(),
		repo,
	)
	reserveManager := infrastructure.NewReserveManager()
	reserveService := shipping_application.NewReserveService(
		entityRepository.ReserveRepository,
		entityRepository.ShipmentRepository,
		entityRepository.RouteRepository,
		reserveManager,
		shipmentPolicy,
		shippingBus,
		repo,
	)
	shippingQueries := shipment.NewQueries(
		pgdriver.New(repo),
		infrastructure.NewAddressResolver(),
	)

	// The showcase reads the deterministic office/route/cargo slice stubs; the
	// basket adapter shares the route provider stub to resolve route display
	// names/offices. External clients are not ported (TODO LASTMILE-107).
	routeProvider := showcasepg.NewRouteProvider(nil)
	showcaseStore := showcasepg.New(
		showcasepg.NewOfficeProvider(nil),
		routeProvider,
		showcasepg.NewCargoProvider(),
	)
	basketStore := basketpg.New(repo, routeProvider)

	handshake, err := a.createHandshakeService()
	if err != nil {
		return nil, err
	}

	driverApp := publicdriver.New(publicdriver.Config{
		ScannerService:     scannerService,
		PackageSyncService: packageSyncService,
		ShippingQueries:    shippingQueries,
		DriversProvider:    publicdriver.NewContextDriversProvider(),

		AssemblerService:       assemblerService,
		ProgressService:        progressService,
		RoutePointService:      routePointService,
		DemandService:          demandService,
		ReserveService:         reserveService,
		RouteTemplatesProvider: infrastructure.NewRouteTemplatesProvider(),
		Handshake:              handshake.(publicdriver.QRHandshake),
		Tx:                     repo,

		ShowcaseStore: showcaseStore,
		BasketStore:   basketStore,
	})

	slog.Debug("application: driver grpc server wired", "handlers", "cargo,wms,shipment")

	a.reserveExpirer = reserveService

	return drivergrpc.New(slog.Default(), driverApp), nil
}
