package wms

import (
	"log/slog"
)

type UnloadedTare struct {
	Sticker string `json:"tare_sticker"`

	SourceOfficeID    int32  `json:"office_id"`
	CountryID         int64  `json:"country_id"`
	UnloadedTimestamp string `json:"unloaded_dt"`
}

func (t UnloadedTare) GetSticker() string {
	return t.Sticker
}

func (t UnloadedTare) GetSourceOfficeID() int32 {
	return t.SourceOfficeID
}

func (t UnloadedTare) GetCountryID() int64 {
	return t.CountryID
}

type LoadedTare struct {
	Sticker string `json:"tare_sticker"`

	SourceOfficeID  int32  `json:"office_id"`
	CountryID       int64  `json:"country_id"`
	LoadedTimestamp string `json:"loaded_dt"`

	DriverName  string `json:"driver_name"`
	DriverPhone string `json:"driver_phone"`
}

func (t LoadedTare) GetSticker() string {
	return t.Sticker
}

func (t LoadedTare) GetSourceOfficeID() int32 {
	return t.SourceOfficeID
}

func (t LoadedTare) GetCountryID() int64 {
	return t.CountryID
}

type Tare interface {
	GetSticker() string
	GetSourceOfficeID() int32
	GetCountryID() int64
}

func GroupTaresByCountry[
	T []E,
	E Tare,
](
	tares T,
	overrideOffices map[int32]CountryCode,
) map[CountryCode][]E {
	taresByCountry := make(map[CountryCode][]E)

	for _, tare := range tares {
		if country, ok := overrideOffices[tare.GetSourceOfficeID()]; ok {
			taresByCountry[country] = append(taresByCountry[country], tare)
		} else {
			code := countryByNumeric(tare.GetCountryID())
			if code == CountryUnknown {
				code = CountryRussia

				slog.Warn(
					"wms: unknown tare country, fallback to russia",
					slog.String("tare_sticker", tare.GetSticker()),
					slog.Int64("country_id", tare.GetCountryID()),
				)
			}

			taresByCountry[code] = append(taresByCountry[code], tare)
		}
	}

	return taresByCountry
}
