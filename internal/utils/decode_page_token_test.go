package utils

import "testing"

func TestDecodePageTokenEmpty(t *testing.T) {
	got, err := DecodePageToken("")
	if err != nil {
		t.Fatalf("DecodePageToken: unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("DecodePageToken = %q, want empty", got)
	}
}

func TestDecodePageToken(t *testing.T) {
	got, err := DecodePageToken(EncodeBase64("abc:123"))
	if err != nil {
		t.Fatalf("DecodePageToken: unexpected error: %v", err)
	}
	if got != "abc:123" {
		t.Fatalf("DecodePageToken = %q, want abc:123", got)
	}
}

func TestDecodePageTokenInvalid(t *testing.T) {
	if _, err := DecodePageToken("###"); err == nil {
		t.Fatal("DecodePageToken: expected error for invalid token")
	}
}
