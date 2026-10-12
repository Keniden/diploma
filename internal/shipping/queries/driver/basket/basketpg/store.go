package basketpg

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"

	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/queries/driver/basket"
	"last-mile-task/internal/shipping/queries/driver/showcase/showcasepg"
)

// basketItemNamespace derivation namespace for the deterministic basket item id
// (idempotent per route: the same route always maps to the same item_id).
var basketItemNamespace = uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")

// RouteProvider resolves the display data (name, office) of a basket route.
//
// The concrete implementation is the deterministic showcase stub
// (showcasepg.NewRouteProvider), shared with the showcase adapter at wiring.
type RouteProvider interface {
	GetRouteByID(ctx context.Context, id int64) (showcasepg.Route, error)
}

// Store implements [basket.BasketItemsStore] over lmt.basket.
type Store struct {
	*database.Repository
	routes RouteProvider
}

// New builds the basket store over the repository and the route provider stub.
func New(repo *database.Repository, routes RouteProvider) *Store {
	return &Store{
		Repository: repo,
		routes:     routes,
	}
}

type basketRecord struct {
	ItemID     uuid.UUID `db:"item_id"`
	RoutecarID int64     `db:"routecar_id"`
	CreatedAt  time.Time `db:"created_at"`
}

// ListItemsByDriverID returns the driver's basket items enriched with route
// display data resolved through the route provider stub.
func (s *Store) ListItemsByDriverID(ctx context.Context, driverID uuid.UUID) ([]basket.Item, error) {
	query, args, err := s.Builder.
		Select("item_id", "routecar_id", "created_at").
		From("lmt.basket").
		Where("driver_id = ?", driverID).
		OrderBy("created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list basket items query: %w", err)
	}

	var records []basketRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("list basket items: %w", err)
	}

	items := make([]basket.Item, 0, len(records))
	for _, rec := range records {
		route, routeErr := s.routes.GetRouteByID(ctx, rec.RoutecarID)
		if routeErr != nil {
			return nil, fmt.Errorf("resolve route %d: %w", rec.RoutecarID, routeErr)
		}
		items = append(items, basket.Item{
			ItemID:    rec.ItemID,
			RouteID:   rec.RoutecarID,
			RouteName: route.Name,
			CreatedAt: rec.CreatedAt,
			OfficeID:  route.OfficeID,
		})
	}

	return items, nil
}

// AddItem adds a route to the basket idempotently: an unknown route maps to
// NotFound and a duplicate is a no-op (ON CONFLICT DO NOTHING).
func (s *Store) AddItem(ctx context.Context, driverID uuid.UUID, routeID int64) error {
	if _, err := s.routes.GetRouteByID(ctx, routeID); err != nil {
		return lmterrors.NewNotFoundError(fmt.Sprintf("route %d not found", routeID)).Wrap(err)
	}

	itemID := itemIDForRoute(routeID)

	query, args, err := s.Builder.
		Insert("lmt.basket").
		Columns("driver_id", "item_id", "routecar_id").
		Values(driverID, itemID, routeID).
		Suffix("ON CONFLICT (driver_id, item_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build add basket item query: %w", err)
	}

	if _, err = s.Querier(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("add basket item: %w", err)
	}

	return nil
}

// DeleteItem removes an item from the driver's basket; a foreign or missing
// item maps to NotFound without any change.
func (s *Store) DeleteItem(ctx context.Context, driverID, itemID uuid.UUID) error {
	query, args, err := s.Builder.
		Delete("lmt.basket").
		Where("driver_id = ?", driverID).
		Where("item_id = ?", itemID).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete basket item query: %w", err)
	}

	tag, err := s.Querier(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete basket item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return lmterrors.NewNotFoundError(fmt.Sprintf("basket item %s not found", itemID))
	}

	return nil
}

// ClearBasket removes every item of the driver's basket.
func (s *Store) ClearBasket(ctx context.Context, driverID uuid.UUID) error {
	query, args, err := s.Builder.
		Delete("lmt.basket").
		Where("driver_id = ?", driverID).
		ToSql()
	if err != nil {
		return fmt.Errorf("build clear basket query: %w", err)
	}

	if _, err = s.Querier(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("clear basket: %w", err)
	}

	return nil
}

// itemIDForRoute derives a deterministic item_id from the route id so that
// re-adding the same route is idempotent (same (driver_id, item_id) row).
func itemIDForRoute(routeID int64) uuid.UUID {
	return uuid.NewSHA1(basketItemNamespace, []byte(strconv.FormatInt(routeID, 10)))
}