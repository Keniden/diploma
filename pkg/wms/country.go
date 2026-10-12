package wms

// TODO(LASTMILE-041): the real repo uses github.com/biter777/countries for
// country codes. It is not in this skeleton's go.mod, so only the country
// names/numeric codes used by last-mile (WB footprint) are mapped here with a
// Russia fallback (mirrors the real repository's fallback behavior). Extend
// the maps when more countries are needed or once the dependency is allowed.

// CountryCode is an ISO 3166-1 alpha-2 country code used to route tares
// requests to the corresponding WMS endpoint.
type CountryCode string

const (
	CountryRussia     CountryCode = "RU"
	CountryBelarus    CountryCode = "BY"
	CountryKazakhstan CountryCode = "KZ"
	CountryArmenia    CountryCode = "AM"
	CountryKyrgyzstan CountryCode = "KG"
	CountryUzbekistan CountryCode = "UZ"
	CountryGeorgia    CountryCode = "GE"
	CountryAzerbaijan CountryCode = "AZ"
	CountryTurkey     CountryCode = "TR"
	CountryChina      CountryCode = "CN"
	CountryUAE        CountryCode = "AE"
	CountryUnknown    CountryCode = ""
)

func (c CountryCode) String() string {
	return string(c)
}

func (c CountryCode) Alpha2() string {
	return string(c)
}

var countriesByName = map[string]CountryCode{
	"russia":     CountryRussia,
	"belarus":    CountryBelarus,
	"kazakhstan": CountryKazakhstan,
	"armenia":    CountryArmenia,
	"kyrgyzstan": CountryKyrgyzstan,
	"uzbekistan": CountryUzbekistan,
	"georgia":    CountryGeorgia,
	"azerbaijan": CountryAzerbaijan,
	"turkey":     CountryTurkey,
	"china":      CountryChina,
	"uae":        CountryUAE,
}

var countriesByNumeric = map[int64]CountryCode{
	643: CountryRussia,
	112: CountryBelarus,
	398: CountryKazakhstan,
	51:  CountryArmenia,
	417: CountryKyrgyzstan,
	860: CountryUzbekistan,
	268: CountryGeorgia,
	31:  CountryAzerbaijan,
	792: CountryTurkey,
	156: CountryChina,
	784: CountryUAE,
}

func countryByName(name string) CountryCode {
	normalized := normalizeCountryName(name)

	code, ok := countriesByName[normalized]
	if ok {
		return code
	}

	// Accept ISO 3166-1 alpha-2 codes directly as well.
	for _, code := range countriesByName {
		if normalizeCountryName(string(code)) == normalized {
			return code
		}
	}

	return CountryUnknown
}

func countryByNumeric(code int64) CountryCode {
	if code == 0 {
		return CountryUnknown
	}

	c, ok := countriesByNumeric[code]
	if !ok {
		return CountryUnknown
	}

	return c
}

func normalizeCountryName(name string) string {
	lower := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}

		lower = append(lower, c)
	}

	return string(lower)
}
