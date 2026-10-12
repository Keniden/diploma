package values

// DocumentType is the type of a generated waybill document.
type DocumentType int16

const (
	// DocumentTypeUnspecified is the zero value (matches lmt.documents default 0).
	DocumentTypeUnspecified DocumentType = 0
	// DocumentTypeWaybillForward is the forward-loading waybill (1).
	DocumentTypeWaybillForward DocumentType = 1
	// DocumentTypeWaybillReturn is the return waybill (2).
	DocumentTypeWaybillReturn DocumentType = 2
)

// DocumentSpec is the immutable input the Document aggregate is built from.
type DocumentSpec struct {
	ShipmentID       ShipmentID
	FromRoutePointID RoutePointID
	ToRoutePointID   RoutePointID
	DocumentType     DocumentType
	EventUUID        string
	ExternalID       int64
	DataPrintTaskID  string
	GenerateTaskID   string
	PackageIDs       []PackageID
}