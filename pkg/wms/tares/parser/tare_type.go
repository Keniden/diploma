package parser

type TareType string

const (
	TareTypeUnknown TareType = "UNKNOWN"

	TareTypeTBX TareType = "TBX" // Тара от склада на пвз с заказами
	TareTypeMPB TareType = "MPB"
	TareTypeCON TareType = "CON"

	TareTypeTEX TareType = "TEX" // Внешняя тара
	TareTypeTRN TareType = "TRN" // Возвратные тары
	TareTypeLAY TareType = "LAY" // Тара раскладки
	TareTypeREP TareType = "REP" // Тара переупаковки
	TareTypeBOX TareType = "BOX" // Коробка мезонина
	TareTypePLT TareType = "PLT" // Паллета
)

var TareTypes = []TareType{
	TareTypeTBX,
	TareTypeMPB,
	TareTypeCON,

	TareTypeTEX,
	TareTypeTRN,
	TareTypeLAY,
	TareTypeREP,
	TareTypeBOX,
	TareTypePLT,
}
