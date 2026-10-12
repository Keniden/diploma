package entities

import (
	"slices"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

type RoutePoint struct {
	ID       values.RoutePointID
	PointID  values.PointID
	SeqNum   int32
	Distance int64
	Duration int64
	Status   values.RoutePointStatus
}

func newRoutePointFromTemplate(template values.RoutePointTemplate) *RoutePoint {
	return &RoutePoint{
		ID:       values.NewID(),
		PointID:  template.RoutePointTemplateID,
		SeqNum:   template.SeqNum,
		Distance: template.Distance,
		Status:   values.RoutePointStatusWaitingArrival,
	}
}

var routePointStatusModel = map[values.RoutePointStatus][]values.RoutePointStatus{
	values.RoutePointStatusWaitingArrival: {
		values.RoutePointStatusDriverArrived,
		values.RoutePointStatusCompleted,
	},
	values.RoutePointStatusDriverArrived: {
		values.RoutePointStatusDriverArrived,
		values.RoutePointStatusPackagesNotUnloaded,
		values.RoutePointStatusPackagesNotLoaded,
		values.RoutePointStatusCompleted,
	},
	values.RoutePointStatusPackagesNotUnloaded: {
		values.RoutePointStatusDriverArrived,
	},
	values.RoutePointStatusPackagesNotLoaded: {
		values.RoutePointStatusDriverArrived,
		values.RoutePointStatusCompleted,
	},
	values.RoutePointStatusCompleted: {},
}

func (rp *RoutePoint) canTransition(next values.RoutePointStatus) bool {
	return slices.Contains(routePointStatusModel[rp.Status], next)
}

func (rp *RoutePoint) DriverArrived() error {
	return rp.setStatus(values.RoutePointStatusDriverArrived)
}

func (rp *RoutePoint) PackagesNotUnloaded() error {
	return rp.setStatus(values.RoutePointStatusPackagesNotUnloaded)
}

func (rp *RoutePoint) PackagesNotLoaded() error {
	return rp.setStatus(values.RoutePointStatusPackagesNotLoaded)
}

func (rp *RoutePoint) Completed() error {
	return rp.setStatus(values.RoutePointStatusCompleted)
}

func (rp *RoutePoint) setStatus(next values.RoutePointStatus) error {
	if !rp.canTransition(next) {
		return lmterrors.NewInvalidRoutePointStatusTransitionError("route point status transition is prohibited").WithAttributes(
			lmterrors.Attributes{
				"actual":   rp.Status.ToString(),
				"expected": next.ToString(),
			},
		)
	}
	rp.Status = next
	return nil
}

func (rp *RoutePoint) SetDistance(distance int64) {
	rp.Distance = distance
}

func (rp *RoutePoint) SetDuration(duration int64) {
	rp.Duration = duration
}

func (rp *RoutePoint) SetSequenceNumber(seqNum int32) {
	rp.SeqNum = seqNum
}
