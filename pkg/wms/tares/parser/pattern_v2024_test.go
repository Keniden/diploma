package parser

import (
	"strconv"
	"strings"
	"testing"
)

// testEncodeV2024 encodes a value into the same char list used by Decode.
func testEncodeV2024(value uint64, minLen int) string {
	const encodeCharList = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/#"

	// Decode reads from the right: the last char is the least significant digit.
	var leastFirst []byte
	for value > 0 {
		leastFirst = append(leastFirst, encodeCharList[value%uint64(len(encodeCharList))])
		value /= uint64(len(encodeCharList))
	}

	for len(leastFirst) < minLen {
		leastFirst = append(leastFirst, encodeCharList[0])
	}

	// Reverse to most-significant-first.
	runes := make([]byte, len(leastFirst))
	for i := range leastFirst {
		runes[len(leastFirst)-1-i] = leastFirst[i]
	}

	return string(runes)
}

func isUppercaseSafe(s string) bool {
	for _, r := range s {
		// PrepareSticker uppercases the whole sticker, so lowercase
		// checksum chars would be corrupted by Parse.
		if r >= 'a' && r <= 'z' {
			return false
		}
	}

	return true
}

// makeValidV2024Sticker builds a sticker whose checksum survives the
// upper-casing done by PrepareSticker.
func makeValidV2024Sticker(t *testing.T) string {
	t.Helper()

	for i := 1000000; ; i++ {
		barcode := strconv.Itoa(i)
		checksum := testEncodeV2024(
			(uint64((&PatternV2024{}).Crc16Ccitt(barcode))<<2)|uint64(len(barcode)%4),
			3,
		)

		if !isUppercaseSafe(checksum) {
			continue
		}

		return strings.Join([]string{"F1", "F2", "F3", "F4", barcode + checksum, "TBX", "F7"}, ";")
	}
}

func TestV2024ValidSticker(t *testing.T) {
	sticker := makeValidV2024Sticker(t)

	parts := strings.Split(sticker, ";")
	barcode := parts[4][:len(parts[4])-3]

	data, err := Parse(sticker)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expectedID, err := (&PatternV2024{}).Decode(barcode)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if data.TareType != TareTypeTBX || uint64(data.TareID) != expectedID {
		t.Fatalf("expected TBX/%d, got %s/%d", expectedID, data.TareType, data.TareID)
	}
}

func TestV2024WrongChecksumRejected(t *testing.T) {
	sticker := strings.Join([]string{"F1", "F2", "F3", "F4", "AAABBBCCC11AAA", "TBX", "F7"}, ";")

	if _, err := Parse(sticker); err == nil {
		t.Fatal("sticker with invalid checksum must not parse")
	}
}

func TestV2024ParseTareType(t *testing.T) {
	p := &PatternV2024{}

	tt, ok := p.ParseTareType("TEX")
	if !ok || tt != TareTypeTEX {
		t.Fatalf("expected TEX, got %s/%v", tt, ok)
	}

	if _, ok := p.ParseTareType("NOPE"); ok {
		t.Fatal("unknown tare type must not match")
	}
}

func TestV2024ParseTareID(t *testing.T) {
	p := &PatternV2024{}

	if _, err := p.ParseTareID("12"); err == nil {
		t.Fatal("too short part must fail")
	}

	if _, err := p.ParseTareID("####!!"); err == nil {
		t.Fatal("forbidden symbols must fail")
	}
}

func TestV2024Decode(t *testing.T) {
	p := &PatternV2024{}

	v, err := p.Decode("A")
	if err != nil || v != 0 {
		t.Fatalf("expected A -> 0, got %d/%v", v, err)
	}

	v, err = p.Decode("B")
	if err != nil || v != 1 {
		t.Fatalf("expected B -> 1, got %d/%v", v, err)
	}

	if _, err := p.Decode("!"); err == nil {
		t.Fatal("forbidden symbol must fail")
	}
}
