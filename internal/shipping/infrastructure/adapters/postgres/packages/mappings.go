package packages

import (
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

const (
	dbPackageStatusReadyToLoad = 1
	dbPackageStatusLoaded      = 2
	dbPackageStatusInTransit   = 3
	dbPackageStatusUnloading   = 4
	dbPackageStatusUnloaded    = 5
	dbPackageStatusUnspecified = 0
)

func packageStatusFromDB(v int) values.PackageStatus {
	return values.PackageStatus(v)
}

func packageStatusToDB(s values.PackageStatus) int {
	return int(s)
}

func packageFromRecord(rec packageRecord) *entities.Package {
	return &entities.Package{
		ID:         rec.ID,
		ShipmentID: rec.ShipmentID,
		Status:     packageStatusFromDB(rec.Status),
		IsReturn:   rec.IsReturn,
		TareID:     values.TareID(rec.TareID),
		TareType:   values.TareType(rec.TareType),
		Actions:    make([]*entities.PackageAction, 0),
	}
}

func packageActionTypeFromDB(v int) values.PackageActionType {
	return values.PackageActionType(v)
}

func packageActionTypeToDB(t values.PackageActionType) int {
	return int(t)
}

func packageActionStatusFromDB(v int) values.PackageActionStatus {
	return values.PackageActionStatus(v)
}

func packageActionStatusToDB(s values.PackageActionStatus) int {
	return int(s)
}
