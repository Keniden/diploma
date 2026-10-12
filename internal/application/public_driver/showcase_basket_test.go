package publicdriver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	publicdriver "last-mile-task/internal/application/public_driver"
	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/basket"
	"last-mile-task/internal/shipping/queries/driver/showcase"
)

// fakeDriverProvider resolves a fixed driver for the use-case tests.
type fakeSBDriverProvider struct {
	driver values.Driver
	err    error
}

func (f *fakeSBDriverProvider) GetDriverFromContext(context.Context) (values.Driver, error) {
	if f.err != nil {
		return values.Driver{}, f.err
	}
	return f.driver, nil
}

// fakeBasketStore implements basket.BasketItemsStore for the use-case tests.
type fakeBasketStore struct {
	items   []basket.Item
	addErr  error
	delErr  error
	cleared bool
	added   []int64
	driver  uuid.UUID
}

func (f *fakeBasketStore) ListItemsByDriverID(context.Context, uuid.UUID) ([]basket.Item, error) {
	return f.items, nil
}

func (f *fakeBasketStore) AddItem(_ context.Context, driver uuid.UUID, routeID int64) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.added = append(f.added, routeID)
	f.driver = driver
	return nil
}

func (f *fakeBasketStore) DeleteItem(context.Context, uuid.UUID, uuid.UUID) error {
	return f.delErr
}

func (f *fakeBasketStore) ClearBasket(context.Context, uuid.UUID) error {
	f.cleared = true
	return nil
}

// fakeShowcaseStore implements showcase.ShowcaseStore for the use-case tests.
type fakeShowcaseStore struct {
	points     []showcase.ShowcasePoint
	routes     []showcase.ShowcaseRoute
	pointErr   error
	routeErr   error
	lastPoint  int64
}

func (f *fakeShowcaseStore) ListPoints(context.Context, primitives.Page[int64]) ([]showcase.ShowcasePoint, primitives.PaginationResponse[int32], error) {
	return f.points, primitives.PaginationResponse[int32]{HasMore: false}, f.pointErr
}

func (f *fakeShowcaseStore) ListRoutesByPoint(_ context.Context, _ primitives.Page[int64], pointID int64) ([]showcase.ShowcaseRoute, primitives.PaginationResponse[int64], error) {
	f.lastPoint = pointID
	return f.routes, primitives.PaginationResponse[int64]{HasMore: false}, f.routeErr
}

func TestAddItem_DriverIDResolved(t *testing.T) {
	driverID := uuid.New()
	store := &fakeBasketStore{}
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeSBDriverProvider{driver: values.Driver{ID: driverID}},
		BasketStore:     store,
	})

	require.NoError(t, app.AddItem(context.Background(), 101))
	require.Equal(t, []int64{101}, store.added)
	require.Equal(t, driverID, store.driver)
}

func TestAddItem_MissingDriver(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeSBDriverProvider{err: errors.New("no driver")},
		BasketStore:     &fakeBasketStore{},
	})

	err := app.AddItem(context.Background(), 101)
	require.Error(t, err)
}

func TestAddItem_InvalidRouteID(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeSBDriverProvider{driver: values.Driver{ID: uuid.New()}},
		BasketStore:     &fakeBasketStore{},
	})

	err := app.AddItem(context.Background(), 0)
	require.Error(t, err)
	require.True(t, lmterrors.Is(err, "validation.invalid_argument"))
}

func TestDeleteItem_InvalidItemID(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeSBDriverProvider{driver: values.Driver{ID: uuid.New()}},
		BasketStore:     &fakeBasketStore{},
	})

	require.Error(t, app.DeleteItem(context.Background(), uuid.Nil))
}

func TestListItems_EmptyBasket(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeSBDriverProvider{driver: values.Driver{ID: uuid.New()}},
		BasketStore:     &fakeBasketStore{},
	})

	items, err := app.ListItems(context.Background())
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestClearBasket_TouchesStore(t *testing.T) {
	store := &fakeBasketStore{}
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeSBDriverProvider{driver: values.Driver{ID: uuid.New()}},
		BasketStore:     store,
	})

	require.NoError(t, app.ClearBasket(context.Background()))
	require.True(t, store.cleared)
}

func TestListPoints_BadPageSize(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		ShowcaseStore: &fakeShowcaseStore{},
	})

	_, _, err := app.ListPoints(context.Background(), publicdriver.ListPointsRequest{PageSize: 0})
	require.Error(t, err)
	require.True(t, lmterrors.Is(err, "validation.invalid_argument"))
}

func TestListPoints_EmptyShowcase(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		ShowcaseStore: &fakeShowcaseStore{points: []showcase.ShowcasePoint{}},
	})

	points, _, err := app.ListPoints(context.Background(), publicdriver.ListPointsRequest{PageSize: 10})
	require.NoError(t, err)
	require.Empty(t, points)
}

func TestListRoutesByPoint_UnknownPoint(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		ShowcaseStore: &fakeShowcaseStore{routeErr: lmterrors.NewNotFoundError("point not found")},
	})

	_, _, err := app.ListRoutesByPoint(context.Background(), publicdriver.ListRoutesByPointRequest{PointID: 999, PageSize: 10})
	require.Error(t, err)
	require.True(t, lmterrors.Is(err, "resource.not_found"))
}

func TestListRoutesByPoint_EmptyRoutePage(t *testing.T) {
	app := publicdriver.New(publicdriver.Config{
		ShowcaseStore: &fakeShowcaseStore{routes: []showcase.ShowcaseRoute{}},
	})

	routes, _, err := app.ListRoutesByPoint(context.Background(), publicdriver.ListRoutesByPointRequest{PointID: 1, PageSize: 10})
	require.NoError(t, err)
	require.Empty(t, routes)
}