package shipment

import (
	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

type DetailedShipment struct {
	readmodels.Shipment
	DetailedRoute
	Reserve            *readmodels.ShipmentReserve
	SchedulesByPointID map[int64]*WeekSchedule
}

type WeekSchedule struct {
	Days []WeekDay
}

type WeekDay struct {
	Day      int32
	Schedule DaySchedule
}

type DaySchedule struct {
	WorkIntervals []WorkInterval
}

type WorkInterval struct {
	Start ScheduleTime
	End   ScheduleTime
}

type ScheduleTime struct {
	Hours   int32
	Minutes int32
}

type DetailedRoute struct {
	readmodels.Route
	RoutePoints []DetailedRoutePoint
}

type DetailedRoutePoint struct {
	readmodels.RoutePoint
	Packages []DetailedPackage
}

func (rp DetailedRoutePoint) UnloadPackageCount(loadDonePackageIDs map[uuid.UUID]struct{}) int32 {
	var count int32
	for _, pkg := range rp.Packages {
		if pkg.Action != int(values.RoutePointActionTypeUnload) {
			continue
		}
		if pkg.Status != int(values.RoutePointPackageStatusTODO) &&
			pkg.Status != int(values.RoutePointPackageStatusDone) {
			continue
		}
		if _, ok := loadDonePackageIDs[pkg.ID]; ok {
			count++
		}
	}
	return count
}

func (s *DetailedShipment) LoadDonePackageIDs() map[uuid.UUID]struct{} {
	loadDonePackageIDs := make(map[uuid.UUID]struct{})
	for _, rp := range s.RoutePoints {
		for _, pkg := range rp.Packages {
			if pkg.Action == int(values.RoutePointActionTypeLoad) &&
				pkg.Status == int(values.RoutePointPackageStatusDone) {
				loadDonePackageIDs[pkg.ID] = struct{}{}
			}
		}
	}
	return loadDonePackageIDs
}

type DetailedPackage struct {
	readmodels.Package
	readmodels.PackageAction
}

func NewDetailedPackages(packages []readmodels.Package, actions []readmodels.PackageAction) []DetailedPackage {
	// TODO(LASTMILE-031): the real repository builds this map via
	// lo.SliceToMap (github.com/samber/lo). samber/lo is not available in the
	// skeleton go.mod, so the map is built with an inline loop instead.
	packagesByID := make(map[uuid.UUID]readmodels.Package, len(packages))
	for _, p := range packages {
		packagesByID[p.ID] = p
	}

	detailedPackages := make([]DetailedPackage, 0, len(packages))
	for _, action := range actions {
		pkg, ok := packagesByID[action.PackageID]
		if !ok {
			continue
		}

		detailedPackages = append(detailedPackages, DetailedPackage{
			Package:       pkg,
			PackageAction: action,
		})
	}

	return detailedPackages
}
