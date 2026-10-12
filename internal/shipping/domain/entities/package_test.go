package entities_test

import (
	"testing"
	"time"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

func TestNewPackage(t *testing.T) {
	p := entities.NewPackage()

	if p.ID == values.NilID {
		t.Fatal("expected non-nil ID")
	}
	if p.Status != values.PackageStatusUnspecified {
		t.Fatalf("expected unspecified status, got %v", p.Status)
	}
	if p.IsReturn {
		t.Fatal("expected IsReturn=false")
	}
	if len(p.Actions) != 0 {
		t.Fatalf("expected 0 actions, got %d", len(p.Actions))
	}
}

func TestPackage_AssignToShipment(t *testing.T) {
	p := entities.NewPackage()
	id := values.NewID()

	p.AssignToShipment(id)

	if p.ShipmentID != id {
		t.Fatalf("expected shipment ID %v, got %v", id, p.ShipmentID)
	}
}

func TestPackage_Loaded(t *testing.T) {
	p := entities.NewPackage()
	p.ToReadyToLoad()

	if err := p.Loaded(); err != nil {
		t.Fatal(err)
	}

	if p.Status != values.PackageStatusLoaded {
		t.Fatalf("expected loaded status, got %v", p.Status)
	}
}

func TestPackage_Unloaded(t *testing.T) {
	p := entities.NewPackage()
	p.ToReadyToLoad()
	_ = p.Loaded()
	_ = p.MarkInTransit()
	_ = p.MarkUnloading()

	if err := p.Unloaded(); err != nil {
		t.Fatal(err)
	}

	if p.Status != values.PackageStatusUnloaded {
		t.Fatalf("expected unloaded status, got %v", p.Status)
	}
}

func TestPackage_ToReadyToLoad(t *testing.T) {
	p := entities.NewPackage()
	p.ToReadyToLoad()

	if p.Status != values.PackageStatusReadyToLoad {
		t.Fatalf("expected ready to load status, got %v", p.Status)
	}
}

func TestPackage_MarkInTransit(t *testing.T) {
	p := entities.NewPackage()
	p.ToReadyToLoad()
	_ = p.Loaded()

	if err := p.MarkInTransit(); err != nil {
		t.Fatal(err)
	}

	if p.Status != values.PackageStatusInTransit {
		t.Fatalf("expected in transit status, got %v", p.Status)
	}
}

func TestPackage_MarkUnloading(t *testing.T) {
	p := entities.NewPackage()
	p.ToReadyToLoad()
	_ = p.Loaded()
	_ = p.MarkInTransit()

	if err := p.MarkUnloading(); err != nil {
		t.Fatal(err)
	}

	if p.Status != values.PackageStatusUnloading {
		t.Fatalf("expected unloading status, got %v", p.Status)
	}
}

func TestPackage_ToDirect(t *testing.T) {
	p := entities.NewPackage()
	p.IsReturn = true
	p.ToDirect()

	if p.IsReturn {
		t.Fatal("expected IsReturn=false after ToDirect")
	}
}

func TestPackage_ToReturn(t *testing.T) {
	p := entities.NewPackage()
	p.ToReturn()

	if !p.IsReturn {
		t.Fatal("expected IsReturn=true after ToReturn")
	}
}

func TestPackage_IsLoaded(t *testing.T) {
	p := entities.NewPackage()
	if p.IsLoaded() {
		t.Fatal("expected not loaded for new package")
	}

	p.ToReadyToLoad()
	_ = p.Loaded()
	if !p.IsLoaded() {
		t.Fatal("expected loaded after Loaded()")
	}
}

func TestPackageIDs(t *testing.T) {
	p1 := entities.NewPackage()
	p2 := entities.NewPackage()
	p3 := entities.NewPackage()

	ids := entities.PackageIDs([]*entities.Package{p1, p2, p3})

	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
	if ids[0] != p1.ID || ids[1] != p2.ID || ids[2] != p3.ID {
		t.Fatal("IDs do not match expected order")
	}
}

func TestPackageIDs_Empty(t *testing.T) {
	ids := entities.PackageIDs(nil)
	if len(ids) != 0 {
		t.Fatalf("expected 0 IDs, got %d", len(ids))
	}
}

func TestNewPackageFromDemand_Forward(t *testing.T) {
	demand := values.PackageDemand{
		PackageID:     values.NewID(),
		TareID:        123,
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	}

	p := entities.NewPackageFromDemand(demand)

	if p.ID != demand.PackageID {
		t.Fatalf("expected package ID %v, got %v", demand.PackageID, p.ID)
	}
	if p.TareID != demand.TareID {
		t.Fatalf("expected tare ID %d, got %d", demand.TareID, p.TareID)
	}
	if p.Status != values.PackageStatusReadyToLoad {
		t.Fatalf("expected ready to load status, got %v", p.Status)
	}
	if p.IsReturn {
		t.Fatal("expected IsReturn=false for forward direction")
	}
	if len(p.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(p.Actions))
	}

	loadAction := p.Actions[0]
	if loadAction.Type != values.RoutePointActionTypeLoad {
		t.Fatal("first action should be Load")
	}
	if loadAction.RoutePointID != demand.LoadPointID {
		t.Fatal("load action has wrong route point ID")
	}
	if loadAction.Status != values.RoutePointPackageStatusTODO {
		t.Fatal("load action should be TODO")
	}

	unloadAction := p.Actions[1]
	if unloadAction.Type != values.RoutePointActionTypeUnload {
		t.Fatal("second action should be Unload")
	}
	if unloadAction.RoutePointID != demand.UnloadPointID {
		t.Fatal("unload action has wrong route point ID")
	}
}

