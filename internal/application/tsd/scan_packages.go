package tsd

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

// ScanFailure describes a package that could not be applied by Scanner. The
// remaining packages are still applied, so a scan is a partial success.
type ScanFailure struct {
	PackageID values.PackageID
	Err       error
}

// ScanPackages scans and verifies packages at a route point via Scanner, with
// partial-success semantics: a package that fails to apply is recorded in the
// returned failures and the remaining packages are applied. The successful
// scans publish PackageScannedEvent (outbox cargo + WMS) inside the writer
// transaction.
func (a *Application) ScanPackages(
	ctx context.Context,
	shipmentID values.ShipmentID,
	scannedPackages []scan.Package,
) ([]ScanFailure, error) {
	var failures []ScanFailure

	for _, pkg := range scannedPackages {
		if err := a.scanner.ScanPackages(ctx, shipmentID, []scan.Package{pkg}); err != nil {
			failures = append(failures, ScanFailure{
				PackageID: pkg.ID,
				Err:       fmt.Errorf("scan package %s: %w", pkg.ID, err),
			})
		}
	}

	return failures, nil
}
