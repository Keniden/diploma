package tsd

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/pkg/qr"
)

func TestScanShipmentSuccess(t *testing.T) {
	shipmentID := testShipmentID()
	routePointID := testRoutePointID()

	handshake := &fakeHandshake{session: qr.Session{
		ShipmentID:   shipmentID.String(),
		RoutePointID: routePointID.String(),
	}}
	approver := &fakeReserveApprover{}
	routeRepo := &fakeRouteRepository{route: &entities.Route{
		Points: []*entities.RoutePoint{{ID: routePointID, Status: values.RoutePointStatusWaitingArrival}},
	}}

	app := New(Config{
		Handshake:       handshake,
		ReserveApprover: approver,
		RouteRepository: routeRepo,
	})

	err := app.ScanShipment(context.Background(), "token")
	if err != nil {
		t.Fatalf("ScanShipment: %v", err)
	}

	if approver.approvedShipment != shipmentID || approver.approvedRoutePoint != routePointID {
		t.Fatalf("expected approve(%s, %s), got (%s, %s)",
			shipmentID, routePointID, approver.approvedShipment, approver.approvedRoutePoint)
	}
	if routeRepo.saved == nil {
		t.Fatal("expected the route to be saved")
	}
	if routeRepo.saved.Points[0].Status != values.RoutePointStatusDriverArrived {
		t.Fatalf("expected route point DRIVER_ARRIVED, got %d", routeRepo.saved.Points[0].Status)
	}
}

func TestScanShipmentInvalidQR(t *testing.T) {
	approver := &fakeReserveApprover{}
	app := New(Config{
		Handshake:       &fakeHandshake{err: qr.ErrMalformedSession},
		ReserveApprover: approver,
		RouteRepository: &fakeRouteRepository{},
	})

	err := app.ScanShipment(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected an error on a malformed QR token")
	}
	if !errors.Is(err, qr.ErrMalformedSession) {
		t.Fatalf("expected ErrMalformedSession, got %v", err)
	}
	if approver.approvedShipment != values.NilID {
		t.Fatal("expected no reserve approval on a malformed token")
	}
}

func TestScanShipmentInvalidSessionID(t *testing.T) {
	app := New(Config{
		Handshake:       &fakeHandshake{session: qr.Session{ShipmentID: "not-a-uuid", RoutePointID: "also-na"}},
		ReserveApprover: &fakeReserveApprover{},
		RouteRepository: &fakeRouteRepository{},
	})

	if err := app.ScanShipment(context.Background(), "token"); err == nil {
		t.Fatal("expected an error on non-UUID session ids")
	}
}

func TestScanShipmentReserveNotApproved(t *testing.T) {
	shipmentID := testShipmentID()
	routePointID := testRoutePointID()
	routeRepo := &fakeRouteRepository{route: &entities.Route{
		Points: []*entities.RoutePoint{{ID: routePointID, Status: values.RoutePointStatusWaitingArrival}},
	}}

	app := New(Config{
		Handshake:       &fakeHandshake{session: qr.Session{ShipmentID: shipmentID.String(), RoutePointID: routePointID.String()}},
		ReserveApprover: &fakeReserveApprover{err: errors.New("reserve not found")},
		RouteRepository: routeRepo,
	})

	if err := app.ScanShipment(context.Background(), "token"); err == nil {
		t.Fatal("expected an error when the reserve cannot be approved")
	}
	if routeRepo.saved != nil {
		t.Fatal("expected no route save when approval fails")
	}
}

func TestScanShipmentRoutePointNotFound(t *testing.T) {
	shipmentID := testShipmentID()
	routePointID := testRoutePointID()
	routeRepo := &fakeRouteRepository{route: &entities.Route{
		Points: []*entities.RoutePoint{{ID: testPackageID()}},
	}}

	app := New(Config{
		Handshake:       &fakeHandshake{session: qr.Session{ShipmentID: shipmentID.String(), RoutePointID: routePointID.String()}},
		ReserveApprover: &fakeReserveApprover{},
		RouteRepository: routeRepo,
	})

	if err := app.ScanShipment(context.Background(), "token"); err == nil {
		t.Fatal("expected an error when the route point is not found")
	}
	if routeRepo.saved != nil {
		t.Fatal("expected no route save when the point is missing")
	}
}

func TestScanShipmentRoutePointTransitionProhibited(t *testing.T) {
	shipmentID := testShipmentID()
	routePointID := testRoutePointID()
	routeRepo := &fakeRouteRepository{route: &entities.Route{
		Points: []*entities.RoutePoint{{ID: routePointID, Status: values.RoutePointStatusCompleted}},
	}}

	app := New(Config{
		Handshake:       &fakeHandshake{session: qr.Session{ShipmentID: shipmentID.String(), RoutePointID: routePointID.String()}},
		ReserveApprover: &fakeReserveApprover{},
		RouteRepository: routeRepo,
	})

	if err := app.ScanShipment(context.Background(), "token"); err == nil {
		t.Fatal("expected an error on a prohibited route point transition")
	}
	if routeRepo.saved != nil {
		t.Fatal("expected no route save on a prohibited transition")
	}
}
