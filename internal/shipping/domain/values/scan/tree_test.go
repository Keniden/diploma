package scan_test

import (
	"testing"
	"time"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

func TestNewRoutePointTree(t *testing.T) {
	rp1 := values.NewID()
	rp2 := values.NewID()
	pkg1 := values.NewID()
	pkg2 := values.NewID()
	pkg3 := values.NewID()

	pkgs := []scan.Package{
		{
			ID:           pkg1,
			RoutePointID: rp1,
			ActionType:   values.RoutePointActionTypeLoad,
			Timestamp:    time.Now(),
		},
		{
			ID:           pkg2,
			RoutePointID: rp1,
			ActionType:   values.RoutePointActionTypeLoad,
			Timestamp:    time.Now(),
		},
		{
			ID:           pkg3,
			RoutePointID: rp2,
			ActionType:   values.RoutePointActionTypeUnload,
			Timestamp:    time.Now(),
		},
	}

	tree := scan.NewRoutePointTree(pkgs)

	if len(tree) != 2 {
		t.Fatalf("expected 2 route points, got %d", len(tree))
	}

	// rp1 has Load action with 2 packages
	loadPackages := tree[rp1][values.RoutePointActionTypeLoad]
	if len(loadPackages) != 2 {
		t.Fatalf("expected 2 load packages for rp1, got %d", len(loadPackages))
	}
	if _, ok := loadPackages[pkg1]; !ok {
		t.Fatal("expected pkg1 in rp1 load packages")
	}
	if _, ok := loadPackages[pkg2]; !ok {
		t.Fatal("expected pkg2 in rp1 load packages")
	}

	// rp2 has Unload action with 1 package
	unloadPackages := tree[rp2][values.RoutePointActionTypeUnload]
	if len(unloadPackages) != 1 {
		t.Fatalf("expected 1 unload package for rp2, got %d", len(unloadPackages))
	}
	if _, ok := unloadPackages[pkg3]; !ok {
		t.Fatal("expected pkg3 in rp2 unload packages")
	}
}

func TestNewRoutePointTree_Empty(t *testing.T) {
	tree := scan.NewRoutePointTree(nil)
	if len(tree) != 0 {
		t.Fatalf("expected empty tree, got %d entries", len(tree))
	}
}

func TestNewShipmentTree(t *testing.T) {
	s1 := values.NewID()
	s2 := values.NewID()

	pkgs := []scan.Package{
		{
			ID:         values.NewID(),
			ShipmentID: s1,
			Timestamp:  time.Now(),
		},
		{
			ID:         values.NewID(),
			ShipmentID: s1,
			Timestamp:  time.Now(),
		},
		{
			ID:         values.NewID(),
			ShipmentID: s2,
			Timestamp:  time.Now(),
		},
	}

	tree := scan.NewShipmentTree(pkgs)

	if len(tree) != 2 {
		t.Fatalf("expected 2 shipments, got %d", len(tree))
	}

	if len(tree[s1]) != 2 {
		t.Fatalf("expected 2 packages for s1, got %d", len(tree[s1]))
	}

	if len(tree[s2]) != 1 {
		t.Fatalf("expected 1 package for s2, got %d", len(tree[s2]))
	}
}

func TestNewShipmentTree_Empty(t *testing.T) {
	tree := scan.NewShipmentTree(nil)
	if len(tree) != 0 {
		t.Fatalf("expected empty tree, got %d entries", len(tree))
	}
}

func TestPackageIDs(t *testing.T) {
	id1 := values.NewID()
	id2 := values.NewID()
	id3 := values.NewID()

	pkgs := []scan.Package{
		{ID: id1, Timestamp: time.Now()},
		{ID: id2, Timestamp: time.Now()},
		{ID: id3, Timestamp: time.Now()},
	}

	ids := scan.PackageIDs(pkgs)

	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	if ids[0] != id1 || ids[1] != id2 || ids[2] != id3 {
		t.Fatal("ids do not match expected order")
	}
}

func TestPackageIDs_Empty(t *testing.T) {
	ids := scan.PackageIDs(nil)
	if len(ids) != 0 {
		t.Fatalf("expected 0 ids, got %d", len(ids))
	}
}
