// Package application wires and runs the service: observability, gRPC
// servers and background workers over the dependencies built by
// internal/platform/application/builder.
package application

import (
	"context"
	"log/slog"
	"time"

	"last-mile-task/internal/platform/application/builder"
	"last-mile-task/internal/platform/database"
	platformgrpc "last-mile-task/internal/platform/grpc"
	"last-mile-task/internal/platform/observability"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/wmsmock"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
	paymentsclient "last-mile-task/internal/clients/payments"
)

const (
	dbProbeInterval = 5 * time.Second
	dbProbeTimeout  = 3 * time.Second
	shutdownGrace   = 10 * time.Second
)

// Application is the assembled service.
//
// The driver wiring lives in register_driver.go and holds its dependency
// slice locally; the fields below track the exporter's mock/client used at
// shutdown.
type Application struct {
	builder    *builder.Builder
	grpc       *platformgrpc.Server
	obs        *observability.Server
	exporter   *wmsExporter
	mockServer *wmsmock.Server

	documentsExporter *documentsExporter
	paymentsExporter  *paymentsExporter

	repo      *database.Repository
	wmsClient *wms.Client

	// reserveExpirer expires due reserves on a schedule (wired in
	// register_driver.go once the reserve service is built).
	reserveExpirer ports.ReserveExpirer
}

// New builds the servers.
func New(b *builder.Builder) (*Application, error) {
	cfg := b.Config()

	return &Application{
		builder: b,
		grpc: platformgrpc.NewServer(slog.Default(), platformgrpc.ServerConfig{
			Addr: cfg.GRPC.Addr,
		}),
		obs: observability.NewServer(observability.Config{
			Addr:    cfg.Lifecycle.Addr,
			Version: b.Version(),
			Logger:  slog.Default(),
		}),
	}, nil
}

// Run starts the lifecycle HTTP server, the gRPC server and the WMS exporter
// worker, then blocks until the context is canceled. Shutdown stops
// everything afterwards.
func (a *Application) Run(ctx context.Context) error {
	cfg := a.builder.Config()

	a.registerServers()

	if err := a.startObservability(); err != nil {
		return err
	}

	if err := a.startExporter(); err != nil {
		return err
	}

	if err := a.startDocumentsExporter(); err != nil {
		return err
	}

	if err := a.startPaymentsExporter(); err != nil {
		return err
	}

	a.startReserveExpirer()

	if err := a.startGRPC(ctx); err != nil {
		return err
	}

	slog.Info("last-mile-task: service is up",
		"grpc_addr", cfg.GRPC.Addr,
		"lifecycle_addr", cfg.Lifecycle.Addr,
		"mock_enabled", cfg.Mock.Enabled,
	)

	<-ctx.Done()

	return nil
}

func (a *Application) registerServers() {
	driver, err := a.CreateDriver()
	if err != nil {
		slog.Error("application: driver wiring failed", "error", err)
	}

	services := []platformgrpc.Service{
		a.registerMaintenance(),
		a.registerPortalShipment(),
		a.registerSupport(),
	}

	if tsd, err := a.registerTSD(); err != nil {
		slog.Error("application: tsd wiring failed", "error", err)
	} else {
		services = append(services, tsd)
	}

	if driver != nil {
		services = append(services, driver)
	}

	a.grpc.RegisterServices(services...)
}

func (a *Application) startObservability() error {
	// Readiness/liveness probe: the service can serve traffic only when the
	// database is reachable.
	ping := a.builder.Ping

	a.obs.AddHealthCheck("database", func() error {
		return ping(context.Background())
	})
	a.obs.AddReadinessCheck("database", dbProbeInterval, dbProbeTimeout, func(ctx context.Context) error {
		return ping(ctx)
	})

	go func() {
		if err := a.obs.ListenAndServe(); err != nil {
			slog.Error("application: observability server", "error", err)
		}
	}()

	return nil
}

func (a *Application) startExporter() error {
	querier, err := a.builder.Outbox()
	if err != nil {
		return err
	}

	cfg := a.builder.Config()

	client, mock, err := newWMSClient(cfg.WMS, cfg.Mock.Enabled)
	if err != nil {
		return err
	}

	if mock != nil {
		a.mockServer = mock
		a.builder.OnShutdown(func() { _ = mock.Close() })
	}

	a.wmsClient = client

	a.exporter = newWMSEporter(slog.Default(), newWMSReaders(querier, client))
	a.exporter.Start()

	a.builder.StartBackground(func(ctx context.Context) {
		a.exporter.Run(ctx)
	})

	return nil
}

func (a *Application) startDocumentsExporter() error {
	querier, err := a.builder.Outbox()
	if err != nil {
		return err
	}

	a.documentsExporter = newDocumentsExporter(
		slog.Default(),
		[]outboxReader{newDocumentsReader(querier, cargodocuments.NewClient())},
	)
	a.documentsExporter.Start()

	a.builder.StartBackground(func(ctx context.Context) {
		a.documentsExporter.Run(ctx)
	})

	return nil
}

func (a *Application) startPaymentsExporter() error {
	querier, err := a.builder.Outbox()
	if err != nil {
		return err
	}

	a.paymentsExporter = newPaymentsExporter(
		slog.Default(),
		[]outboxReader{newPaymentsReader(querier, paymentsclient.NewClient())},
	)
	a.paymentsExporter.Start()

	a.builder.StartBackground(func(ctx context.Context) {
		a.paymentsExporter.Run(ctx)
	})

	return nil
}

func (a *Application) startReserveExpirer() {
	if a.reserveExpirer == nil {
		return
	}

	expirer := newReserveExpirer(slog.Default(), a.reserveExpirer)
	a.builder.StartBackground(expirer.Run)
}

func (a *Application) startGRPC(ctx context.Context) error {
	go func() {
		if err := a.grpc.ListenAndServe(ctx); err != nil {
			slog.Error("application: grpc server", "error", err)
		}
	}()

	return nil
}

// Shutdown stops the background workers and the servers with a grace
// period, then closes the builder resources (db pool, mock server).
func (a *Application) Shutdown() error {
	graceCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()

	if a.exporter != nil {
		if err := a.exporter.Stop(graceCtx); err != nil {
			slog.Error("application: stop wms exporter", "error", err)
		}
	}

	if a.documentsExporter != nil {
		if err := a.documentsExporter.Stop(graceCtx); err != nil {
			slog.Error("application: stop documents exporter", "error", err)
		}
	}

	if a.paymentsExporter != nil {
		if err := a.paymentsExporter.Stop(graceCtx); err != nil {
			slog.Error("application: stop payments exporter", "error", err)
		}
	}

	if err := a.grpc.Shutdown(graceCtx); err != nil {
		slog.Error("application: stop grpc server", "error", err)
	}

	if err := a.obs.Shutdown(graceCtx); err != nil {
		slog.Error("application: stop observability server", "error", err)
	}

	a.builder.Shutdown()

	return nil
}
