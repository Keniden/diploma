package values

const (
	statusLabelUnspecified = "UNSPECIFIED"
	statusLabelUnloading   = "UNLOADING"
	statusLabelCompleted   = "COMPLETED"
	statusLabelCanceled    = "CANCELED"
)

type ShipmentStatus int

const (
	ShipmentStatusUnspecified      ShipmentStatus = 0 // "UNSPECIFIED"
	ShipmentStatusNew              ShipmentStatus = 1 // "NEW"
	ShipmentStatusAssigned         ShipmentStatus = 2 // "ASSIGNED"
	ShipmentStatusReserveLoading   ShipmentStatus = 3 // "RESERVE_LOADING"
	ShipmentStatusLoading          ShipmentStatus = 4 // "LOADING"
	ShipmentStatusInProgress       ShipmentStatus = 5 // "IN_PROGRESS"
	ShipmentStatusReserveUnloading ShipmentStatus = 6 // "RESERVE_UNLOADING"
	ShipmentStatusUnloading        ShipmentStatus = 7 // "UNLOADING"
	ShipmentStatusCompleted        ShipmentStatus = 8 // "COMPLETED"
	ShipmentStatusCanceled         ShipmentStatus = 9 // "CANCELED"
)

func ActiveShipmentStatuses() []ShipmentStatus {
	return []ShipmentStatus{
		ShipmentStatusNew,
		ShipmentStatusAssigned,
		ShipmentStatusReserveLoading,
		ShipmentStatusLoading,
		ShipmentStatusInProgress,
		ShipmentStatusReserveUnloading,
		ShipmentStatusUnloading,
	}
}

func (s ShipmentStatus) ToString() string {
	switch s {
	case ShipmentStatusNew:
		return "NEW"
	case ShipmentStatusAssigned:
		return "ASSIGNED"
	case ShipmentStatusReserveLoading:
		return "RESERVE_LOADING"
	case ShipmentStatusLoading:
		return "LOADING"
	case ShipmentStatusInProgress:
		return "IN_PROGRESS"
	case ShipmentStatusReserveUnloading:
		return "RESERVE_UNLOADING"
	case ShipmentStatusUnloading:
		return statusLabelUnloading
	case ShipmentStatusCompleted:
		return statusLabelCompleted
	case ShipmentStatusCanceled:
		return statusLabelCanceled
	default:
		return statusLabelUnspecified
	}
}
