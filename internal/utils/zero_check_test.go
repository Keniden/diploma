package utils

import "testing"

func TestIsZero(t *testing.T) {
	if !IsZero(0) {
		t.Fatal("IsZero(0): expected true")
	}
	if IsZero(1) {
		t.Fatal("IsZero(1): expected false")
	}
	if !IsZero("") {
		t.Fatal(`IsZero(""): expected true`)
	}
	if IsZero("x") {
		t.Fatal(`IsZero("x"): expected false`)
	}
}
