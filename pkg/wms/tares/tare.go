package tares

import (
	"time"
	// Embeds the timezone database, fixes error during LoadLocation("Europe/Moscow") in containers
	_ "time/tzdata"
)

var loc, _ = time.LoadLocation("Europe/Moscow")

type Tare struct {
	Sticker Sticker

	OfficeID  int32
	CountryID int64
	Timestamp time.Time

	DestinationOfficeID int32
}

func (t *Tare) FormatTimestamp() string {
	return t.Timestamp.In(loc).Format(time.RFC3339Nano)
}
