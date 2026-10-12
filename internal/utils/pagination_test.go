package utils

import (
	"reflect"
	"testing"
)

func TestGetHasNextAndTrim(t *testing.T) {
	got, hasNext := GetHasNextAndTrim([]int{1, 2, 3, 4, 5}, 3)
	if !hasNext {
		t.Fatal("GetHasNextAndTrim: expected hasNext")
	}
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetHasNextAndTrim = %v, want %v", got, want)
	}
}

func TestGetHasNextAndTrimNoTrim(t *testing.T) {
	got, hasNext := GetHasNextAndTrim([]int{1, 2}, 3)
	if hasNext {
		t.Fatal("GetHasNextAndTrim: expected no next")
	}
	if !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("GetHasNextAndTrim = %v, want [1 2]", got)
	}
}
