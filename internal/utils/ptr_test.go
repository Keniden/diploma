package utils

import (
	"testing"
)

func TestPtr(t *testing.T) {
	v := Ptr(42)
	if v == nil || *v != 42 {
		t.Fatalf("Ptr = %v, want 42", v)
	}
}

func TestRef(t *testing.T) {
	v := Ref("x")
	if v == nil || *v != "x" {
		t.Fatalf("Ref = %v, want x", v)
	}
}

func TestUnref(t *testing.T) {
	if got := Unref(Ptr(7)); got != 7 {
		t.Fatalf("Unref = %d, want 7", got)
	}
	if got := Unref[int](nil); got != 0 {
		t.Fatalf("Unref(nil) = %d, want 0", got)
	}
}
