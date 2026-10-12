package utils

import (
	"errors"
	"reflect"
	"testing"

	"last-mile-task/internal/primitives"
)

func TestGetAll(t *testing.T) {
	pages := [][]string{
		{"a", "b"},
		{"c"},
		{},
	}
	pageNo := 0

	f := func(page primitives.Page[string]) ([]string, primitives.PaginationResponse[string], error) {
		if page.Size != 10 {
			return nil, primitives.PaginationResponse[string]{}, errors.New("unexpected page size")
		}
		items := pages[pageNo]
		next := primitives.Page[string]{}
		if pageNo < len(pages)-1 {
			next = primitives.Page[string]{Size: 10, Token: "token"}
		}
		resp := primitives.PaginationResponse[string]{
			Items:    items,
			HasMore:  pageNo < len(pages)-1,
			NextPage: &next,
		}
		pageNo++
		return items, resp, nil
	}

	got, err := GetAll(f, primitives.Page[string]{Size: 10})
	if err != nil {
		t.Fatalf("GetAll: unexpected error: %v", err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAll = %v, want %v", got, want)
	}
}

func TestGetAllStopsOnEmptyPage(t *testing.T) {
	calls := 0
	f := func(page primitives.Page[string]) ([]string, primitives.PaginationResponse[string], error) {
		calls++
		return nil, primitives.PaginationResponse[string]{HasMore: true}, nil
	}

	got, err := GetAll(f, primitives.Page[string]{Size: 5})
	if err != nil {
		t.Fatalf("GetAll: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetAll = %v, want empty", got)
	}
	if calls != 1 {
		t.Fatalf("GetAll: expected 1 call, got %d", calls)
	}
}

func TestGetAllPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	f := func(page primitives.Page[string]) ([]string, primitives.PaginationResponse[string], error) {
		return nil, primitives.PaginationResponse[string]{}, boom
	}

	if _, err := GetAll(f, primitives.Page[string]{Size: 5}); !errors.Is(err, boom) {
		t.Fatalf("GetAll: expected boom, got %v", err)
	}
}
