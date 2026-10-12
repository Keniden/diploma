package parser

import "strings"

// WHEN POSITION('WBMP' IN UPPER(_tare_sticker)) > 0
// AND (STRING_TO_ARRAY(_tare_sticker, ':'))[4] IS NOT NULL
// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[3] ~ '^[0-9]+$')
// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[4] ~ '^[0-9]+$')
// THEN 'MPB'
//
// WHEN POSITION('ЦИЬЗ' IN UPPER(_tare_sticker)) > 0
// AND (STRING_TO_ARRAY(_tare_sticker, 'Ж'))[4] IS NOT NULL
// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[3] ~ '^[0-9]+$')
// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[4] ~ '^[0-9]+$')
// THEN 'MPB'

type PatternMPB struct{}

func (p *PatternMPB) Match(sticker string) (Data, bool) {
	for _, item := range [][2]string{
		{"WBMP", ":"},
		{"ЦИЬЗ", "Ж"},
	} {
		prefix := item[0]
		separator := item[1]

		if !strings.HasPrefix(sticker, prefix) {
			continue
		}

		parts := strings.Split(sticker, separator)
		if len(parts) != 4 {
			continue
		}

		_, ok := toInt64(parts[2])
		if !ok {
			continue
		}

		id, ok := toInt64(parts[3])
		if !ok {
			continue
		}

		return Data{
			TareID:   id,
			TareType: TareTypeMPB,
		}, true
	}

	return Data{}, false
}
