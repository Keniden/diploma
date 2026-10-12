package wms

import "testing"

type tareStub struct {
	sticker  string
	officeID int32
	country  int64
}

func (t tareStub) GetSticker() string       { return t.sticker }
func (t tareStub) GetSourceOfficeID() int32 { return t.officeID }
func (t tareStub) GetCountryID() int64      { return t.country }

func TestGroupTaresByCountryNumeric(t *testing.T) {
	grouped := GroupTaresByCountry([]tareStub{
		{sticker: "RU1", country: 643},
		{sticker: "BY1", country: 112},
		{sticker: "RU2", country: 643},
	}, nil)

	if got := len(grouped[CountryRussia]); got != 2 {
		t.Fatalf("expected 2 russia tares, got %d", got)
	}

	if got := len(grouped[CountryBelarus]); got != 1 {
		t.Fatalf("expected 1 belarus tare, got %d", got)
	}
}

func TestGroupTaresByCountryUnknownFallsBackToRussia(t *testing.T) {
	grouped := GroupTaresByCountry([]tareStub{
		{sticker: "X1", country: 999},
	}, nil)

	if len(grouped[CountryRussia]) != 1 {
		t.Fatalf("unknown country must fall back to russia: %+v", grouped)
	}
}

func TestGroupTaresByCountryOverrideOffice(t *testing.T) {
	override := map[int32]CountryCode{42: CountryKazakhstan}

	grouped := GroupTaresByCountry([]tareStub{
		{sticker: "KZ1", officeID: 42, country: 643},
	}, override)

	if len(grouped[CountryKazakhstan]) != 1 {
		t.Fatalf("override office must route to kazakhstan: %+v", grouped)
	}

	if _, ok := grouped[CountryRussia]; ok {
		t.Fatalf("overridden tare must not be grouped under russia: %+v", grouped)
	}
}
