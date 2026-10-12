package drivergrpc

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/basket"
	"last-mile-task/internal/shipping/queries/driver/showcase"
)

type sbFakeShowcaseStore struct {
	points   []showcase.ShowcasePoint
	routes   []showcase.ShowcaseRoute
	pointErr error
	routeErr error
}

func (f *sbFakeShowcaseStore) ListPoints(context.Context, primitives.Page[int64]) ([]showcase.ShowcasePoint, primitives.PaginationResponse[int32], error) {
	return f.points, primitives.PaginationResponse[int32]{HasMore: false}, f.pointErr
}

func (f *sbFakeShowcaseStore) ListRoutesByPoint(context.Context, primitives.Page[int64], int64) ([]showcase.ShowcaseRoute, primitives.PaginationResponse[int64], error) {
	return f.routes, primitives.PaginationResponse[int64]{HasMore: false}, f.routeErr
}

type sbFakeBasketStore struct {
	items   []basket.Item
	addErr  error
	delErr  error
	cleared bool
}

func (f *sbFakeBasketStore) ListItemsByDriverID(context.Context, uuid.UUID) ([]basket.Item, error) { return f.items, nil }

func (f *sbFakeBasketStore) AddItem(context.Context, uuid.UUID, int64) error { return f.addErr }

func (f *sbFakeBasketStore) DeleteItem(context.Context, uuid.UUID, uuid.UUID) error { return f.delErr }

func (f *sbFakeBasketStore) ClearBasket(context.Context, uuid.UUID) error {
	f.cleared = true
	return nil
}

func sbServer(showcaseStore showcase.ShowcaseStore, basketStore basket.BasketItemsStore) *Server {
	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &fakeDriverProvider{driverID: values.NewID()},
		ShowcaseStore:   showcaseStore,
		BasketStore:     basketStore,
	})
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)
}

func incomingCtx(pairs ...string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(pairs...))
}

func TestSBServer_ListPoints_Success(t *testing.T) {
	srv := sbServer(&sbFakeShowcaseStore{points: []showcase.ShowcasePoint{{ID: 1, Name: "pt"}}}, nil)

	if _, err := srv.ListPoints(incomingCtx(), &emptypb.Empty{}); err != nil {
		t.Fatalf("ListPoints: %v", err)
	}
}

func TestSBServer_ListPoints_BadPageSize(t *testing.T) {
	srv := sbServer(&sbFakeShowcaseStore{}, nil)

	_, err := srv.ListPoints(incomingCtx(metaPageSize, "abc"), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error for a bad page size")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSBServer_ListRoutesByPoint_Success(t *testing.T) {
	srv := sbServer(&sbFakeShowcaseStore{routes: []showcase.ShowcaseRoute{{ID: 101}}}, nil)

	if _, err := srv.ListRoutesByPoint(incomingCtx(metaPointID, "1"), &emptypb.Empty{}); err != nil {
		t.Fatalf("ListRoutesByPoint: %v", err)
	}
}

func TestSBServer_ListRoutesByPoint_MissingPointID(t *testing.T) {
	srv := sbServer(&sbFakeShowcaseStore{}, nil)

	_, err := srv.ListRoutesByPoint(incomingCtx(), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error for a missing point id")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSBServer_ListRoutesByPoint_NotFound(t *testing.T) {
	srv := sbServer(&sbFakeShowcaseStore{routeErr: lmterrors.NewNotFoundError("point not found")}, nil)

	_, err := srv.ListRoutesByPoint(incomingCtx(metaPointID, "999"), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected a NotFound error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestSBServer_ListItems_Success(t *testing.T) {
	srv := sbServer(nil, &sbFakeBasketStore{items: []basket.Item{{ItemID: uuid.New(), RouteID: 101}}})

	if _, err := srv.ListItems(incomingCtx(), &emptypb.Empty{}); err != nil {
		t.Fatalf("ListItems: %v", err)
	}
}

func TestSBServer_AddItem_Success(t *testing.T) {
	srv := sbServer(nil, &sbFakeBasketStore{})

	if _, err := srv.AddItem(incomingCtx(metaRouteID, "101"), &emptypb.Empty{}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
}

func TestSBServer_AddItem_MissingRouteID(t *testing.T) {
	srv := sbServer(nil, &sbFakeBasketStore{})

	_, err := srv.AddItem(incomingCtx(), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error for a missing route id")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSBServer_AddItem_NotFound(t *testing.T) {
	srv := sbServer(nil, &sbFakeBasketStore{addErr: lmterrors.NewNotFoundError("route 999 not found")})

	_, err := srv.AddItem(incomingCtx(metaRouteID, "999"), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected a NotFound error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestSBServer_DeleteItem_Success(t *testing.T) {
	srv := sbServer(nil, &sbFakeBasketStore{})
	itemID := uuid.New().String()

	if _, err := srv.DeleteItem(incomingCtx(metaItemID, itemID), &emptypb.Empty{}); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
}

func TestSBServer_DeleteItem_InvalidItemID(t *testing.T) {
	srv := sbServer(nil, &sbFakeBasketStore{})

	_, err := srv.DeleteItem(incomingCtx(metaItemID, "not-a-uuid"), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error for an invalid item id")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSBServer_ClearBasket_Success(t *testing.T) {
	store := &sbFakeBasketStore{}
	srv := sbServer(nil, store)

	if _, err := srv.ClearBasket(incomingCtx(), &emptypb.Empty{}); err != nil {
		t.Fatalf("ClearBasket: %v", err)
	}
	if !store.cleared {
		t.Fatal("expected the basket store to be cleared")
	}
}