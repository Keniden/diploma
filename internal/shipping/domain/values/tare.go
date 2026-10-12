package values

import (
	"strconv"
	"strings"
	"unicode"
)

func SplitTareSticker(raw string) (tareType TareType, tareID TareID) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return "", 0
	}

	firstDigitIdx := strings.IndexFunc(raw, unicode.IsDigit)
	if firstDigitIdx == -1 {
		return TareType(raw), 0
	}

	if firstDigitIdx == 0 {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return "", 0
		}
		return "", TareID(id)
	}

	numStr := raw[firstDigitIdx:]
	id, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return "", 0
	}

	return TareType(raw[:firstDigitIdx]), TareID(id)
}
