package values

type RoutePointStatus int

const (
	RoutePointStatusWaitingArrival      RoutePointStatus = 0 // "WAITING_ARRIVAL"
	RoutePointStatusDriverArrived       RoutePointStatus = 1 // "DRIVER_ARRIVED"
	RoutePointStatusPackagesNotUnloaded RoutePointStatus = 2 // "PACKAGES_NOT_UNLOADED"
	RoutePointStatusPackagesNotLoaded   RoutePointStatus = 3 // "PACKAGES_NOT_LOADED"
	RoutePointStatusCompleted           RoutePointStatus = 4 // "COMPLETED"
)

func (s RoutePointStatus) ToString() string {
	switch s {
	case RoutePointStatusWaitingArrival:
		return "WAITING_ARRIVAL"
	case RoutePointStatusDriverArrived:
		return "DRIVER_ARRIVED"
	case RoutePointStatusPackagesNotUnloaded:
		return "PACKAGES_NOT_UNLOADED"
	case RoutePointStatusPackagesNotLoaded:
		return "PACKAGES_NOT_LOADED"
	case RoutePointStatusCompleted:
		return statusLabelCompleted
	default:
		return statusLabelUnspecified
	}
}
