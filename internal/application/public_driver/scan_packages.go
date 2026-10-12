package publicdriver

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

// ScanPackages applies the scanned packages to the given shipment.
func (a *Application) ScanPackages(
	ctx context.Context,
	shipmentID values.ShipmentID,
	scannedPackages []scan.Package,
) error {
	if err := a.scannerService.ScanPackages(ctx, shipmentID, scannedPackages); err != nil {
		return fmt.Errorf("scan packages: %w", err)
	}

	return nil
}
