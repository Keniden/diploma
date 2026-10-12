package tsd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	tsdapp "last-mile-task/internal/application/tsd"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
	"last-mile-task/pkg/qr"
)

type testHandshake struct {
	session qr.Session
	err     error
}

func (t *testHandshake) DecodeQR(_ string) (qr.Session, error) {
	return t.session, t.err
}

type testReserveApprover struct {
	err error
}

func (t *testReserveApprover) ApproveReserveByRoutePointID(context.Context, values.ShipmentID, values.RoutePointID) error {
	return t.err
}

type testRouteRepo struct {
	err    error
	status values.RoutePointStatus
}

func (t *testRouteRepo) GetRouteByShipmentID(context.Context, values.ShipmentID) (*entities.Route, error) {
	if t.err != nil {
		return nil, t.err
	}
	return &entities.Route{Points: []*entities.RoutePoint{{ID: testTransportPointID(), Status: t.status}}}, nil
}

func (t *testRouteRepo) SaveRoute(context.Context, *entities.Route) error { return nil }

type testScanner struct {
	err error
}

func (t *testScanner) ScanPackages(context.Context, values.ShipmentID, []scan.Package) error {
	return t.err
}

type testLister struct {
	err error
}

func (t *testLister) ListPackagesToScan(context.Context, values.ShipmentID, values.RoutePointID) ([]tsdapp.PackageToScan, error) {
	return nil, t.err
}

func testTransportShipmentID() string {
	return "11111111-1111-1111-1111-111111111111"
}

func testTransportPointID() values.RoutePointID {
	return values.RoutePointID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
}

func testTransportPackageID() string {
	return "33333333-3333-3333-3333-333333333333"
}

func TestServerScanShipmentSuccess(t *testing.T) {
	app := tsdapp.New(tsdapp.Config{
		Handshake:       &testHandshake{session: qr.Session{ShipmentID: testTransportShipmentID(), RoutePointID: testTransportPointID().String()}},
		ReserveApprover: &testReserveApprover{},
		RouteRepository: &testRouteRepo{status: values.RoutePointStatusWaitingArrival},
	})
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("tsd_qr_token", "token"))
	resp, err := srv.ScanShipment(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ScanShipment: %v", err)
	}
	if resp == nil {
		t.Fatal("expected a non-nil empty response")
	}
}

func TestServerScanShipmentMissingToken(t *testing.T) {
	app := tsdapp.New(tsdapp.Config{ReserveApprover: &testReserveApprover{}})
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	_, err := srv.ScanShipment(context.Background(), &emptypb.Empty{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for a missing token, got %v", err)
	}
}

func TestServerScanShipmentMalformedQR(t *testing.T) {
	app := tsdapp.New(tsdapp.Config{
		Handshake:       &testHandshake{err: qr.ErrMalformedSession},
		ReserveApprover: &testReserveApprover{},
		RouteRepository: &testRouteRepo{},
	})
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("tsd_qr_token", "bad"))
	_, err := srv.ScanShipment(ctx, &emptypb.Empty{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for a malformed QR, got %v", err)
	}
}

func TestServerScanPackagesPartialSuccess(t *testing.T) {
	app := tsdapp.New(tsdapp.Config{Scanner: &testScanner{err: errors.New("invalid scan")}})
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"tsd_shipment_id", testTransportShipmentID(),
		"tsd_packages", `[{"package_id":"`+testTransportPackageID()+`","route_point_id":"22222222-2222-2222-2222-222222222222","point_id":1,"action":1}]`,
	))
	resp, err := srv.ScanPackages(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ScanPackages: expected partial success with nil error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected a non-nil empty response")
	}
}

func TestServerScanPackagesMissingShipment(t *testing.T) {
	app := tsdapp.New(tsdapp.Config{Scanner: &testScanner{}})
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	_, err := srv.ScanPackages(ctx, &emptypb.Empty{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing shipment id, got %v", err)
	}
}

func TestServerListPackagesToScanSuccess(t *testing.T) {
	app := tsdapp.New(tsdapp.Config{PackagesToScan: &testLister{}})
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"tsd_shipment_id", testTransportShipmentID(),
		"tsd_route_point_id", testTransportPointID().String(),
	))
	resp, err := srv.ListPackagesToScan(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListPackagesToScan: %v", err)
	}
	if resp == nil {
		t.Fatal("expected a non-nil empty response")
	}
}
