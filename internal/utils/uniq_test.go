package utils

import (
	"reflect"
	"testing"
)

func TestUniq(t *testing.T) {
	got := Uniq([]int{1, 2, 2, 3, 1})
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Uniq = %v, want %v", got, want)
	}
}

func TestUniqEmpty(t *testing.T) {
	var in []int
	got := Uniq(in)
	if got != nil {
		t.Fatalf("Uniq(empty) = %v, want nil", got)
	}
}
