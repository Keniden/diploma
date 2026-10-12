package values

import (
	"last-mile-task/internal/documents/domain/directions"
)

// Driver is the driver snapshot used in document metadata.
type Driver struct {
	ID    string
	Name  string
	Phone string
}

// Meta carries the per-document metadata (driver, vehicle type).
type Meta struct {
	Driver      Driver
	RouteCarType string
}

// OfficePoint is the warehouse/office the cargo is associated with.
type OfficePoint struct {
	PointID   int64
	CountryID int64
}

// Price is a money amount in a currency.
type Price struct {
	Amount   float64
	Currency string
}

// Quantity is a quantity with a unit of measure.
type Quantity struct {
	Value float64
	Unit  string
}

// Good is the cargo line of the waybill (short-goods description).
type Good struct {
	Name    string
	Article string
}

// ContentItem is a single goods line grouped by direction.
type ContentItem struct {
	Good     Good
	Price    Price
	Quantity Quantity
}

// BuildInput is the data the DocumentBuilderService needs to assemble the
// short-goods content of a single direction.
type BuildInput struct {
	ShipmentID       ShipmentID
	DocumentType     DocumentType
	FromRoutePointID RoutePointID
	ToRoutePointID   RoutePointID
	PackageIDs       []PackageID
	Packages         []directions.Package
	Driver           Driver
	RouteCarType     string
	OfficePoint      OfficePoint
	EventUUID        string
	ExternalID       int64
	DataPrintTaskID  string
	GenerateTaskID   string
}

// WithShortGoods is the builder output: the document content (meta, office,
// goods lines) ready for persistence and outbox delivery.
type WithShortGoods struct {
	ShipmentID       ShipmentID
	DocumentType     DocumentType
	FromRoutePointID RoutePointID
	ToRoutePointID   RoutePointID
	PackageIDs       []PackageID
	EventUUID        string
	ExternalID       int64
	DataPrintTaskID  string
	GenerateTaskID   string
	Meta             Meta
	OfficePoint      OfficePoint
	Items            []ContentItem
}