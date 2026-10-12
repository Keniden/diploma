package parser

import (
	"strings"
)

// WHEN POSITION('WCT' IN _tare_sticker) > 0
// AND (TRIM('WCT' FROM _tare_sticker) ~ '^[0-9]+$')
// THEN 'CON'

type PatternCON struct{}

func (p *PatternCON) Match(sticker string) (Data, bool) {
	for _, prefix := range []string{"WCT"} {
		if !strings.HasPrefix(sticker, prefix) {
			continue
		}

		_, ok := toInt64(strings.TrimPrefix(sticker, prefix))
		if !ok {
			continue
		}

		return Data{
			TareID:   0,
			TareType: TareTypeCON,
		}, true
	}

	return Data{}, false
}
