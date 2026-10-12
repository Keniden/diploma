package parser

import (
	"fmt"
	"strings"
)

type Data struct {
	TareID   int64
	TareType TareType
}

type Pattern interface {
	Match(sticker string) (Data, bool)
}

var Patterns = []Pattern{
	&PatternTBXPLT{},
	&PatternMPB{},
	&PatternCON{},
	&PatternV2024{},
}

func PrepareSticker(sticker string) string {
	return strings.ToUpper(strings.Trim(sticker, " "))
}

func Parse(sticker string) (Data, error) {
	sticker = PrepareSticker(sticker)

	for _, pattern := range Patterns {
		data, ok := pattern.Match(sticker)
		if !ok {
			continue
		}

		return data, nil
	}

	return Data{
		TareID:   -1,
		TareType: TareTypeUnknown,
	}, fmt.Errorf("parser: unknown sticker format %s", sticker)
}
