package values

import "strings"

type LicensePlate string

var plateNormalizer = strings.NewReplacer("-", "", " ", "", ".", "")

func NewLicensePlate(raw string) LicensePlate {
	return LicensePlate(strings.ToUpper(plateNormalizer.Replace(raw)))
}

type Vehicle struct {
	ID           int64
	LicensePlate LicensePlate
}
