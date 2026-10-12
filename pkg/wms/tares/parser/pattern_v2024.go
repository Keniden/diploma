//nolint:gosec
package parser

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
)

type PatternV2024 struct{}

func (p *PatternV2024) Match(sticker string) (Data, bool) {
	parts := strings.Split(sticker, ";")
	if len(parts) < 7 { // only 7 fields are required
		return Data{}, false
	}

	id, err := p.ParseTareID(parts[4])
	if err != nil {
		slog.Error("pattern_v2024: parse tare id", slog.Any("error", err))

		return Data{}, false
	}

	tareType, ok := p.ParseTareType(parts[5])
	if !ok {
		return Data{}, false
	}

	return Data{
		TareID:   id,
		TareType: tareType,
	}, true
}

func (p *PatternV2024) ParseTareType(encodedPart string) (TareType, bool) {
	for _, tareType := range TareTypes {
		if encodedPart == string(tareType) {
			return tareType, true
		}
	}

	return TareTypeUnknown, false
}

func (p *PatternV2024) ParseTareID(encodedPart string) (int64, error) {
	const (
		lenByteSize     = 2
		lenModulo       = 4
		checkSumSymSize = 3
	)

	if len(encodedPart) <= checkSumSymSize {
		return 0, fmt.Errorf("parser: invalid part length")
	}

	checksum := encodedPart[len(encodedPart)-checkSumSymSize:]
	barcode := encodedPart[:len(encodedPart)-checkSumSymSize]

	crc, barcodeLength, err := p.ExtractFromChecksum(checksum)
	if err != nil {
		return 0, fmt.Errorf("parser: extract from checksum")
	}

	if barcodeLength != (len(barcode) % lenModulo) {
		return 0, fmt.Errorf("parser: invalid barcode length")
	}

	if p.Crc16Ccitt(barcode) != crc {
		return 0, fmt.Errorf("parser: invalid crc16")
	}

	id, err := p.Decode(barcode)
	if err != nil {
		return 0, err
	}

	return int64(id), nil
}

func (*PatternV2024) Decode(str string) (uint64, error) {
	const encodeCharList = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/#"

	var encodingAlphabetRegex = regexp.MustCompile(fmt.Sprintf("^[%s]+$", encodeCharList))

	if !encodingAlphabetRegex.MatchString(str) {
		return 0, errors.New("string contains forbidden symbols")
	}

	var pkg uint64
	pos := 0
	for i := len(str) - 1; i >= 0; i-- {
		chI := strings.Index(encodeCharList, string(str[i]))
		pkg += uint64(math.Pow(float64(len(encodeCharList)), float64(pos))) * uint64(chI)
		pos++
	}

	return pkg, nil
}

func (p *PatternV2024) ExtractFromChecksum(checksum string) (crc uint64, barcodeLength int, err error) {
	const (
		lenByteSize = 2
	)

	pkg, err := p.Decode(checksum)
	if err != nil {
		return 0, 0, err
	}

	data, mask := 0, 0

	for i := 0; i < lenByteSize; i++ {
		mask |= 1 << i
	}

	data = int(pkg) & mask

	return pkg >> lenByteSize, data, nil
}

func (*PatternV2024) Crc16Ccitt(data string) uint64 {
	entries := [256]uint64{}
	for i := range entries {
		crc := uint64(i)

		for range 8 {
			if crc&1 > 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}

		entries[i] = crc
	}

	crc := uint64(0)
	crc ^= 0xFFFF

	for _, v := range data {
		crc = uint64(entries[(crc^uint64(v))&0xFF]^(crc>>8)) & 0xFFFF
	}

	return crc ^ 0xFFFF
}
