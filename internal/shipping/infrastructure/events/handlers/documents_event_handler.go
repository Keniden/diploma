package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
	docentities "last-mile-task/internal/documents/domain/entities"
	docdirect "last-mile-task/internal/documents/domain/directions"
	docservices "last-mile-task/internal/documents/domain/services"
	docvalues "last-mile-task/internal/documents/domain/values"
	"last-mile-task/internal/shipping/domain/events"
	shippingvalues "last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
)

// documentsStore serves the reads the documents handler needs: shipment,
// route points and package directions (load point → unload point).
type documentsStore interface {
	ShipmentByID(ctx context.Context, shipmentID shippingvalues.ShipmentID) (dto.Shipment, error)
	RoutePointByID(ctx context.Context, routePointID shippingvalues.RoutePointID) (dto.RoutePoint, error)
	PackagesForDirection(ctx context.Context, shipmentID uuid.UUID, fromRoutePointID uuid.UUID) ([]docdirect.Package, error)
	ShipmentPackages(ctx context.Context, shipmentID uuid.UUID) ([]docdirect.Package, error)
	HasLoadedReturnPackagesAtRoutePoint(ctx context.Context, routePointID uuid.UUID) (bool, error)
}

// DocumentRepository is the write surface the documents handler persists.
// documentspg.DocumentRepository implements it.
type DocumentRepository interface {
	Save(ctx context.Context, doc *docentities.Document) error
}

// DocumentWriter is the outbox write surface. documents_outbox.Writer
// implements it.
type DocumentWriter interface {
	WritePublishAfterStartShipment(ctx context.Context, event cargodocuments.Event) error
}

// documentsEventHandler bridges shipment lifecycle events into the documents
// vertical: it assembles a waybill Document (forward on start, returned on
// intermediate-point departure), persists it and publishes the
// documents.publish_after_start_shipment outbox message inside the same
// transaction as the event.
type documentsEventHandler struct {
	store   documentsStore
	offices warehouse.Offices
	users   warehouse.Users
	builder *docservices.DocumentBuilderService
	repo    DocumentRepository
	writer  DocumentWriter
}

// NewDocumentsEventHandler builds the documents event handler.
func NewDocumentsEventHandler(
	store documentsStore,
	offices warehouse.Offices,
	users warehouse.Users,
	builder *docservices.DocumentBuilderService,
	repo DocumentRepository,
	writer DocumentWriter,
) *documentsEventHandler {
	return &documentsEventHandler{
		store:   store,
		offices: offices,
		users:   users,
		builder: builder,
		repo:    repo,
		writer:  writer,
	}
}

// HandleShipmentStartedEvent publishes a forward waybill for the started
// shipment.
func (h *documentsEventHandler) HandleShipmentStartedEvent(
	ctx context.Context,
	event events.ShipmentStartedEvent,
) error {
	packages, err := h.store.ShipmentPackages(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("documents: shipment packages: %w", err)
	}

	return h.publishDirections(ctx, event.ID, docvalues.DocumentTypeWaybillForward, packages)
}

// HandlePointCompletedEvent publishes a return waybill when the driver departs
// an intermediate point that has loaded return packages. Otherwise it is a
// no-op (ensureReturnDocsScheduled).
func (h *documentsEventHandler) HandlePointCompletedEvent(
	ctx context.Context,
	event events.PointCompletedEvent,
) error {
	hasReturn, err := h.store.HasLoadedReturnPackagesAtRoutePoint(ctx, event.RoutePointID)
	if err != nil {
		return fmt.Errorf("documents: has loaded return packages: %w", err)
	}
	if !hasReturn {
		return nil
	}

	packages, err := h.store.PackagesForDirection(ctx, event.ShipmentID, event.RoutePointID)
	if err != nil {
		return fmt.Errorf("documents: return packages: %w", err)
	}

	return h.publishDirections(ctx, event.ShipmentID, docvalues.DocumentTypeWaybillReturn, packages)
}

