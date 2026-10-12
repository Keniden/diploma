package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	cargodocuments "last-mile-task/internal/clients/cargo-documents"
	docentities "last-mile-task/internal/documents/domain/entities"
	docdirect "last-mile-task/internal/documents/domain/directions"
	docservices "last-mile-task/internal/documents/domain/services"
	docvalues "last-mile-task/internal/documents/domain/values"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
)

type docFakeStore struct {
	shipment      dto.Shipment
	shipmentErr   error
	routePoint    dto.RoutePoint
	routePointErr error
	packages      []docdirect.Package
	packagesErr   error
	hasReturn     bool
	hasReturnErr  error
}

func (f *docFakeStore) ShipmentByID(context.Context, values.ShipmentID) (dto.Shipment, error) {
	return f.shipment, f.shipmentErr
}

func (f *docFakeStore) RoutePointByID(context.Context, values.RoutePointID) (dto.RoutePoint, error) {
	return f.routePoint, f.routePointErr
}

func (f *docFakeStore) PackagesForDirection(context.Context, uuid.UUID, uuid.UUID) ([]docdirect.Package, error) {
	return f.packages, f.packagesErr
}

func (f *docFakeStore) ShipmentPackages(context.Context, uuid.UUID) ([]docdirect.Package, error) {
	return f.packages, f.packagesErr
}

func (f *docFakeStore) HasLoadedReturnPackagesAtRoutePoint(context.Context, uuid.UUID) (bool, error) {
	return f.hasReturn, f.hasReturnErr
}

type docFakeOffices struct {
	office warehouse.Office
	err    error
}

func (f *docFakeOffices) GetOfficeByPointID(context.Context, int64) (warehouse.Office, error) {
	return f.office, f.err
}

type docFakeUsers struct {
	user *warehouse.User
	err  error
}

func (f *docFakeUsers) UserByID(context.Context, values.DriverID) (*warehouse.User, error) {
	return f.user, f.err
}

type docFakeRepo struct {
	err   error
	saved []*docentities.Document
}

func (f *docFakeRepo) Save(_ context.Context, doc *docentities.Document) error {
	f.saved = append(f.saved, doc)
	return f.err
}

type docFakeWriter struct {
	err    error
	events []cargodocuments.Event
}

func (f *docFakeWriter) WritePublishAfterStartShipment(_ context.Context, event cargodocuments.Event) error {
	f.events = append(f.events, event)
	return f.err
}

func newDocumentsHandler(
	store documentsStore,
	offices warehouse.Offices,
	users warehouse.Users,
	repo DocumentRepository,
	writer DocumentWriter,
) *documentsEventHandler {
	return NewDocumentsEventHandler(
		store, offices, users, docservices.NewDocumentBuilderService(), repo, writer,
	)
}

func docDefaultStore(from, to uuid.UUID) *docFakeStore {
	return &docFakeStore{
		shipment:   dto.Shipment{ID: uuid.New(), DriverID: uuid.New(), CarLicensePlate: "E777AA77"},
		routePoint: dto.RoutePoint{ID: from, PointID: 1001},
		hasReturn:  true,
		packages: []docdirect.Package{
			{ID: uuid.New(), TareID: 1, TareType: "BOX", TareSticker: "BOX1", Action: docdirect.PackageActionLoad, From: from, To: to},
		},
	}
}

func docOffices() *docFakeOffices {
	return &docFakeOffices{office: warehouse.Office{ID: 7, CountryID: 643}}
}

func docUsers() *docFakeUsers {
	return &docFakeUsers{user: &warehouse.User{FirstName: "Ivan", LastName: "Petrov", Phone: "+7-900"}}
}

