package application

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

const reserveExpireBatchLimit = 100

// ReserveService orchestrates the gate reserve flow (ARCHITECTURE.md §23 flows
// 4 and 7): creating, approving, releasing, canceling and expiring reserves,
// each aggregate transition inside its own transaction with the corresponding
// domain event.
type ReserveService struct {
	reserveRepo     ports.ReserveRepository
	shipmentRepo    ports.ShipmentRepository
	routeRepo       ports.RouteRepository
	reserveManager  ports.ReserveManager
	shipmentPolicy  *policies.ShipmentPolicy
	reserveEvents   ports.ReserveEventPublisher
	tx              ports.Transactional
}

// NewReserveService builds the reserve service.
func NewReserveService(
	reserveRepo ports.ReserveRepository,
	shipmentRepo ports.ShipmentRepository,
	routeRepo ports.RouteRepository,
	reserveManager ports.ReserveManager,
	shipmentPolicy *policies.ShipmentPolicy,
	reserveEvents ports.ReserveEventPublisher,
	tx ports.Transactional,
) *ReserveService {
	return &ReserveService{
		reserveRepo:    reserveRepo,
		shipmentRepo:   shipmentRepo,
		routeRepo:      routeRepo,
		reserveManager: reserveManager,
		shipmentPolicy: shipmentPolicy,
		reserveEvents:  reserveEvents,
		tx:             tx,
	}
}

// QueueReserveByShipmentID creates a gate reserve for a route point: it checks
// the reservation policy, delegates the gate allocation to ReserveManager,
// stores the reserve and moves the shipment into RESERVE_LOADING (source) or
// RESERVE_UNLOADING (last point) inside one transaction.
func (s *ReserveService) QueueReserveByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) (*entities.Reserve, error) {
	var reserve *entities.Reserve

	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		shipment, err := s.shipmentRepo.GetShipmentByID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		route, err := s.routeRepo.GetRouteByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get route by shipment id: %w", err)
		}

		source, err := route.Source()
		if err != nil {
			return err
		}
		target, err := route.Target()
		if err != nil {
			return err
		}
		targets, err := route.RoutePointsForUnloading()
		if err != nil {
			return err
		}

		isSource := routePointID == source.ID
		isTarget := routePointID == target.ID

		switch {
		case isSource:
			if err := s.shipmentPolicy.CanReserveGate(txCtx, shipmentID, routePointID); err != nil {
				return fmt.Errorf("can reserve gate: %w", err)
			}
		case isTarget:
			if err := s.shipmentPolicy.CanReserveUnloadingGate(txCtx, shipmentID, routePointID); err != nil {
				return fmt.Errorf("can reserve unloading gate: %w", err)
			}
		default:
			return lmterrors.NewInvalidReserveMustBeSourceError("route point must be source or target to reserve a gate")
		}

		reserved, err := s.reserveManager.ReserveOrQueue(txCtx, shipment, source, targets)
		if err != nil {
			return fmt.Errorf("reserve or queue: %w", err)
		}
		reserved.RoutePointID = routePointID

		if err := s.reserveRepo.SaveReserve(txCtx, reserved); err != nil {
			return fmt.Errorf("save reserve: %w", err)
		}

		if isSource {
			if err := shipment.ReserveLoading(); err != nil {
				return fmt.Errorf("reserve loading: %w", err)
			}
		} else {
			if err := shipment.ReserveUnloading(); err != nil {
				return fmt.Errorf("reserve unloading: %w", err)
			}
		}

		if err := s.shipmentRepo.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		if err := s.reserveEvents.PublishReserveEvent(txCtx, events.ReserveQueuedEvent{
			ID:         reserved.ID,
			ShipmentID: shipmentID,
			Driver:     values.Driver{ID: shipment.DriverID},
		}); err != nil {
			return fmt.Errorf("publish reserve queued event: %w", err)
		}

		reserve = reserved
		return nil
	})
	if err != nil {
		return nil, err
	}

	return reserve, nil
}

// ApproveReserveByShipmentID confirms the active reserve of a shipment.
func (s *ReserveService) ApproveReserveByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		reserve, err := s.reserveRepo.GetReserveByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get reserve by shipment id: %w", err)
		}

		return s.approveReserve(txCtx, reserve)
	})
}

// ApproveReserveByRoutePointID confirms the reserve that belongs to the given
// route point. It is the confirmation hook used by the TSD scan flow.
func (s *ReserveService) ApproveReserveByRoutePointID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		reserve, err := s.reserveRepo.GetReserveByRoutePointID(txCtx, routePointID)
		if err != nil {
			return fmt.Errorf("get reserve by route point id: %w", err)
		}

		if !reserve.BelongsToShipment(shipmentID) {
			return lmterrors.NewInvalidReserveCannotApproveError("reserve does not belong to the shipment")
		}

		return s.approveReserve(txCtx, reserve)
	})
}

