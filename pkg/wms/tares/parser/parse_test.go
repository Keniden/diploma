package parser

import (
	"strings"
	"testing"
)

func TestParseTBX(t *testing.T) {
	for _, sticker := range []string{"TRBX12345", "ЕКИЧ42", "trbx999"} {
		data, err := Parse(sticker)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", sticker, err)
		}

		if data.TareType != TareTypeTBX {
			t.Fatalf("Parse(%q): expected TBX, got %q", sticker, data.TareType)
		}

		if data.TareID <= 0 {
			t.Fatalf("Parse(%q): expected positive TareID, got %d", sticker, data.TareID)
		}
	}
}

func TestParsePLT(t *testing.T) {
	data, err := Parse("WHPT777")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if data.TareType != TareTypePLT || data.TareID != 777 {
		t.Fatalf("expected PLT/777, got %s/%d", data.TareType, data.TareID)
	}
}

func TestParseDollarColonTBX(t *testing.T) {
	// "$:....:...." -> parts[2] is the id for ':' and ';' separators.
	for _, sticker := range []string{"$prefix:one:100:two", "$p;one;100;two"} {
		data, err := Parse(sticker)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", sticker, err)
		}

		if data.TareType != TareTypeTBX || data.TareID != 100 {
			t.Fatalf("Parse(%q): expected TBX/100, got %s/%d", sticker, data.TareType, data.TareID)
		}
	}
}

func TestParsePercentAndStarTBX(t *testing.T) {
	for _, sticker := range []string{"%p:one:101:two", "*p;one;102;two"} {
		data, err := Parse(sticker)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", sticker, err)
		}

		if data.TareType != TareTypeTBX {
			t.Fatalf("Parse(%q): expected TBX, got %q", sticker, data.TareType)
		}
	}
}

func TestParseSemicolonZHTBX(t *testing.T) {
	data, err := Parse(";Ж10Ж2Ж3")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Split by "Ж": [";", "10", "2", "3"] - id is parts[2].
	if data.TareType != TareTypeTBX || data.TareID != 2 {
		t.Fatalf("expected TBX/2, got %s/%d", data.TareType, data.TareID)
	}
}

func TestParseNumericTBX(t *testing.T) {
	data, err := Parse("902056546")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if data.TareType != TareTypeTBX || data.TareID != 902056546 {
		t.Fatalf("expected TBX/902056546, got %s/%d", data.TareType, data.TareID)
	}

	if _, err := Parse("902056545"); err == nil {
		t.Fatal("902056545 must not match TBX threshold")
	}
}

func TestParseCON(t *testing.T) {
	data, err := Parse("WCT1234500")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if data.TareType != TareTypeCON || data.TareID != 0 {
		t.Fatalf("expected CON/0, got %s/%d", data.TareType, data.TareID)
	}

	if _, err := Parse("WCTnot-a-number"); err == nil {
		t.Fatal("WCT with non-numeric suffix must not match")
	}
}

func TestParseMPB(t *testing.T) {
	data, err := Parse("WBMP:111:222:333")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if data.TareType != TareTypeMPB || data.TareID != 333 {
		t.Fatalf("expected MPB/333, got %s/%d", data.TareType, data.TareID)
	}

	data, err = Parse("ЦИЬЗЖ111Ж222Ж333")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if data.TareType != TareTypeMPB || data.TareID != 333 {
		t.Fatalf("expected MPB/333, got %s/%d", data.TareType, data.TareID)
	}
}

func TestParseUnknown(t *testing.T) {
	for _, sticker := range []string{"", "ABCXYZ", "WCT", "WBMP:1:2"} {
		if _, err := Parse(sticker); err == nil {
			t.Fatalf("Parse(%q) must fail", sticker)
		}
	}
}

func TestPrepareSticker(t *testing.T) {
	if got := PrepareSticker("  trbx42  "); got != "TRBX42" {
		t.Fatalf("expected TRBX42, got %q", got)
	}

	if got := PrepareSticker("циьзЖ1Ж2Ж3"); !strings.HasPrefix(got, "ЦИЬЗ") {
		t.Fatalf("expected uppercased prefix, got %q", got)
	}
}
