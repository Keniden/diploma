package shipment_test

import (
	"testing"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

func TestDetailedRoutePoint_UnloadPackageCount(t *testing.T) {
	loadPointID := uuid.New()
	unloadPointID := uuid.New()
	loadedPackageID := uuid.New()
	unloadedPackageID := uuid.New()
	notLoadedPackageID := uuid.New()
	loadDoneOnlyAtUnloadPointID := uuid.New()
	canceledPackageID := uuid.New()

	routePoints := []shipment.DetailedRoutePoint{
		{
			RoutePoint: readmodels.RoutePoint{ID: loadPointID},
			Packages: []shipment.DetailedPackage{
				{
					Package: readmodels.Package{ID: loadedPackageID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeLoad),
						Status: int(values.RoutePointPackageStatusDone),
					},
				},
				{
					Package: readmodels.Package{ID: unloadedPackageID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeLoad),
						Status: int(values.RoutePointPackageStatusDone),
					},
				},
			},
		},
		{
			RoutePoint: readmodels.RoutePoint{ID: unloadPointID},
			Packages: []shipment.DetailedPackage{
				{
					Package: readmodels.Package{ID: loadedPackageID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeUnload),
						Status: int(values.RoutePointPackageStatusTODO),
					},
				},
				{
					Package: readmodels.Package{ID: unloadedPackageID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeUnload),
						Status: int(values.RoutePointPackageStatusDone),
					},
				},
				{
					Package: readmodels.Package{ID: notLoadedPackageID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeUnload),
						Status: int(values.RoutePointPackageStatusTODO),
					},
				},
				{
					Package: readmodels.Package{ID: loadDoneOnlyAtUnloadPointID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeLoad),
						Status: int(values.RoutePointPackageStatusDone),
					},
				},
				{
					Package: readmodels.Package{ID: loadDoneOnlyAtUnloadPointID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeUnload),
						Status: int(values.RoutePointPackageStatusTODO),
					},
				},
				{
					Package: readmodels.Package{ID: canceledPackageID},
					PackageAction: readmodels.PackageAction{
						Action: int(values.RoutePointActionTypeUnload),
						Status: int(values.RoutePointPackageStatusCanceled),
					},
				},
			},
		},
	}

	ship := shipment.DetailedShipment{
		DetailedRoute: shipment.DetailedRoute{
			RoutePoints: routePoints,
		},
	}
	loadDonePackageIDs := ship.LoadDonePackageIDs()

	if got := routePoints[1].UnloadPackageCount(loadDonePackageIDs); got != 3 {
		t.Fatalf("expected unload package count 3, got %d", got)
	}
}
