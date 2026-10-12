package ports

import "errors"

var (
	ErrShipmentNotFound      = errors.New("shipment not found")
	ErrRouteNotFound         = errors.New("route not found")
	ErrPackageNotFound       = errors.New("package not found")
	ErrReserveNotFound       = errors.New("reserve not found")
	ErrRouteTemplateNotFound = errors.New("route template not found")
)
