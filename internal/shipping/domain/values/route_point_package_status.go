package values

type PackageActionStatus int

const (
	RoutePointPackageStatusUnspecified PackageActionStatus = 0 // "UNSPECIFIED"
	RoutePointPackageStatusTODO        PackageActionStatus = 1 // "TODO"
	RoutePointPackageStatusDone        PackageActionStatus = 2 // "DONE"
	RoutePointPackageStatusCanceled    PackageActionStatus = 3 // "CANCELED"
)

func (s PackageActionStatus) ToString() string {
	switch s {
	case RoutePointPackageStatusTODO:
		return "TODO"
	case RoutePointPackageStatusDone:
		return "DONE"
	case RoutePointPackageStatusCanceled:
		return statusLabelCanceled
	default:
		return statusLabelUnspecified
	}
}