// approveReserve approves the gate reserve task, confirms the reserve and moves
// the shipment into LOADING / UNLOADING inside the caller's transaction.
func (s *ReserveService) approveReserve(ctx context.Context, reserve *entities.Reserve) error {
	shipment, err := s.shipmentRepo.GetShipmentByID(ctx, reserve.ShipmentID)
	if err != nil {
		return fmt.Errorf("get shipment by id: %w", err)
	}

	isFirstApproval := reserve.Status == values.ReserveStatusWaitingApproval

	// Fake (task-less) reserves have no gate task to approve.
	if !reserve.IsFake() {
		if _, err := s.reserveManager.ApproveReserveTask(ctx, reserve.Task); err != nil {
			return fmt.Errorf("approve reserve task: %w", err)
		}
	}

	if err := reserve.Approve(); err != nil {
		return fmt.Errorf("approve reserve: %w", err)
	}

	if err := s.reserveRepo.SaveReserve(ctx, reserve); err != nil {
		return fmt.Errorf("save reserve: %w", err)
	}

	switch {
	case shipment.IsReserveLoading():
		if err := shipment.Loading(); err != nil {
			return fmt.Errorf("shipment loading: %w", err)
		}
	case shipment.IsReserveUnloading():
		if err := shipment.Unloading(); err != nil {
			return fmt.Errorf("shipment unloading: %w", err)
		}
	default:
		return lmterrors.NewInvalidShipmentMustBeReserveStatusError("shipment must be in reserve status to confirm reserve")
	}

	if err := s.shipmentRepo.SaveShipment(ctx, shipment); err != nil {
		return fmt.Errorf("save shipment: %w", err)
	}

	return s.reserveEvents.PublishReserveEvent(ctx, events.ReserveApprovedEvent{
		ID:              reserve.ID,
		ShipmentID:      reserve.ShipmentID,
		Task:            reserve.Task,
		Timestamp:       values.Now(),
		IsFirstApproval: isFirstApproval,
		IsFake:          reserve.IsFake(),
	})
}

// ReleaseReserveByShipmentID releases the active reserve of a shipment and
// tells the gate service the driver departed from the point.
func (s *ReserveService) ReleaseReserveByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	pointID values.PointID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		reserve, err := s.reserveRepo.GetReserveByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get reserve by shipment id: %w", err)
		}

		if err := s.reserveManager.CancelReserve(txCtx, reserve, pointID); err != nil {
			return fmt.Errorf("cancel gate reserve: %w", err)
		}

		if err := reserve.Release(); err != nil {
			return fmt.Errorf("release reserve: %w", err)
		}

		if err := s.reserveRepo.SaveReserve(txCtx, reserve); err != nil {
			return fmt.Errorf("save reserve: %w", err)
		}

		return s.reserveEvents.PublishReserveEvent(txCtx, events.ReserveReleasedEvent{
			ID:        reserve.ID,
			Task:      reserve.Task,
			Timestamp: values.Now(),
		})
	})
}

// GetReserveByRoutePointID returns the active reserve of a route point.
func (s *ReserveService) GetReserveByRoutePointID(
	ctx context.Context,
	routePointID values.RoutePointID,
) (*entities.Reserve, error) {
	return s.reserveRepo.GetReserveByRoutePointID(ctx, routePointID)
}

// GetReserveByShipmentID returns the active reserve of a shipment.
func (s *ReserveService) GetReserveByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
) (*entities.Reserve, error) {
	return s.reserveRepo.GetReserveByShipmentID(ctx, shipmentID)
}

// CancelReserveByShipmentID cancels a not-yet-confirmed reserve and returns the
// shipment back to its previous stage.
func (s *ReserveService) CancelReserveByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		reserve, err := s.reserveRepo.GetReserveByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get reserve by shipment id: %w", err)
		}

		if err := reserve.Cancel(); err != nil {
			return fmt.Errorf("cancel reserve: %w", err)
		}

		shipment, err := s.shipmentRepo.GetShipmentByID(txCtx, reserve.ShipmentID)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		switch {
		case shipment.IsReserveLoading():
			if err := shipment.RevertReserveLoading(); err != nil {
				return fmt.Errorf("revert reserve loading: %w", err)
			}
		case shipment.IsReserveUnloading():
			if err := shipment.RevertReserveUnloading(); err != nil {
				return fmt.Errorf("revert reserve unloading: %w", err)
			}
		}

		if err := s.reserveRepo.SaveReserve(txCtx, reserve); err != nil {
			return fmt.Errorf("save reserve: %w", err)
		}

		if err := s.shipmentRepo.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		return nil
	})
}

// ExpireDueWaitingApprovalReserves expires due reserves waiting for approval.
func (s *ReserveService) ExpireDueWaitingApprovalReserves(ctx context.Context) error {
	return s.expireDue(ctx, s.reserveRepo.ListExpiredWaitingApprovalReserves)
}

// ExpireDueActiveReserves expires due active reserves.
func (s *ReserveService) ExpireDueActiveReserves(ctx context.Context) error {
	return s.expireDue(ctx, s.reserveRepo.ListExpiredActiveReserves)
}

func (s *ReserveService) expireDue(
	ctx context.Context,
	list func(context.Context, int) ([]*entities.Reserve, error),
) error {
	reserves, err := list(ctx, reserveExpireBatchLimit)
	if err != nil {
		return fmt.Errorf("list due reserves: %w", err)
	}

	for _, reserve := range reserves {
		if err := s.expireOne(ctx, reserve); err != nil {
			return fmt.Errorf("expire reserve %s: %w", reserve.ID, err)
		}
	}

	return nil
}

func (s *ReserveService) expireOne(ctx context.Context, reserve *entities.Reserve) error {
	switch reserve.Status {
	case values.ReserveStatusWaitingApproval:
		if err := reserve.ExpireWaitingApproval(); err != nil {
			return err
		}
	case values.ReserveStatusActive:
		if err := reserve.ExpireActive(); err != nil {
			return err
		}
	default:
		// Already handled status: idempotent skip.
		return nil
	}

	return s.reserveRepo.SaveReserve(ctx, reserve)
}