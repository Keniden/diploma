// Package cargodocuments is the seam to the external cargo-documents gRPC
// service (CreateEventByType). The external SDK/easyp proto is not ported, so
// a narrow DocumentsClient interface is used; the real client replaces this
// adapter once ported (see TODO(LASTMILE-099)).
package cargodocuments

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Item is a single goods line of the document event.
type Item struct {
	Article string  `json:"article"`
	Qty     float64 `json:"qty"`
	Unit    string  `json:"unit,omitempty"`
}

// Event is the cargo-documents CreateEventByType payload. Field names and the
// JSON layout are mirrored from the real repository's message; they must be
// confirmed against the real client when it is ported.
//
// TODO(LASTMILE-099): switch to the generated cargo_documents_client types
// (CreateEventByType request) once the external proto/SDK is ported.
type Event struct {
	EventUUID        string     `json:"event_uuid"`
	Type             string     `json:"type"`
	ShipmentID       uuid.UUID  `json:"shipment_id"`
	FromRoutePointID *uuid.UUID `json:"from_route_point_id,omitempty"`
	ToRoutePointID   *uuid.UUID `json:"to_route_point_id,omitempty"`
	DocumentType     int16      `json:"document_type"`
	DocumentID       int64      `json:"document_id"`
	DataPrintTaskID  string     `json:"data_print_task_uuid"`
	GenerateTaskID   string     `json:"generate_task_uuid"`
	PackageIDs       []uuid.UUID `json:"package_ids"`
	Items            []Item     `json:"items,omitempty"`
	DriverName       string     `json:"driver_name,omitempty"`
	DriverPhone      string     `json:"driver_phone,omitempty"`
	RouteCarType     string     `json:"route_car_type,omitempty"`
	OfficePointID    int64      `json:"office_point_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at,omitempty"`
}

// DocumentsClient is the narrow surface the documents outbox reader needs to
// deliver a document event to cargo-documents.
type DocumentsClient interface {
	CreateEventByType(ctx context.Context, event Event) error
}

// Client is the slice-scoped stand-in for the cargo-documents service.
//
// TODO(LASTMILE-099): replace with the client-backed adapter once the
// internal/clients/cargo-documents/v1 (generated proto) is ported. Calls are
// logged and discarded, keeping the delivery path green.
type Client struct{}

// NewClient builds the cargo-documents stub.
func NewClient() DocumentsClient { return &Client{} }

// CreateEventByType logs the outbound call (client not ported).
func (c *Client) CreateEventByType(_ context.Context, event Event) error {
	slog.Debug("stub cargo-documents: CreateEventByType",
		"event_uuid", event.EventUUID,
		"shipment_id", event.ShipmentID.String(),
		"document_type", event.DocumentType,
		"packages", len(event.PackageIDs),
	)

	return nil
}