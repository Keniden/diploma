package tsd

import (
	"context"
	"errors"
	"testing"

	"last-mile-task/internal/shipping/domain/values/scan"
)

func TestScanPackagesSuccess(t *testing.T) {
	scanner := &fakeScanner{}
	app := New(Config{Scanner: scanner})

	pkgs := []scan.Package{
		{ID: testPackageID()},
		{ID: testRoutePointID()},
	}

	failures, err := app.ScanPackages(context.Background(), testShipmentID(), pkgs)
	if err != nil {
		t.Fatalf("ScanPackages: %v", err)
	}
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %d", len(failures))
	}
	if scanner.calls != len(pkgs) {
		t.Fatalf("expected %d scanner calls, got %d", len(pkgs), scanner.calls)
	}
}

func TestScanPackagesPartialSuccess(t *testing.T) {
	okID := testPackageID()
	badID := testRoutePointID()
	scanner := &fakeScanner{errByID: map[string]error{badID.String(): errors.New("invalid scan")}}
	app := New(Config{Scanner: scanner})

	pkgs := []scan.Package{
		{ID: okID},
		{ID: badID},
	}

	failures, err := app.ScanPackages(context.Background(), testShipmentID(), pkgs)
	if err != nil {
		t.Fatalf("ScanPackages: expected partial success with nil error, got %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].PackageID != badID {
		t.Fatalf("expected failure for package %s, got %s", badID, failures[0].PackageID)
	}
	if scanner.calls != len(pkgs) {
		t.Fatalf("expected both packages to be attempted, got %d calls", scanner.calls)
	}
}

func TestScanPackagesEmpty(t *testing.T) {
	scanner := &fakeScanner{}
	app := New(Config{Scanner: scanner})

	failures, err := app.ScanPackages(context.Background(), testShipmentID(), nil)
	if err != nil {
		t.Fatalf("ScanPackages(empty): %v", err)
	}
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %d", len(failures))
	}
	if scanner.calls != 0 {
		t.Fatalf("expected no scanner calls, got %d", scanner.calls)
	}
}
