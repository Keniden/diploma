package values

type PackageStatus int

const (
	PackageStatusUnspecified PackageStatus = 0 // "UNSPECIFIED"
	PackageStatusReadyToLoad PackageStatus = 1 // "READY_TO_LOAD"
	PackageStatusLoaded      PackageStatus = 2 // "LOADED"
	PackageStatusInTransit   PackageStatus = 3 // "IN_TRANSIT"
	PackageStatusUnloading   PackageStatus = 4 // "UNLOADING"
	PackageStatusUnloaded    PackageStatus = 5 // "UNLOADED"
	PackageStatusCanceled    PackageStatus = 6 // "CANCELED"
)

func (s PackageStatus) ToString() string {
	switch s {
	case PackageStatusReadyToLoad:
		return "READY_TO_LOAD"
	case PackageStatusLoaded:
		return "LOADED"
	case PackageStatusInTransit:
		return "IN_TRANSIT"
	case PackageStatusUnloading:
		return statusLabelUnloading
	case PackageStatusUnloaded:
		return "UNLOADED"
	case PackageStatusCanceled:
		return statusLabelCanceled
	default:
		return statusLabelUnspecified
	}
}
