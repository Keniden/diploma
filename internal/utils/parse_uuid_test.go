package utils

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseUUID(t *testing.T) {
	raw := uuid.NewString()
	got, err := ParseUUID("  " + raw + "  ")
	if err != nil {
		t.Fatalf("ParseUUID: unexpected error: %v", err)
	}
	if got.String() != raw {
		t.Fatalf("ParseUUID = %s, want %s", got, raw)
	}
}

func TestParseUUIDInvalid(t *testing.T) {
	if _, err := ParseUUID("not-a-uuid"); err == nil {
		t.Fatal("ParseUUID: expected error for invalid input")
	}
}

func TestParseUUIDListEmpty(t *testing.T) {
	got, err := ParseUUIDList(nil)
	if err != nil {
		t.Fatalf("ParseUUIDList: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ParseUUIDList = %v, want empty", got)
	}
}

func TestParseUUIDList(t *testing.T) {
	raw := []string{uuid.NewString(), uuid.NewString()}
	got, err := ParseUUIDList(raw)
	if err != nil {
		t.Fatalf("ParseUUIDList: unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].String() != raw[0] || got[1].String() != raw[1] {
		t.Fatalf("ParseUUIDList = %v, want %v", got, raw)
	}
}

func TestParseUUIDListInvalid(t *testing.T) {
	if _, err := ParseUUIDList([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "parse uuid") {
		t.Fatalf("ParseUUIDList: expected invalid uuid error, got %v", err)
	}
}
