package entities_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

func TestPackage_ApplyScan_Load(t *testing.T) {
	routePointID := values.NewID()
	pkg := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   routePointID,
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	pkg.AssignToShipment(values.NewID())
	pkg.TareID = 1

	applied, err := pkg.ApplyScan(scan.Package{
		ID:           pkg.ID,
		RoutePointID: routePointID,
		ShipmentID:   pkg.ShipmentID,
		ActionType:   values.RoutePointActionTypeLoad,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatalf("expected applied=true")
	}
	if pkg.Status != values.PackageStatusLoaded {
		t.Fatalf("expected status loaded, got %v", pkg.Status)
	}
}
