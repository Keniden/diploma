package entities

import (
	"errors"
	"fmt"

	"last-mile-task/internal/documents/domain/values"
	"last-mile-task/internal/primitives"
)

// ErrDocumentNotFound is returned when a document cannot be found by its key.
var ErrDocumentNotFound = errors.New("document: not found")

// Document is the aggregate persisting a waybill correlation row in
// lmt.documents plus its package bindings in lmt.document_package.
type Document struct {
	ID               values.DocumentID
	ShipmentID       values.ShipmentID
	FromRoutePointID values.RoutePointID
	ToRoutePointID   values.RoutePointID
	DocumentType     values.DocumentType
	EventUUID        string
	ExternalID       int64
	DataPrintTaskID  string
	GenerateTaskID   string
	PackageIDs       []values.PackageID
}

// NewDocument builds a Document aggregate from a spec, generating a fresh ID.
func NewDocument(spec values.DocumentSpec) (*Document, error) {
	if spec.ShipmentID == values.NilID {
		return nil, fmt.Errorf("document: shipment id is required")
	}
	if spec.DocumentType == values.DocumentTypeUnspecified {
		return nil, fmt.Errorf("document: document type is required")
	}
	if spec.EventUUID == "" {
		return nil, fmt.Errorf("document: event uuid is required")
	}
	if len(spec.PackageIDs) == 0 {
		return nil, fmt.Errorf("document: package ids are required")
	}

	return &Document{
		ID:               primitives.NewID(),
		ShipmentID:       spec.ShipmentID,
		FromRoutePointID: spec.FromRoutePointID,
		ToRoutePointID:   spec.ToRoutePointID,
		DocumentType:     spec.DocumentType,
		EventUUID:        spec.EventUUID,
		ExternalID:       spec.ExternalID,
		DataPrintTaskID:  spec.DataPrintTaskID,
		GenerateTaskID:   spec.GenerateTaskID,
		PackageIDs:       append([]values.PackageID{}, spec.PackageIDs...),
	}, nil
}

// Validate checks the aggregate invariants after assembly.
func (d *Document) Validate() error {
	if d.ID == values.NilID {
		return fmt.Errorf("document: id is required")
	}
	if d.ShipmentID == values.NilID {
		return fmt.Errorf("document: shipment id is required")
	}
	if d.DocumentType == values.DocumentTypeUnspecified {
		return fmt.Errorf("document: document type is required")
	}
	if d.EventUUID == "" {
		return fmt.Errorf("document: event uuid is required")
	}
	if len(d.PackageIDs) == 0 {
		return fmt.Errorf("document: package ids are required")
	}
	return nil
}

// AssignExternalID records the external document id produced by
// cargo-documents. It only succeeds once.
func (d *Document) AssignExternalID(externalID int64) error {
	if d.ExternalID != 0 {
		return fmt.Errorf("document: external id is already assigned")
	}
	d.ExternalID = externalID
	return nil
}

// AssignDataPrintTask records the print task id. It only succeeds once.
func (d *Document) AssignDataPrintTask(taskID string) error {
	if d.DataPrintTaskID != "" {
		return fmt.Errorf("document: data print task is already assigned")
	}
	d.DataPrintTaskID = taskID
	return nil
}