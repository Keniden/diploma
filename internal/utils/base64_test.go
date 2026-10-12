package utils

import "testing"

func TestEncodeBase64(t *testing.T) {
	if got := EncodeBase64("hello"); got != "aGVsbG8=" {
		t.Fatalf("EncodeBase64(hello) = %q, want aGVsbG8=", got)
	}
}

func TestDecodeBase64(t *testing.T) {
	got, err := DecodeBase64("aGVsbG8=")
	if err != nil {
		t.Fatalf("DecodeBase64: unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("DecodeBase64 = %q, want hello", got)
	}
}

func TestDecodeBase64Invalid(t *testing.T) {
	if _, err := DecodeBase64("!!!not-base64!!!"); err == nil {
		t.Fatal("DecodeBase64: expected error for invalid input")
	}
}
