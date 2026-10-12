package warehouse

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/tares"
)

// Store resolves the shipment and route-point data that WMS payloads need.
//
// The real repository implements these queries over *database.Repository
// (lmt.shipment, lmt.route_point + lmt.package_route_point); the skeleton has
// no database layer yet, so the publisher is written against narrow
// interfaces until the adapters are ported. TODO(LASTMILE-052).
type Store interface {
	ShipmentByID(ctx context.Context, shipmentID values.ShipmentID) (dto.Shipment, error)
	PackageDestinationRoutePoint(ctx context.Context, packageID values.PackageID) (dto.RoutePoint, error)
}

// Office is the minimal office data used to build a tare payload.
// It mirrors office_client_v1.Office (ID, CountryID); the skeleton's office
// client is not ported yet. TODO(LASTMILE-052): use office_client_v1.Office.
type Office struct {
	ID        int32
	CountryID int64
}

// Offices mirrors office_client_v1.Client.GetOfficeByPointID.
// TODO(LASTMILE-052): switch to the real client once ported.
type Offices interface {
	GetOfficeByPointID(ctx context.Context, pointID int64) (Office, error)
}

// User is the minimal driver data used to build a tare payload.
// It mirrors users.User (FirstName, MiddleName, LastName, Phone); the
// skeleton's users client is not ported yet. TODO(LASTMILE-052): use
// users.User.
type User struct {
	FirstName  string
	MiddleName string
	LastName   string
	Phone      string
}

// Users mirrors users.Client.UserByID.
// TODO(LASTMILE-052): switch to the real client once ported.
type Users interface {
	UserByID(ctx context.Context, id values.DriverID) (*User, error)
}

// Publisher turns package scan records into WMS outbox messages
// (wms.tares.load / wms.tares.unload).
type Publisher struct {
	store   Store
	writer  *wms_outbox.Writer
	offices Offices
	users   Users
}

func NewPublisher(
	store Store,
	writer *wms_outbox.Writer,
	offices Offices,
	users Users,
) *Publisher {
	return &Publisher{
		store:   store,
		writer:  writer,
		offices: offices,
		users:   users,
	}
}

func (p *Publisher) Publish(
	ctx context.Context,
	shipmentID uuid.UUID,
	routePoint dto.RoutePoint,
	pkgs []dto.Package,
) error {
	shipment, err := p.store.ShipmentByID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("get shipment by id: %w", err)
	}

	ts := primitives.Now()

	office, err := p.offices.GetOfficeByPointID(ctx, int64(routePoint.PointID))
	if err != nil {
		return fmt.Errorf("get office by point id: %w", err)
	}

	user, err := p.users.UserByID(ctx, shipment.DriverID)
	if err != nil {
		return fmt.Errorf("get user by id: %w", err)
	}

	messageShipment := wms.Shipment{
		Driver: wms.Driver{
			Name:  values.NewDriverName(user.FirstName, user.MiddleName, user.LastName).FullName(),
			Phone: user.Phone,
		},
	}

	loadTares := make([]tares.Tare, 0, len(pkgs))
	unloadTares := make([]tares.Tare, 0, len(pkgs))

	for _, pkg := range pkgs {
		tare := tares.Tare{
			Sticker: tares.NewSticker(pkg.TareSticker),

			OfficeID:  office.ID,
			CountryID: office.CountryID,

			Timestamp: ts,
		}

		switch pkg.ActionType {
		case values.RoutePointActionTypeLoad:
			destinationRoutePoint, rpError := p.store.PackageDestinationRoutePoint(ctx, pkg.ID)
			if rpError != nil {
				return fmt.Errorf("get package destination route point: %w", rpError)
			}

			destinationOffice, officeError := p.offices.GetOfficeByPointID(ctx, int64(destinationRoutePoint.PointID))
			if officeError != nil {
				return fmt.Errorf("get destination office by point id: %w", officeError)
			}

			tare.DestinationOfficeID = destinationOffice.ID

			loadTares = append(loadTares, tare)
		case values.RoutePointActionTypeUnload:
			unloadTares = append(unloadTares, tare)
		}
	}

	if len(loadTares) > 0 {
		if err = p.writer.LoadTares(ctx, messageShipment, loadTares); err != nil {
			return fmt.Errorf("load tares: %w", err)
		}
	}

	if len(unloadTares) > 0 {
		if err = p.writer.UnloadTares(ctx, messageShipment, unloadTares); err != nil {
			return fmt.Errorf("unload tares: %w", err)
		}
	}

	return nil
}
