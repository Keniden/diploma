package basketpg_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
	basketpg "last-mile-task/internal/shipping/queries/driver/basket/basketpg"
	"last-mile-task/internal/shipping/queries/driver/showcase/showcasepg"
)

// newStore builds the basket store over a migrated test database and the
// deterministic showcase route provider stub.
func newStore(t *testing.T) *basketpg.Store {
	t.Helper()

	repo := postgrestest.Repo(t)
	postgrestest.CleanTables(t, repo.Pool)

	return basketpg.New(repo, showcasepg.NewRouteProvider(nil))
}

func TestBasketStore_AddItemAndList(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	driver := uuid.New()

	if err := s.AddItem(ctx, driver, 101); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	items, err := s.ListItemsByDriverID(ctx, driver)
	if err != nil {
		t.Fatalf("ListItemsByDriverID: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 basket item, got %d", len(items))
	}
	item := items[0]
	if item.RouteID != 101 {
		t.Fatalf("expected route 101, got %d", item.RouteID)
	}
	if item.RouteName == "" {
		t.Fatal("expected route name resolved from the provider")
	}
	if item.OfficeID == 0 {
		t.Fatal("expected office id resolved from the provider")
	}
	if item.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be set")
	}
}

func TestBasketStore_AddItem_Idempotent(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	driver := uuid.New()

	if err := s.AddItem(ctx, driver, 101); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := s.AddItem(ctx, driver, 101); err != nil {
		t.Fatalf("AddItem (duplicate): %v", err)
	}

	items, err := s.ListItemsByDriverID(ctx, driver)
	if err != nil {
		t.Fatalf("ListItemsByDriverID: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected no duplicate, got %d items", len(items))
	}
}

func TestBasketStore_AddItem_UnknownRouteNotFound(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	driver := uuid.New()

	err := s.AddItem(ctx, driver, 999)
	if err == nil {
		t.Fatal("expected a NotFound error for an unknown route")
	}
	if !lmterrors.Is(err, "resource.not_found") {
		t.Fatalf("expected resource.not_found, got %v", err)
	}

	items, listErr := s.ListItemsByDriverID(ctx, driver)
	if listErr != nil {
		t.Fatalf("ListItemsByDriverID: %v", listErr)
	}
	if len(items) != 0 {
		t.Fatalf("expected the basket to be unchanged, got %d items", len(items))
	}
}

func TestBasketStore_DeleteItem_Own(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	driver := uuid.New()

	if err := s.AddItem(ctx, driver, 102); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	items, err := s.ListItemsByDriverID(ctx, driver)
	if err != nil {
		t.Fatalf("ListItemsByDriverID: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if err := s.DeleteItem(ctx, driver, items[0].ItemID); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	after, err := s.ListItemsByDriverID(ctx, driver)
	if err != nil {
		t.Fatalf("ListItemsByDriverID: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected an empty basket after delete, got %d", len(after))
	}
}

func TestBasketStore_DeleteItem_ForeignNotFound(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	driver := uuid.New()
	other := uuid.New()

	// An item belongs to another driver.
	if err := s.AddItem(ctx, other, 101); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	otherItems, err := s.ListItemsByDriverID(ctx, other)
	if err != nil {
		t.Fatalf("ListItemsByDriverID: %v", err)
	}

	err = s.DeleteItem(ctx, driver, otherItems[0].ItemID)
	if err == nil {
		t.Fatal("expected a NotFound error when deleting a foreign item")
	}
	if !lmterrors.Is(err, "resource.not_found") {
		t.Fatalf("expected resource.not_found, got %v", err)
	}

	// A completely unknown item id also maps to NotFound.
	missingErr := s.DeleteItem(ctx, driver, uuid.New())
	if missingErr == nil {
		t.Fatal("expected a NotFound error for a missing item")
	}
}

func TestBasketStore_ClearBasket(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	driver := uuid.New()

	if err := s.AddItem(ctx, driver, 101); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := s.AddItem(ctx, driver, 102); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := s.ClearBasket(ctx, driver); err != nil {
		t.Fatalf("ClearBasket: %v", err)
	}
	items, err := s.ListItemsByDriverID(ctx, driver)
	if err != nil {
		t.Fatalf("ListItemsByDriverID: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected an empty basket after clear, got %d", len(items))
	}

	// Clearing an already empty basket is not an error.
	if err := s.ClearBasket(ctx, driver); err != nil {
		t.Fatalf("ClearBasket (again): %v", err)
	}
}