func TestHandleShipmentStartedEvent_BuildsAndPersistsForward(t *testing.T) {
	from, to := uuid.New(), uuid.New()
	store := docDefaultStore(from, to)
	repo := &docFakeRepo{}
	writer := &docFakeWriter{}
	h := newDocumentsHandler(store, docOffices(), docUsers(), repo, writer)

	err := h.HandleShipmentStartedEvent(context.Background(), events.ShipmentStartedEvent{ID: store.shipment.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.saved) != 1 {
		t.Fatalf("expected 1 saved document, got %d", len(repo.saved))
	}
	doc := repo.saved[0]
	if doc.DocumentType != docvalues.DocumentTypeWaybillForward {
		t.Fatalf("expected FORWARD, got %v", doc.DocumentType)
	}
	if doc.ShipmentID != store.shipment.ID {
		t.Fatal("shipment id mismatch")
	}
	if doc.FromRoutePointID != from || doc.ToRoutePointID != to {
		t.Fatal("route point ids mismatch")
	}
	if len(doc.PackageIDs) != 1 {
		t.Fatalf("expected 1 package id, got %d", len(doc.PackageIDs))
	}

	if len(writer.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(writer.events))
	}
	if writer.events[0].DocumentType != int16(docvalues.DocumentTypeWaybillForward) {
		t.Fatalf("expected FORWARD outbox event, got %d", writer.events[0].DocumentType)
	}
}

func TestHandleShipmentStartedEvent_NoPackagesSkips(t *testing.T) {
	store := &docFakeStore{shipment: dto.Shipment{ID: uuid.New(), DriverID: uuid.New()}}
	repo := &docFakeRepo{}
	writer := &docFakeWriter{}
	h := newDocumentsHandler(store, docOffices(), docUsers(), repo, writer)

	if err := h.HandleShipmentStartedEvent(context.Background(), events.ShipmentStartedEvent{ID: store.shipment.ID}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.saved) != 0 || len(writer.events) != 0 {
		t.Fatalf("expected no publish without packages")
	}
}

func TestHandlePointCompletedEvent_NoReturnPackagesSkips(t *testing.T) {
	store := &docFakeStore{hasReturn: false}
	repo := &docFakeRepo{}
	writer := &docFakeWriter{}
	h := newDocumentsHandler(store, docOffices(), docUsers(), repo, writer)

	err := h.HandlePointCompletedEvent(context.Background(), events.PointCompletedEvent{
		ShipmentID: uuid.New(), RoutePointID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.saved) != 0 || len(writer.events) != 0 {
		t.Fatalf("expected no publish without return packages")
	}
}

func TestHandlePointCompletedEvent_BuildsAndPersistsReturn(t *testing.T) {
	from, to := uuid.New(), uuid.New()
	store := docDefaultStore(from, to)
	store.hasReturn = true
	repo := &docFakeRepo{}
	writer := &docFakeWriter{}
	h := newDocumentsHandler(store, docOffices(), docUsers(), repo, writer)

	err := h.HandlePointCompletedEvent(context.Background(), events.PointCompletedEvent{
		ShipmentID: store.shipment.ID, RoutePointID: from,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.saved) != 1 {
		t.Fatalf("expected 1 saved document, got %d", len(repo.saved))
	}
	if repo.saved[0].DocumentType != docvalues.DocumentTypeWaybillReturn {
		t.Fatalf("expected RETURN, got %v", repo.saved[0].DocumentType)
	}
	if len(writer.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(writer.events))
	}
}

func TestHandleShipmentStartedEvent_PropagatesSaveError(t *testing.T) {
	from, to := uuid.New(), uuid.New()
	store := docDefaultStore(from, to)
	repo := &docFakeRepo{err: errors.New("db down")}
	writer := &docFakeWriter{}
	h := newDocumentsHandler(store, docOffices(), docUsers(), repo, writer)

	err := h.HandleShipmentStartedEvent(context.Background(), events.ShipmentStartedEvent{ID: store.shipment.ID})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(writer.events) != 0 {
		t.Fatalf("expected no outbox event after save failure")
	}
}