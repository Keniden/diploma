package entities

import (
	"slices"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

// Shipment is aggregate
type Shipment struct {
	ID           values.ShipmentID
	ShortID      values.ShipmentShortID
	Status       values.ShipmentStatus
	DriverID     values.DriverID
	SupplierID   values.SupplierID
	LicensePlate values.LicensePlate
}

func NewShipment() *Shipment {
	id := values.NewID()
	return &Shipment{
		ID:      id,
		ShortID: values.NewShipmentShortID(),
		Status:  values.ShipmentStatusNew,
	}
}

func (s *Shipment) AssignDriver(driverID values.DriverID, supplierID values.SupplierID) error {
	if err := s.setStatus(values.ShipmentStatusAssigned); err != nil {
		return err
	}

	s.DriverID = driverID
	s.SupplierID = supplierID

	return nil
}

func (s *Shipment) SetLicensePLate(licensePlate values.LicensePlate) error {
	s.LicensePlate = values.NewLicensePlate(string(licensePlate))

	return nil
}

func (s *Shipment) Loading() error {
	return s.setStatus(values.ShipmentStatusLoading)
}

func (s *Shipment) ReserveLoading() error {
	return s.setStatus(values.ShipmentStatusReserveLoading)
}

func (s *Shipment) Start() error {
	return s.setStatus(values.ShipmentStatusInProgress)
}

func (s *Shipment) ReserveUnloading() error {
	return s.setStatus(values.ShipmentStatusReserveUnloading)
}

func (s *Shipment) Unloading() error {
	return s.setStatus(values.ShipmentStatusUnloading)
}

// RevertReserveLoading returns the shipment back to ASSIGNED when the loading
// gate reserve is canceled before it is confirmed.
func (s *Shipment) RevertReserveLoading() error {
	return s.setStatus(values.ShipmentStatusAssigned)
}

// RevertReserveUnloading returns the shipment back to IN_PROGRESS when the
// unloading gate reserve is canceled before it is confirmed.
func (s *Shipment) RevertReserveUnloading() error {
	return s.setStatus(values.ShipmentStatusInProgress)
}

func (s *Shipment) Complete() error {
	return s.setStatus(values.ShipmentStatusCompleted)
}

func (s *Shipment) Cancel() error {
	return s.setStatus(values.ShipmentStatusCanceled)
}

func (s *Shipment) IsAssigned() bool {
	return s.Status == values.ShipmentStatusAssigned
}

func (s *Shipment) IsReserveLoading() bool {
	return s.Status == values.ShipmentStatusReserveLoading
}

func (s *Shipment) IsInProgress() bool {
	return s.Status == values.ShipmentStatusInProgress
}

func (s *Shipment) IsReserveUnloading() bool {
	return s.Status == values.ShipmentStatusReserveUnloading
}

func (s *Shipment) IsUnloading() bool {
	return s.Status == values.ShipmentStatusUnloading
}

var shipmentStatusModel = map[values.ShipmentStatus][]values.ShipmentStatus{
	values.ShipmentStatusNew: {
		values.ShipmentStatusAssigned,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusAssigned: {
		values.ShipmentStatusReserveLoading,
		values.ShipmentStatusLoading,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusReserveLoading: {
		values.ShipmentStatusLoading,
		values.ShipmentStatusAssigned,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusLoading: {
		values.ShipmentStatusLoading,
		values.ShipmentStatusInProgress,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusInProgress: {
		values.ShipmentStatusReserveUnloading,
		values.ShipmentStatusUnloading,
		values.ShipmentStatusCompleted,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusReserveUnloading: {
		values.ShipmentStatusUnloading,
		values.ShipmentStatusInProgress,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusUnloading: {
		values.ShipmentStatusUnloading,
		values.ShipmentStatusCompleted,
		values.ShipmentStatusCanceled,
	},
	values.ShipmentStatusCompleted:   {},
	values.ShipmentStatusCanceled:    {},
	values.ShipmentStatusUnspecified: {},
}

func (s *Shipment) canTransition(next values.ShipmentStatus) bool {
	return slices.Contains(shipmentStatusModel[s.Status], next)
}

func (s *Shipment) setStatus(next values.ShipmentStatus) error {
	if !s.canTransition(next) {
		return lmterrors.NewInvalidShipmentStatusTransitionError("shipment status transition is prohibited").WithAttributes(
			lmterrors.Attributes{
				"actual":   s.Status.ToString(),
				"expected": next.ToString(),
			},
		)
	}

	s.Status = next

	return nil
}
