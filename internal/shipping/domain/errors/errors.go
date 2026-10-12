// Package errors adapts the shipping domain to the skeleton's shared error
// machinery.
//
// TODO(LASTMILE-020): the real repository generates the per-error constructors
// (NewInvalidShipmentStatusTransitionError and friends) via errgen into
// internal/errors, driven by errors.yaml. The skeleton's internal/errors
// deliberately exposes only the generic New/WithAttributes/Is API (see
// internal/errors/errors.go). This package re-implements the constructors used
// by the shipping domain on top of that generic API so the ported domain code
// keeps the exact call shape of the real repository. Once the shared errgen
// output lands, delete this package and point the domain imports (lmterrors)
// at last-mile-task/internal/errors.
package errors

import (
	_http "net/http"

	_codes "google.golang.org/grpc/codes"
	lmterrors "last-mile-task/internal/errors"
)

// Attributes is the domain view of the shared structured attributes.
type Attributes = lmterrors.Attributes

// Error codes. Mirrors the generated error codes of the real repository
// (internal/errors/definitions.go).
const (
	InvalidReserveAlreadyExistsCode                = "lmt_invalid_reserve_already_exists"
	InvalidRoutePointNotFoundCode                  = "lmt_invalid_route_point_not_found"
	InvalidReserveMustBeSourceCode                 = "lmt_invalid_reserve_must_be_source"
	InvalidReserveShipmentNotAssignedCode          = "lmt_invalid_reserve_shipment_not_assigned"
	InvalidShipmentPendingUnloadAtAnotherPointCode = "lmt_invalid_shipment_pending_unload_at_another_point"
	InvalidShipmentNoLoadedPackagesAtTargetCode    = "lmt_invalid_shipment_no_loaded_packages_at_target"
	InvalidShipmentMustBeReserveStatusCode         = "lmt_invalid_shipment_must_be_reserve_status"
	InvalidShipmentNoActiveReserveCode             = "lmt_invalid_shipment_no_active_reserve"
	InvalidShipmentCannotCompleteCode              = "lmt_invalid_shipment_cannot_complete"
	InvalidPackagePendingUnloadCode                = "lmt_invalid_package_pending_unload"
	InvalidShipmentCannotStartCode                 = "lmt_invalid_shipment_cannot_start"
	InvalidShipmentInvalidRouteCode                = "lmt_invalid_shipment_invalid_route"
	InvalidShipmentNoLoadedPackagesCode            = "lmt_invalid_shipment_no_loaded_packages"
	InvalidRouteNoPackagesAtSourceCode             = "lmt_invalid_route_no_packages_at_source"
	InvalidDriverHasActiveShipmentsCode            = "lmt_invalid_driver_has_active_shipments"
	InvalidShipmentCarLicensePlateCode             = "lmt_invalid_shipment_car_license_plate"

	InvalidShipmentStatusTransitionCode   = "lmt_invalid_shipment_status_transition"
	InvalidPackageStatusCode              = "lmt_invalid_package_status"
	InvalidPackageScanActionCode          = "lmt_invalid_package_scan_action"
	InvalidRouteTooShortCode              = "lmt_invalid_route_too_short"
	InvalidRoutePointStatusTransitionCode = "lmt_invalid_route_point_status_transition"
	InvalidReserveCannotApproveCode       = "lmt_invalid_reserve_cannot_approve"
	InvalidReserveCannotReleaseCode       = "lmt_invalid_reserve_cannot_release"
	InvalidReserveCannotCancelCode        = "lmt_invalid_reserve_cannot_cancel"
	InvalidReserveCannotExpireCode        = "lmt_invalid_reserve_cannot_expire"
)

// Is reports whether the error chain contains an error with the given code.
func Is(err error, code string) bool {
	return lmterrors.Is(err, code)
}

func build(message, code string, protocode _codes.Code, httpcode int, details string) *lmterrors.Error {
	return lmterrors.New(
		message,
		code,
		int32(protocode),
		httpcode,
	).WithDetails(details)
}

