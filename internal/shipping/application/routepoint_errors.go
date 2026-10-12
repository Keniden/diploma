package application

import (
	_http "net/http"

	_codes "google.golang.org/grpc/codes"

	lmterrors "last-mile-task/internal/errors"
)

// TODO(LASTMILE-032): the real repository generates these constructors via
// errgen into internal/errors (driven by errors.yaml). The skeleton's
// internal/errors deliberately exposes only the generic New/WithAttributes
// API, so these constructors are re-implemented here to keep the exact call
// shape of the reference RoutePointService. Once the shared errgen output
// lands, delete this file and point routepoint.go's calls at
// last-mile-task/internal/errors.

// Error codes, mirroring the generated ones of the real repository
// (internal/errors/definitions.go).
const (
	invalidRoutePointNotFoundCode                 = "lmt_invalid_route_point_not_found"
	invalidRoutePointDriverAlreadyArrivedCode     = "lmt_invalid_route_point_driver_already_arrived"
	invalidRoutePointNotReadyForTargetArrivalCode = "lmt_invalid_route_point_not_ready_for_target_arrival"
	invalidRoutePointNoLoadedPackagesAtSourceCode = "lmt_invalid_route_point_no_loaded_packages_at_source"
	invalidPackagePendingUnloadCode               = "lmt_invalid_package_pending_unload"
)

func buildRoutePointError(message, code string, protocode _codes.Code, httpcode int, details string) *lmterrors.Error {
	return lmterrors.New(
		message,
		code,
		int32(protocode),
		httpcode,
	).WithDetails(details)
}

func newInvalidRoutePointNotFoundError(details string) *lmterrors.Error {
	return buildRoutePointError("Route point not found.", invalidRoutePointNotFoundCode, _codes.NotFound, _http.StatusNotFound, details)
}

func newInvalidRoutePointDriverAlreadyArrivedError(details string) *lmterrors.Error {
	return buildRoutePointError("Driver has already arrived at another route point.", invalidRoutePointDriverAlreadyArrivedCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

func newInvalidRoutePointNotReadyForTargetArrivalError(details string) *lmterrors.Error {
	return buildRoutePointError("Some route points have packages which are not unloaded.", invalidRoutePointNotReadyForTargetArrivalCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

func newInvalidRoutePointNoLoadedPackagesAtSourceError(details string) *lmterrors.Error {
	return buildRoutePointError("No loaded packages at source route point.", invalidRoutePointNoLoadedPackagesAtSourceCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

func newInvalidPackagePendingUnloadError(details string) *lmterrors.Error {
	return buildRoutePointError("Shipment has packages that have not been unloaded yet.", invalidPackagePendingUnloadCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// wrapLmtError mirrors internal/errors.Wrap(e *Error, err error) *Error of the
// real repository: the primary error e is kept and err is attached to its
// chain (the skeleton only exposes the single-argument Wrap).
func wrapLmtError(e *lmterrors.Error, err error) *lmterrors.Error {
	if e == nil || err == nil {
		return e
	}
	return e.Wrap(err)
}

// Attributes is the view of the shared structured attributes used by
// routepoint.go, matching the real repository's lmterrors.Attributes.
type Attributes = lmterrors.Attributes
