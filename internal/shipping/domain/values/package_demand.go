package values

type PackageDemand struct {
	PackageID     PackageID
	TareID        TareID
	TareType      TareType
	LoadPointID   RoutePointID
	UnloadPointID RoutePointID
	Direction     PackageDemandDirection
}

type PackageDemandDirection int

const (
	PackageDemandDirectionForward PackageDemandDirection = iota
	PackageDemandDirectionBackward
)