// NewInvalidShipmentStatusTransitionError builds a prohibited shipment status
// transition error (FailedPrecondition).
func NewInvalidShipmentStatusTransitionError(details string) *lmterrors.Error {
	return build("Shipment status transition from {{ .actual }} to {{ .expected }} is prohibited.", InvalidShipmentStatusTransitionCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidPackageStatusError builds an incorrect package status error
// (InvalidArgument).
func NewInvalidPackageStatusError(details string) *lmterrors.Error {
	return build("Incorrect package status {{ .actual }}, expected {{ .expected }}", InvalidPackageStatusCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// NewInvalidPackageScanActionError builds an incompatible package scan action
// error (InvalidArgument).
func NewInvalidPackageScanActionError(details string) *lmterrors.Error {
	return build("Unable to scan package {{ .packageID }} with action {{ .actual }}, expected {{ .expected }}", InvalidPackageScanActionCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// NewInvalidRouteTooShortError builds a too-short route error (InvalidArgument).
func NewInvalidRouteTooShortError(details string) *lmterrors.Error {
	return build("Route is too short.", InvalidRouteTooShortCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// NewInvalidRoutePointNotFoundError builds a route point not found error
// (NotFound).
func NewInvalidRoutePointNotFoundError(details string) *lmterrors.Error {
	return build("Route point not found.", InvalidRoutePointNotFoundCode, _codes.NotFound, _http.StatusNotFound, details)
}

// NewInvalidRoutePointStatusTransitionError builds a prohibited route point
// status transition error (InvalidArgument).
func NewInvalidRoutePointStatusTransitionError(details string) *lmterrors.Error {
	return build("Route point status transition from {{ .actual }} to {{ .expected }} is prohibited.", InvalidRoutePointStatusTransitionCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// NewInvalidReserveCannotApproveError builds a reserve not approvable error
// (FailedPrecondition).
func NewInvalidReserveCannotApproveError(details string) *lmterrors.Error {
	return build("Reserve cannot be approved.", InvalidReserveCannotApproveCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidReserveCannotReleaseError builds a reserve not releasable error
// (FailedPrecondition).
func NewInvalidReserveCannotReleaseError(details string) *lmterrors.Error {
	return build("Attempt to release not active reserve.", InvalidReserveCannotReleaseCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidReserveCannotCancelError builds a reserve not cancellable error
// (FailedPrecondition).
func NewInvalidReserveCannotCancelError(details string) *lmterrors.Error {
	return build("Reserve cannot be canceled.", InvalidReserveCannotCancelCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidReserveCannotExpireError builds a reserve not expirable error
// (FailedPrecondition).
func NewInvalidReserveCannotExpireError(details string) *lmterrors.Error {
	return build("Reserve cannot be expired.", InvalidReserveCannotExpireCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidReserveAlreadyExistsError builds a duplicate reserve error
// (FailedPrecondition).
func NewInvalidReserveAlreadyExistsError(details string) *lmterrors.Error {
	return build("Shipment already has a queued or active reserve.", InvalidReserveAlreadyExistsCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidReserveMustBeSourceError builds a reserve must be at source error
// (InvalidArgument).
func NewInvalidReserveMustBeSourceError(details string) *lmterrors.Error {
	return build("Route point must be source to reserve gate.", InvalidReserveMustBeSourceCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// NewInvalidReserveShipmentNotAssignedError builds a shipment not assigned for
// reserve error (FailedPrecondition).
func NewInvalidReserveShipmentNotAssignedError(details string) *lmterrors.Error {
	return build("Shipment must be in assigned status to reserve gate at source.", InvalidReserveShipmentNotAssignedCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentPendingUnloadAtAnotherPointError builds a pending unload at
// another route point error (FailedPrecondition).
func NewInvalidShipmentPendingUnloadAtAnotherPointError(details string) *lmterrors.Error {
	return build("Shipment has pending unload at another route point.", InvalidShipmentPendingUnloadAtAnotherPointCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentNoLoadedPackagesAtTargetError builds a no loaded packages at
// target error (FailedPrecondition).
func NewInvalidShipmentNoLoadedPackagesAtTargetError(details string) *lmterrors.Error {
	return build("Shipment has no loaded packages at target route point.", InvalidShipmentNoLoadedPackagesAtTargetCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentMustBeReserveStatusError builds a shipment must be in
// reserve status error (FailedPrecondition).
func NewInvalidShipmentMustBeReserveStatusError(details string) *lmterrors.Error {
	return build("Shipment must be in reserve loading or reserve unloading status.", InvalidShipmentMustBeReserveStatusCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentNoActiveReserveError builds a no active reserve error
// (FailedPrecondition).
func NewInvalidShipmentNoActiveReserveError(details string) *lmterrors.Error {
	return build("No active shipment reserve.", InvalidShipmentNoActiveReserveCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentCannotCompleteError builds a shipment not completable error
// (FailedPrecondition).
func NewInvalidShipmentCannotCompleteError(details string) *lmterrors.Error {
	return build("Shipment must be in unloading or in progress status to complete.", InvalidShipmentCannotCompleteCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidPackagePendingUnloadError builds a package pending unload error
// (InvalidArgument).
func NewInvalidPackagePendingUnloadError(details string) *lmterrors.Error {
	return build("Shipment has packages that have not been unloaded yet.", InvalidPackagePendingUnloadCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}

// NewInvalidShipmentCannotStartError builds a shipment not startable error
// (FailedPrecondition).
func NewInvalidShipmentCannotStartError(details string) *lmterrors.Error {
	return build("Shipment must have completed loading to start.", InvalidShipmentCannotStartCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentInvalidRouteError builds an invalid route error
// (FailedPrecondition).
func NewInvalidShipmentInvalidRouteError(details string) *lmterrors.Error {
	return build("Shipment has invalid route.", InvalidShipmentInvalidRouteCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentNoLoadedPackagesError builds a no loaded packages error
// (FailedPrecondition).
func NewInvalidShipmentNoLoadedPackagesError(details string) *lmterrors.Error {
	return build("Shipment has no loaded packages.", InvalidShipmentNoLoadedPackagesCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidRouteNoPackagesAtSourceError builds a route without source packages
// error (FailedPrecondition).
func NewInvalidRouteNoPackagesAtSourceError(details string) *lmterrors.Error {
	return build("Route has no packages at source.", InvalidRouteNoPackagesAtSourceCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidDriverHasActiveShipmentsError builds a driver with active shipments
// error (FailedPrecondition).
func NewInvalidDriverHasActiveShipmentsError(details string) *lmterrors.Error {
	return build("Driver already has an active shipment.", InvalidDriverHasActiveShipmentsCode, _codes.FailedPrecondition, _http.StatusBadRequest, details)
}

// NewInvalidShipmentCarLicensePlateError builds an empty car license plate
// error (InvalidArgument).
func NewInvalidShipmentCarLicensePlateError(details string) *lmterrors.Error {
	return build("Car license plate is invalid.", InvalidShipmentCarLicensePlateCode, _codes.InvalidArgument, _http.StatusBadRequest, details)
}