func TestNewPackageFromDemand_Backward(t *testing.T) {
	demand := values.PackageDemand{
		PackageID:     values.NewID(),
		TareID:        456,
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionBackward,
	}

	p := entities.NewPackageFromDemand(demand)

	if !p.IsReturn {
		t.Fatal("expected IsReturn=true for backward direction")
	}
}

func TestNewPackagesFromDemands(t *testing.T) {
	demands := []values.PackageDemand{
		{
			PackageID:     values.NewID(),
			TareID:        1,
			LoadPointID:   values.NewID(),
			UnloadPointID: values.NewID(),
			Direction:     values.PackageDemandDirectionForward,
		},
		{
			PackageID:     values.NewID(),
			TareID:        2,
			LoadPointID:   values.NewID(),
			UnloadPointID: values.NewID(),
			Direction:     values.PackageDemandDirectionBackward,
		},
	}

	pkgs := entities.NewPackagesFromDemands(demands)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].ID != demands[0].PackageID {
		t.Fatal("first package ID mismatch")
	}
	if pkgs[1].ID != demands[1].PackageID {
		t.Fatal("second package ID mismatch")
	}
	if pkgs[0].IsReturn {
		t.Fatal("first package should be forward (not return)")
	}
	if !pkgs[1].IsReturn {
		t.Fatal("second package should be backward (return)")
	}
}

func TestPackage_ApplyScan_MismatchedID(t *testing.T) {
	p := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})

	applied, err := p.ApplyScan(scan.Package{
		ID:           values.NewID(), // different ID
		ActionType:   values.RoutePointActionTypeLoad,
		RoutePointID: p.Actions[0].RoutePointID,
		Timestamp:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("expected not applied for mismatched package ID")
	}
}

func TestPackage_ApplyScan_MismatchedShipmentID(t *testing.T) {
	p := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	p.AssignToShipment(values.NewID())

	applied, err := p.ApplyScan(scan.Package{
		ID:           p.ID,
		ShipmentID:   values.NewID(),
		ActionType:   values.RoutePointActionTypeLoad,
		RoutePointID: p.Actions[0].RoutePointID,
		Timestamp:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("expected not applied for mismatched shipment id")
	}
}

func TestPackage_ApplyScan_MismatchedActionType(t *testing.T) {
	p := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})

	// scan says unload, but action is load
	applied, err := p.ApplyScan(scan.Package{
		ID:           p.ID,
		ActionType:   values.RoutePointActionTypeUnload,
		RoutePointID: p.Actions[0].RoutePointID,
		Timestamp:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("expected not applied for mismatched action type")
	}
}

func TestPackage_ApplyScan_MismatchedRoutePointID(t *testing.T) {
	p := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})

	applied, err := p.ApplyScan(scan.Package{
		ID:           p.ID,
		ActionType:   values.RoutePointActionTypeLoad,
		RoutePointID: values.NewID(), // different route point
		Timestamp:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("expected not applied for mismatched route point ID")
	}
}

func TestPackage_ApplyScan_Unload(t *testing.T) {
	p := entities.NewPackageFromDemand(values.PackageDemand{
		PackageID:     values.NewID(),
		LoadPointID:   values.NewID(),
		UnloadPointID: values.NewID(),
		Direction:     values.PackageDemandDirectionForward,
	})
	// Package starts at ReadyToLoad. Transition through Loaded → Unloading first.
	_ = p.Loaded()
	_ = p.MarkInTransit()
	_ = p.MarkUnloading()

	ts := time.Now()

	applied, err := p.ApplyScan(scan.Package{
		ID:           p.ID,
		ActionType:   values.RoutePointActionTypeUnload,
		RoutePointID: p.Actions[1].RoutePointID,
		Timestamp:    ts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("expected applied=true")
	}
	if p.Status != values.PackageStatusUnloaded {
		t.Fatalf("expected unloaded status, got %v", p.Status)
	}
	if p.Actions[1].Status != values.RoutePointPackageStatusDone {
		t.Fatalf("expected action status DONE, got %v", p.Actions[1].Status)
	}
}

func TestPackage_ApplyScan_NoActions(t *testing.T) {
	p := entities.NewPackage()
	applied, err := p.ApplyScan(scan.Package{
		ID:           p.ID,
		ActionType:   values.RoutePointActionTypeLoad,
		RoutePointID: values.NewID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("expected not applied for package with no actions")
	}
}
