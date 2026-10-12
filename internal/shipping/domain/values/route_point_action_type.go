package values

type PackageActionType int

const (
	RoutePointActionTypeUnspecified PackageActionType = 0 // "UNSPECIFIED"
	RoutePointActionTypeLoad        PackageActionType = 1 // "LOAD"
	RoutePointActionTypeUnload      PackageActionType = 2 // "UNLOAD"
	// Can use to specify which route point needs reserve to be queued, TODO
	RoutePointActionTypeReserve PackageActionType = 3 // "RESERVE"
)

func (t PackageActionType) ToString() string {
	switch t {
	case RoutePointActionTypeLoad:
		return "LOAD"
	case RoutePointActionTypeUnload:
		return "UNLOAD"
	case RoutePointActionTypeReserve:
		return "RESERVE"
	default:
		return "UNSPECIFIED"
	}
}
