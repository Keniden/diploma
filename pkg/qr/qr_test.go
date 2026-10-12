package qr

import (
	"testing"
	"time"
)

var testSecret = []byte("test-handshake-secret")

func TestCreateSessionRoundTrip(t *testing.T) {
	in := Session{
		ShipmentID:   "11111111-1111-1111-1111-111111111111",
		RoutePointID: "22222222-2222-2222-2222-222222222222",
		TaskID:       "task-42",
		ExpiresAt:    time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	token, err := CreateSession(in, testSecret)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	out, err := DecodeQR(token, testSecret)
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}

	if out.ShipmentID != in.ShipmentID || out.RoutePointID != in.RoutePointID || out.TaskID != in.TaskID {
		t.Fatalf("round trip mismatch: got %+v, want %+v", out, in)
	}
	if !out.ExpiresAt.Equal(in.ExpiresAt) {
		t.Fatalf("expires_at mismatch: got %v, want %v", out.ExpiresAt, in.ExpiresAt)
	}
}

func TestCreateSessionEmptyTaskProducesValidToken(t *testing.T) {
	in := Session{
		ShipmentID:   "11111111-1111-1111-1111-111111111111",
		RoutePointID: "22222222-2222-2222-2222-222222222222",
	}

	token, err := CreateSession(in, testSecret)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	out, err := DecodeQR(token, testSecret)
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}
	if out.TaskID != "" {
		t.Fatalf("expected empty task id, got %q", out.TaskID)
	}
}

func TestDecodeQR_WrongSecretFails(t *testing.T) {
	in := Session{
		ShipmentID:   "11111111-1111-1111-1111-111111111111",
		RoutePointID: "22222222-2222-2222-2222-222222222222",
		TaskID:       "task-7",
	}

	token, err := CreateSession(in, testSecret)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if _, err := DecodeQR(token, []byte("wrong-secret")); err == nil {
		t.Fatal("expected error for a mismatched secret")
	}
}

func TestDecodeQR_MalformedTokenFails(t *testing.T) {
	cases := []string{
		"",
		"not-a-valid-token",
		"only-onepart",
		"@@.@@",
	}

	for _, token := range cases {
		if _, err := DecodeQR(token, testSecret); err == nil {
			t.Fatalf("expected error for malformed token %q", token)
		}
	}
}