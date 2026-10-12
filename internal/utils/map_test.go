package utils

import (
	"errors"
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(v int) string { return string(rune('a' + v - 1)) })
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Map = %v, want %v", got, want)
	}
}

func TestMapErr(t *testing.T) {
	got, err := MapErr([]int{1, 2}, func(v int) (int, error) {
		if v == 2 {
			return 0, errors.New("boom")
		}
		return v, nil
	})
	if err == nil {
		t.Fatal("MapErr: expected error")
	}
	if got != nil {
		t.Fatalf("MapErr result = %v, want nil", got)
	}
}

func TestMapSkip(t *testing.T) {
	got := MapSkip([]int{1, 2, 3}, func(v int) (int, error) {
		if v == 2 {
			return 0, errors.New("skip")
		}
		return v, nil
	})
	want := []int{1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MapSkip = %v, want %v", got, want)
	}
}

func TestMapUnique(t *testing.T) {
	type pair struct{ key string }
	in := []pair{{key: "a"}, {key: "b"}, {key: "a"}}
	got := MapUnique(in, func(p pair) string { return p.key })
	want := []string{"a", "b"}
	if len(got) != len(want) || !reflect.DeepEqual(got, want) {
		t.Fatalf("MapUnique = %v, want %v", got, want)
	}
}
