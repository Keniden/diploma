package scan

import (
	"time"

	"last-mile-task/internal/shipping/domain/values"
)

type Package struct {
	ID           values.PackageID
	RoutePointID values.RoutePointID
	PointID      values.PointID
	ShipmentID   values.ShipmentID
	ActionType   values.PackageActionType
	Timestamp    time.Time
	TareSticker  string
}

func PackageIDs(pkgs []Package) []values.PackageID {
	ids := make([]values.PackageID, 0, len(pkgs))

	for _, pkg := range pkgs {
		ids = append(ids, pkg.ID)
	}

	return ids
}
