package cargo

import (
	"context"
	"fmt"
	"time"

	cargo_outbox "last-mile-task/internal/platform/outbox/cargo"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/internal/utils"
	"last-mile-task/pkg/outbox"
)

// Publisher turns package scan records into cargo outbox messages
// (cargo.packages.scanned / cargo.packages.status.updated).
type Publisher struct {
	cargoWriter *cargo_outbox.Writer
}

// The real repository constructs the writer from *database.Repository; the
// skeleton has no database layer yet. TODO(LASTMILE-057): accept
// *database.Repository once it is ported.
func NewPublisher(querier outbox.Querier) *Publisher {
	return &Publisher{
		cargoWriter: cargo_outbox.NewWriter(querier),
	}
}

func (p *Publisher) Publish(
	ctx context.Context,
	routePoint dto.RoutePoint,
	pkgs []dto.Package,
) error {
	unloadPkgs := utils.Filter(pkgs, func(pkg dto.Package) bool {
		return pkg.ActionType == values.RoutePointActionTypeUnload
	})
	if len(unloadPkgs) == 0 {
		return nil
	}

	message := cargo_outbox.ScannedPackagesMessage{
		Packages: utils.Map(unloadPkgs, func(pkg dto.Package) cargo_outbox.UpdatePackagesStatusPackage {
			return cargo_outbox.UpdatePackagesStatusPackage{
				ID:               pkg.ID,
				Action:           cargo_outbox.UpdatePackagesStatusActionTypeUnload,
				CurrentPointUUID: routePoint.PointUUID,
				Dt:               pkg.ScannedAt,
			}
		}),
	}

	if err := p.cargoWriter.WriteScannedPackages(ctx, message); err != nil {
		return fmt.Errorf("writer: %w", err)
	}

	return nil
}

func (p *Publisher) PublishInTransitStatus(
	ctx context.Context,
	routePoint dto.RoutePoint,
	packageIDs []values.PackageID,
	transitionedAt time.Time,
) error {
	if len(packageIDs) == 0 {
		return nil
	}

	message := cargo_outbox.PackagesStatusMessage{
		Packages: utils.Map(packageIDs, func(id values.PackageID) cargo_outbox.UpdatePackagesStatusPackage {
			return cargo_outbox.UpdatePackagesStatusPackage{
				ID:               id,
				Action:           cargo_outbox.UpdatePackagesStatusActionTypeDepart,
				CurrentPointUUID: routePoint.PointUUID,
				Dt:               transitionedAt,
			}
		}),
	}

	if err := p.cargoWriter.WritePackagesStatus(ctx, message); err != nil {
		return fmt.Errorf("writer: %w", err)
	}

	return nil
}