// publishDirections groups the packages by direction, and for each direction
// builds, persists and publishes one waybill document.
func (h *documentsEventHandler) publishDirections(
	ctx context.Context,
	shipmentID uuid.UUID,
	documentType docvalues.DocumentType,
	packages []docdirect.Package,
) error {
	groups := docdirect.GroupByDirection(packages)
	if len(groups) == 0 {
		return nil
	}

	shipment, err := h.store.ShipmentByID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("documents: shipment: %w", err)
	}

	user, err := h.users.UserByID(ctx, shipment.DriverID)
	if err != nil {
		return fmt.Errorf("documents: driver: %w", err)
	}

	driverName := dto.Username(user.FirstName, user.MiddleName, user.LastName)

	for key, group := range groups {
		if len(group) == 0 {
			continue
		}

		fromPoint, err := h.store.RoutePointByID(ctx, key.From)
		if err != nil {
			return fmt.Errorf("documents: from route point: %w", err)
		}

		office, err := h.offices.GetOfficeByPointID(ctx, int64(fromPoint.PointID))
		if err != nil {
			return fmt.Errorf("documents: office: %w", err)
		}

		built, err := h.builder.Build(docvalues.BuildInput{
			ShipmentID:   docvalues.ShipmentID(shipmentID),
			DocumentType: documentType,
			Packages:     group,
			Driver: docvalues.Driver{
				ID:    shipment.DriverID.String(),
				Name:  driverName,
				Phone: user.Phone,
			},
			OfficePoint: docvalues.OfficePoint{
				PointID:   int64(office.ID),
				CountryID: office.CountryID,
			},
			EventUUID: uuid.NewString(),
		})
		if err != nil {
			return fmt.Errorf("documents: build: %w", err)
		}

		doc, err := docentities.NewDocument(docvalues.DocumentSpec{
			ShipmentID:       built.ShipmentID,
			FromRoutePointID: built.FromRoutePointID,
			ToRoutePointID:   built.ToRoutePointID,
			DocumentType:     built.DocumentType,
			EventUUID:        built.EventUUID,
			ExternalID:       built.ExternalID,
			DataPrintTaskID:  built.DataPrintTaskID,
			GenerateTaskID:   built.GenerateTaskID,
			PackageIDs:       built.PackageIDs,
		})
		if err != nil {
			return fmt.Errorf("documents: new document: %w", err)
		}

		if err := h.repo.Save(ctx, doc); err != nil {
			return fmt.Errorf("documents: save document: %w", err)
		}

		if err := h.writer.WritePublishAfterStartShipment(ctx, h.toEvent(built)); err != nil {
			return fmt.Errorf("documents: write outbox: %w", err)
		}
	}

	return nil
}

func (h *documentsEventHandler) toEvent(built docvalues.WithShortGoods) cargodocuments.Event {
	var from, to *uuid.UUID
	if built.FromRoutePointID != docvalues.NilID {
		v := uuid.UUID(built.FromRoutePointID)
		from = &v
	}
	if built.ToRoutePointID != docvalues.NilID {
		v := uuid.UUID(built.ToRoutePointID)
		to = &v
	}

	packageIDs := make([]uuid.UUID, 0, len(built.PackageIDs))
	for _, id := range built.PackageIDs {
		packageIDs = append(packageIDs, uuid.UUID(id))
	}

	items := make([]cargodocuments.Item, 0, len(built.Items))
	for _, it := range built.Items {
		items = append(items, cargodocuments.Item{
			Article: it.Good.Article,
			Qty:     it.Quantity.Value,
			Unit:    it.Quantity.Unit,
		})
	}

	return cargodocuments.Event{
		EventUUID:      built.EventUUID,
		Type:           "waybill",
		ShipmentID:     built.ShipmentID,
		FromRoutePointID: from,
		ToRoutePointID: to,
		DocumentType:   int16(built.DocumentType),
		DocumentID:     built.ExternalID,
		DataPrintTaskID: built.DataPrintTaskID,
		GenerateTaskID: built.GenerateTaskID,
		PackageIDs:     packageIDs,
		Items:          items,
		DriverName:     built.Meta.Driver.Name,
		DriverPhone:    built.Meta.Driver.Phone,
		RouteCarType:   built.Meta.RouteCarType,
		OfficePointID:  built.OfficePoint.PointID,
	}
}