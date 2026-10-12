package utils

import "testing"

func TestLenToUint32Capped(t *testing.T) {
	if got := LenToUint32Capped([]int{1, 2, 3}); got != 3 {
		t.Fatalf("LenToUint32Capped = %d, want 3", got)
	}
}

func TestInt64ToInt32(t *testing.T) {
	if got, err := Int64ToInt32(42); err != nil || got != 42 {
		t.Fatalf("Int64ToInt32(42) = (%d, %v), want (42, nil)", got, err)
	}
	if _, err := Int64ToInt32(1 << 40); err == nil {
		t.Fatal("Int64ToInt32: expected overflow error")
	}
}

func TestInt64ToUint32(t *testing.T) {
	if got, err := Int64ToUint32(42); err != nil || got != 42 {
		t.Fatalf("Int64ToUint32(42) = (%d, %v), want (42, nil)", got, err)
	}
	if _, err := Int64ToUint32(-1); err == nil {
		t.Fatal("Int64ToUint32: expected negative error")
	}
	if _, err := Int64ToUint32(1 << 40); err == nil {
		t.Fatal("Int64ToUint32: expected overflow error")
	}
}

func TestInt64ToUint64(t *testing.T) {
	if got, err := Int64ToUint64(42); err != nil || got != 42 {
		t.Fatalf("Int64ToUint64(42) = (%d, %v), want (42, nil)", got, err)
	}
	if _, err := Int64ToUint64(-1); err == nil {
		t.Fatal("Int64ToUint64: expected negative error")
	}
}

func TestIntToInt32(t *testing.T) {
	if got, err := IntToInt32(7); err != nil || got != 7 {
		t.Fatalf("IntToInt32(7) = (%d, %v), want (7, nil)", got, err)
	}
}

func TestIntToUint32(t *testing.T) {
	if got, err := IntToUint32(7); err != nil || got != 7 {
		t.Fatalf("IntToUint32(7) = (%d, %v), want (7, nil)", got, err)
	}
}
