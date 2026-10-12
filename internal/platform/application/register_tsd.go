package application

import (
	"log/slog"

	tsdapp "last-mile-task/internal/application/tsd"
	"last-mile-task/internal/platform/database"
	platformgrpc "last-mile-task/internal/platform/grpc"
	grpctsd "last-mile-task/internal/platform/grpc/tsd"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	shipping_application "last-mile-task/internal/shipping/application"
	shipping_scanner "last-mile-task/internal/shipping/application/scanner"
	"last-mile-task/internal/shipping/domain/events"
	shipping_policies "last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/infrastructure"
	"last-mile-task/internal/shipping/infrastructure/eventdeps"
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	cargo_handlers "last-mile-task/internal/shipping/infrastructure/events/handlers/cargo"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
	"last-mile-task/internal/shipping/infrastructure/tsdpackages"
)

// registerTSD wires the TSD gRPC service: aggregate repositories, the sync
// event bus with the package-scan outbox handlers (cargo + WMS), the scanner
// and the reserve service, and the packages-to-scan read query. The dependency
// slice mirrors register_driver.go narrowed to the TSD flows 4-6.
func (a *Application) registerTSD() (platformgrpc.Service, error) {
	repo, err := a.builder.Repository()
	if err != nil {
		return nil, err
	}
	a.repo = repo

	entityRepository := infrastructure.NewEntityRepository(repo)

	// Synchronous in-process event bus: the scanner publishes package scan
	// events; handlers translate them into outbox messages within the same
	// transaction (cargo + WMS).
	shippingBus := inmem.NewSyncEventBus()

	outboxAdapter := database.NewOutboxAdapter(repo)
	wmsWriter := wms_outbox.NewWriter(outboxAdapter)

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

	scannerService := shipping_scanner.NewPackageScanner(
		entityRepository.PackageRepository,
		shippingBus,
		repo,
	)

	shipmentPolicy := shipping_policies.NewShipmentPolicy(
		infrastructure.NewShipmentPolicyRepository(
			entityRepository.ShipmentRepository,
			entityRepository.RouteRepository,
			entityRepository.PackageRepository,
			entityRepository.ReserveRepository,
			infrastructure.NewDemandProvider(),
		),
	)
	reserveService := shipping_application.NewReserveService(
		entityRepository.ReserveRepository,
		entityRepository.ShipmentRepository,
		entityRepository.RouteRepository,
		infrastructure.NewReserveManager(),
		shipmentPolicy,
		shippingBus,
		repo,
	)

	handshake, err := a.createHandshakeService()
	if err != nil {
		return nil, err
	}

	tsdApp := tsdapp.New(tsdapp.Config{
		Handshake:       handshake.(tsdapp.QRHandshake),
		Scanner:         scannerService,
		ReserveApprover: reserveService,
		RouteRepository: entityRepository.RouteRepository,
		PackagesToScan:  tsdpackages.NewStore(repo),
	})

	slog.Debug("application: tsd grpc server wired", "handlers", "cargo,wms")

	return grpctsd.New(slog.Default(), tsdApp), nil
}
