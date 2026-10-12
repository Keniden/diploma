package utils

import (
	"reflect"
	"testing"
)

func TestDivideSlice(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	want := [][]int{{1, 2}, {3, 4}, {5}}

	got := DivideSlice(in, 2)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DivideSlice = %v, want %v", got, want)
	}
}

func TestDivideSliceEmpty(t *testing.T) {
	if got := DivideSlice([]int{}, 2); len(got) != 0 {
		t.Fatalf("DivideSlice(empty) = %v, want empty", got)
	}
}

func TestDivideSliceInvalidSize(t *testing.T) {
	if got := DivideSlice([]int{1, 2}, 0); got != nil {
		t.Fatalf("DivideSlice(size=0) = %v, want nil", got)
	}
}
