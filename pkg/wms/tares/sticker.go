package tares

import (
	"last-mile-task/pkg/wms/tares/parser"
)

type Sticker struct {
	Raw string

	tareID   int64
	tareType string
}

func (s *Sticker) String() string {
	return s.Raw
}

func (s *Sticker) TareID() int64 {
	return s.tareID
}

func (s *Sticker) TareType() string {
	return s.tareType
}

func NewSticker(sticker string) Sticker {
	return Sticker{
		Raw: sticker,
	}
}

func ParseSticker(sticker string) (Sticker, error) {
	data, err := parser.Parse(sticker)
	if err != nil {
		return Sticker{}, err
	}

	return Sticker{
		Raw: sticker,

		tareID:   data.TareID,
		tareType: string(data.TareType),
	}, nil
}
