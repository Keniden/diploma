package scanner

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

type PackageScanner struct {
	packages              ports.PackageRepository
	packageEventPublisher ports.PackageEventPublisher
	tx                    ports.Transactional
}

func NewPackageScanner(
	packages ports.PackageRepository,
	packageEventPublisher ports.PackageEventPublisher,
	tx ports.Transactional,
) *PackageScanner {
	return &PackageScanner{
		packages:              packages,
		packageEventPublisher: packageEventPublisher,
		tx:                    tx,
	}
}

func (s *PackageScanner) ScanPackages(
	ctx context.Context,
	shipmentID values.ShipmentID,
	scannedPackages []scan.Package,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		pkgs, err := s.packages.GetPackagesByIDs(txCtx, shipmentID, scan.PackageIDs(scannedPackages)...)
		if err != nil {
			return fmt.Errorf("get packages by id: %w", err)
		}

		packageEvents, appliedScans, err := s.applyScans(pkgs, scannedPackages)
		if err != nil {
			return fmt.Errorf("apply scans: %w", err)
		}

		for _, applied := range appliedScans {
			if err := s.packages.SavePackageScan(txCtx, applied.pkg, applied.action); err != nil {
				return fmt.Errorf("save package scan: %w", err)
			}
		}

		return s.packageEventPublisher.PublishPackageEvent(txCtx, events.PackageScannedEvent{
			Packages: packageEvents,
		})
	})
}

type appliedPackageScan struct {
	pkg    *entities.Package
	action *entities.PackageAction
}

func (s *PackageScanner) applyScans(
	pkgs []*entities.Package,
	scannedPackages []scan.Package,
) ([]events.ScannedPackage, []appliedPackageScan, error) {
	packageEvents := make([]events.ScannedPackage, 0, len(scannedPackages))
	appliedScans := make([]appliedPackageScan, 0, len(scannedPackages))

	pkgByID := make(map[string]*entities.Package, len(pkgs))
	for _, pkg := range pkgs {
		pkgByID[pkg.ID.String()] = pkg
	}

	for _, scannedPackage := range scannedPackages {
		pkg, ok := pkgByID[scannedPackage.ID.String()]
		if !ok {
			continue
		}

		applied, err := pkg.ApplyScan(scannedPackage)
		if err != nil {
			return nil, nil, fmt.Errorf("apply scan: %w", err)
		}

		if !applied {
			continue
		}

		action := findScannedAction(pkg, scannedPackage)
		if action == nil {
			continue
		}

		appliedScans = append(appliedScans, appliedPackageScan{
			pkg:    pkg,
			action: action,
		})

		packageEvents = append(packageEvents, events.ScannedPackage{
			PackageID:    scannedPackage.ID,
			TareID:       pkg.TareID,
			TareType:     pkg.TareType,
			RoutePointID: scannedPackage.RoutePointID,
			PointID:      scannedPackage.PointID,
			ShipmentID:   scannedPackage.ShipmentID,
			ActionType:   scannedPackage.ActionType,
			Timestamp:    scannedPackage.Timestamp,
			TareSticker:  scannedPackage.TareSticker,
		})
	}

	return packageEvents, appliedScans, nil
}

func findScannedAction(pkg *entities.Package, scanned scan.Package) *entities.PackageAction {
	for _, action := range pkg.Actions {
		if action.Type == scanned.ActionType && action.RoutePointID == scanned.RoutePointID {
			return action
		}
	}

	return nil
}
