package parser

import "strings"

type PatternTBXPLT struct{}

func (p *PatternTBXPLT) Match(sticker string) (Data, bool) {
	// WHEN POSITION('TRBX' IN UPPER(_tare_sticker)) > 0
	// AND (TRIM('TRBX' FROM UPPER(_tare_sticker)) ~ '^[0-9]+$')
	// THEN 'TBX'

	// WHEN POSITION('ЕКИЧ' IN UPPER(_tare_sticker)) > 0
	// AND (TRIM('ЕКИЧ' FROM UPPER(_tare_sticker)) ~ '^[0-9]+$')
	// THEN 'TBX'

	for _, prefix := range []string{"TRBX", "ЕКИЧ", "WHPT"} {
		if !strings.HasPrefix(sticker, prefix) {
			continue
		}

		id, ok := toInt64(strings.TrimPrefix(sticker, prefix))
		if !ok {
			continue
		}

		tt := TareTypeTBX

		if prefix == "WHPT" {
			tt = TareTypePLT
		}

		return Data{
			TareID:   id,
			TareType: tt,
		}, true
	}

	// WHEN LEFT(_tare_sticker, 1) = '$'
	// AND (STRING_TO_ARRAY(_tare_sticker, ':'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'
	//
	// WHEN LEFT(_tare_sticker, 1) = '$'
	// AND (STRING_TO_ARRAY(_tare_sticker, ';'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, ';'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'
	//
	// WHEN LEFT(_tare_sticker, 1) = '%'
	// AND (STRING_TO_ARRAY(_tare_sticker, ':'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'
	//
	// WHEN LEFT(_tare_sticker, 1) = '%'
	// AND (STRING_TO_ARRAY(_tare_sticker, ';'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, ';'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'
	//
	// WHEN LEFT(_tare_sticker, 1) = '*'
	// AND (STRING_TO_ARRAY(_tare_sticker, ':'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, ':'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'
	//
	// WHEN LEFT(_tare_sticker, 1) = '*'
	// AND (STRING_TO_ARRAY(_tare_sticker, ';'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, ';'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'
	//
	// WHEN LEFT(_tare_sticker, 1) = ';'
	// AND (STRING_TO_ARRAY(_tare_sticker, 'Ж'))[3] IS NOT NULL
	// AND ((STRING_TO_ARRAY(_tare_sticker, 'Ж'))[3] ~ '^[0-9]+$')
	// THEN 'TBX'

	for _, item := range [][2]string{
		{"$", ":"},
		{"$", ";"},

		{"%", ":"},
		{"%", ";"},

		{"*", ":"},
		{"*", ";"},

		{";", "Ж"},
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

		id, ok := toInt64(parts[2])
		if !ok {
			continue
		}

		return Data{
			TareID:   id,
			TareType: TareTypeTBX,
		}, true
	}

	// WHEN (_tare_sticker ~ '^[0-9]+$') AND CAST(_tare_sticker AS BIGINT) > 902056545 THEN 'TBX'

	if id, ok := toInt64(sticker); ok && id > 902056545 {
		return Data{
			TareID:   id,
			TareType: TareTypeTBX,
		}, true
	}

	return Data{}, false
}
