package values

import "github.com/google/uuid"

// Documents identifiers are UUID aliases mirroring the shipping domain values
// but owned by the documents bounded context (no foreign imports).
type (
	DocumentID   = uuid.UUID
	ShipmentID   = uuid.UUID
	RoutePointID = uuid.UUID
	PackageID    = uuid.UUID
)

// NilID is the zero (empty) UUID used to detect missing identifiers.
var NilID = uuid.Nil

// NewID returns a fresh document identifier.
func NewID() DocumentID {
	return uuid.New()
}