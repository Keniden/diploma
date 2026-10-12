package utils

import (
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	in := []int{1, 2, 3, 4}
	got := Filter(in, func(v int) bool { return v%2 == 0 })
	want := []int{2, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filter = %v, want %v", got, want)
	}
}

func TestFilterOne(t *testing.T) {
	got, ok := FilterOne([]int{1, 2, 3}, func(v int) bool { return v == 2 })
	if !ok || got != 2 {
		t.Fatalf("FilterOne = (%d, %v), want (2, true)", got, ok)
	}

	_, ok = FilterOne([]int{1, 3}, func(v int) bool { return v == 2 })
	if ok {
		t.Fatal("FilterOne: expected not found")
	}
}
