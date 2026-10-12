package tares

import (
	"testing"
	"time"
)

func TestFormatTimestamp(t *testing.T) {
	ts := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	tare := Tare{
		Timestamp: ts,
	}

	// Moscow is UTC+3, expected: 12:00:00+03:00.
	if got := tare.FormatTimestamp(); got != "2026-10-10T12:00:00+03:00" {
		t.Fatalf("unexpected formatted timestamp: %q", got)
	}
}

func TestNewSticker(t *testing.T) {
	s := NewSticker("TRBX123")

	if s.String() != "TRBX123" {
		t.Fatalf("String mismatch: %q", s.String())
	}

	if s.TareID() != 0 || s.TareType() != "" {
		t.Fatalf("plain sticker must have no parsed fields: %d/%q", s.TareID(), s.TareType())
	}
}

func TestParseSticker(t *testing.T) {
	if _, err := ParseSticker("TBX-9012345678"); err == nil {
		t.Fatal("TBX- prefix must not parse")
	}

	s, err := ParseSticker("TRBX777")
	if err != nil {
		t.Fatalf("ParseSticker failed: %v", err)
	}

	if s.TareType() != "TBX" || s.TareID() != 777 {
		t.Fatalf("expected TBX/777, got %s/%d", s.TareType(), s.TareID())
	}

	if s.String() != "TRBX777" {
		t.Fatalf("raw sticker mismatch: %q", s.String())
	}
}
