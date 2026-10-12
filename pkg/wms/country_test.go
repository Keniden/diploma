package wms

import "testing"

func TestCountryByName(t *testing.T) {
	cases := map[string]CountryCode{
		"russia":     CountryRussia,
		"Russia":     CountryRussia,
		"RUSSIA":     CountryRussia,
		"belarus":    CountryBelarus,
		"Kazakhstan": CountryKazakhstan,
		"BY":         CountryBelarus,
		"by":         CountryBelarus,
	}

	for name, expected := range cases {
		if got := countryByName(name); got != expected {
			t.Fatalf("countryByName(%q) = %q, want %q", name, got, expected)
		}
	}

	if got := countryByName("atlantis"); got != CountryUnknown {
		t.Fatalf("unknown country must map to CountryUnknown, got %q", got)
	}
}

func TestCountryByNumeric(t *testing.T) {
	cases := map[int64]CountryCode{
		643: CountryRussia,
		112: CountryBelarus,
		398: CountryKazakhstan,
		51:  CountryArmenia,
	}

	for code, expected := range cases {
		if got := countryByNumeric(code); got != expected {
			t.Fatalf("countryByNumeric(%d) = %q, want %q", code, got, expected)
		}
	}

	if got := countryByNumeric(0); got != CountryUnknown {
		t.Fatalf("0 must map to CountryUnknown, got %q", got)
	}

	if got := countryByNumeric(999); got != CountryUnknown {
		t.Fatalf("999 must map to CountryUnknown, got %q", got)
	}
}
