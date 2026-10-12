package tsd

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestListPackagesToScanEmpty(t *testing.T) {
	lister := &fakePackagesToScan{}
	app := New(Config{PackagesToScan: lister})

	packages, err := app.ListPackagesToScan(context.Background(), testShipmentID(), testRoutePointID())
	if err != nil {
		t.Fatalf("ListPackagesToScan: %v", err)
	}
	if len(packages) != 0 {
		t.Fatalf("expected an empty list, got %d", len(packages))
	}
}

func TestListPackagesToScanReturnsPackages(t *testing.T) {
	expected := []PackageToScan{
		{ID: testPackageID(), ShipmentID: testShipmentID(), Status: values.PackageStatusReadyToLoad, TareID: 1, TareType: "BOX"},
	}
	lister := &fakePackagesToScan{packages: expected}
	app := New(Config{PackagesToScan: lister})

	packages, err := app.ListPackagesToScan(context.Background(), testShipmentID(), testRoutePointID())
	if err != nil {
		t.Fatalf("ListPackagesToScan: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}
	if packages[0].ID != expected[0].ID {
		t.Fatalf("expected package %s, got %s", expected[0].ID, packages[0].ID)
	}
	if lister.askedShipment != testShipmentID() || lister.askedRoutePoint != testRoutePointID() {
		t.Fatalf("expected query for (%s, %s), got (%s, %s)",
			testShipmentID(), testRoutePointID(), lister.askedShipment, lister.askedRoutePoint)
	}
}

func TestListPackagesToScanPropagatesError(t *testing.T) {
	lister := &fakePackagesToScan{err: errors.New("store failure")}
	app := New(Config{PackagesToScan: lister})

	if _, err := app.ListPackagesToScan(context.Background(), testShipmentID(), testRoutePointID()); err == nil {
		t.Fatal("expected an error to propagate")
	}
}
