package entities

import (
	"slices"
	"time"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

// Package is aggregate
type Package struct {
	ID         values.PackageID
	ShipmentID values.ShipmentID
	Status     values.PackageStatus
	IsReturn   bool
	TareID     values.TareID
	TareType   values.TareType
	Actions    []*PackageAction
}

type PackageAction struct {
	RoutePointID    values.RoutePointID
	Type            values.PackageActionType
	Status          values.PackageActionStatus
	StatusUpdatedAt *time.Time
}

func (pa PackageAction) Scanned() bool {
	return pa.Status != values.RoutePointPackageStatusTODO
}

func NewPackage() *Package {
	return &Package{
		ID:       values.NewID(),
		Status:   values.PackageStatusUnspecified,
		IsReturn: false,
		Actions:  make([]*PackageAction, 0),
	}
}

func ReadyToLoadPackages(pkgs []*Package) []*Package {
	readyToLoadPackages := make([]*Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.IsReadyToLoad() {
			readyToLoadPackages = append(readyToLoadPackages, pkg)
		}
	}
	return readyToLoadPackages
}

func LoadedPackages(pkgs []*Package) []*Package {
	loadedPackages := make([]*Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.IsLoaded() {
			loadedPackages = append(loadedPackages, pkg)
		}
	}
	return loadedPackages
}

func PackageIDs(pkgs []*Package) []values.PackageID {
	ids := make([]values.PackageID, 0, len(pkgs))

	for _, pkg := range pkgs {
		ids = append(ids, pkg.ID)
	}

	return ids
}

func (p *Package) AssignToShipment(id values.ShipmentID) {
	p.ShipmentID = id
}

func (p *Package) IsLoaded() bool {
	return p.Status == values.PackageStatusLoaded
}

func (p *Package) IsCarried() bool {
	switch p.Status {
	case values.PackageStatusLoaded,
		values.PackageStatusInTransit,
		values.PackageStatusUnloading:
		return true
	default:
		return false
	}
}

func (p *Package) IsReadyToLoad() bool {
	return p.Status == values.PackageStatusReadyToLoad
}

var packageStatusModel = map[values.PackageStatus][]values.PackageStatus{
	values.PackageStatusUnspecified: {values.PackageStatusReadyToLoad, values.PackageStatusCanceled},
	values.PackageStatusReadyToLoad: {values.PackageStatusLoaded, values.PackageStatusCanceled},
	values.PackageStatusLoaded:      {values.PackageStatusInTransit},
	values.PackageStatusInTransit:   {values.PackageStatusUnloading},
	values.PackageStatusUnloading:   {values.PackageStatusUnloading, values.PackageStatusUnloaded},
	values.PackageStatusUnloaded:    {},
	values.PackageStatusCanceled:    {},
}

func (p *Package) canTransition(next values.PackageStatus) bool {
	return slices.Contains(packageStatusModel[p.Status], next)
}

func (p *Package) setStatus(next values.PackageStatus) error {
	if !p.canTransition(next) {
		return lmterrors.NewInvalidPackageStatusError("package status transition is prohibited").WithAttributes(
			lmterrors.Attributes{
				"actual":   p.Status.ToString(),
				"expected": next.ToString(),
			},
		)
	}
	p.Status = next
	return nil
}

func (p *Package) cancelPackage() error {
	if err := p.setStatus(values.PackageStatusCanceled); err != nil {
		return err
	}

	for _, action := range p.Actions {
		action.Status = values.RoutePointPackageStatusCanceled
		now := time.Now()
		action.StatusUpdatedAt = &now
	}

	return nil
}

func (p *Package) Loaded() error {
	return p.setStatus(values.PackageStatusLoaded)
}

func (p *Package) Unloaded() error {
	return p.setStatus(values.PackageStatusUnloaded)
}

func (p *Package) ToReadyToLoad() {
	p.Status = values.PackageStatusReadyToLoad
}

func (p *Package) ToDirect() {
	p.IsReturn = false
}

func (p *Package) ToReturn() {
	p.IsReturn = true
}

func (p *Package) MarkInTransit() error {
	return p.setStatus(values.PackageStatusInTransit)
}

func (p *Package) MarkUnloading() error {
	return p.setStatus(values.PackageStatusUnloading)
}

func (p *Package) CancelLoad() (bool, error) {
	for _, action := range p.Actions {
		if action.Type != values.RoutePointActionTypeLoad {
			continue
		}
		if action.Status != values.RoutePointPackageStatusTODO {
			continue
		}

		if err := p.cancelPackage(); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func NewPackageFromDemand(demand values.PackageDemand) *Package {
	pkg := &Package{
		ID:       demand.PackageID,
		Status:   values.PackageStatusReadyToLoad,
		TareID:   demand.TareID,
		TareType: demand.TareType,
		Actions: []*PackageAction{
			{
				RoutePointID: demand.LoadPointID,
				Type:         values.RoutePointActionTypeLoad,
				Status:       values.RoutePointPackageStatusTODO,
			},
			{
				RoutePointID: demand.UnloadPointID,
				Type:         values.RoutePointActionTypeUnload,
				Status:       values.RoutePointPackageStatusTODO,
			},
		},
	}

	switch demand.Direction {
	case values.PackageDemandDirectionForward:
		pkg.ToDirect()
	case values.PackageDemandDirectionBackward:
		pkg.ToReturn()
	}

	return pkg
}

func NewPackagesFromDemands(demands []values.PackageDemand) []*Package {
	pkgs := make([]*Package, 0, len(demands))
	for _, demand := range demands {
		pkgs = append(pkgs, NewPackageFromDemand(demand))
	}
	return pkgs
}

// ApplyScan applies a scanned action to a package.
// It returns a domain event, a bool indicating whether the scan was applied, and an error if any.
func (p *Package) ApplyScan(scanned scan.Package) (bool, error) {
	if scanned.ID != p.ID {
		return false, nil
	}
	if scanned.ShipmentID != p.ShipmentID {
		return false, nil
	}

	for _, action := range p.Actions {
		if action.Type != scanned.ActionType {
			continue
		}
		if action.RoutePointID != scanned.RoutePointID {
			continue
		}

		switch scanned.ActionType {
		case values.RoutePointActionTypeLoad:
			if err := p.Loaded(); err != nil {
				return false, err
			}
		case values.RoutePointActionTypeUnload:
			if err := p.Unloaded(); err != nil {
				return false, err
			}
		default:
			return false, lmterrors.NewInvalidPackageScanActionError("unsupported scan action").WithAttributes(
				lmterrors.Attributes{
					"packageID": scanned.ID.String(),
					"actual":    scanned.ActionType.ToString(),
				},
			)
		}

		action.Status = values.RoutePointPackageStatusDone
		now := scanned.Timestamp
		action.StatusUpdatedAt = &now

		return true, nil
	}

	return false, nil
}
