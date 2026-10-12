package entities_test

import (
	"testing"

	"last-mile-task/internal/documents/domain/entities"
	"last-mile-task/internal/documents/domain/values"
)

func validSpec() values.DocumentSpec {
	return values.DocumentSpec{
		ShipmentID:       values.NewID(),
		FromRoutePointID: values.NewID(),
		ToRoutePointID:   values.NewID(),
		DocumentType:     values.DocumentTypeWaybillForward,
		EventUUID:        "evt-1",
		PackageIDs:       []values.PackageID{values.NewID(), values.NewID()},
	}
}

func TestNewDocument_Valid(t *testing.T) {
	spec := validSpec()

	doc, err := entities.NewDocument(spec)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if doc.ID == values.NilID {
		t.Fatal("expected non-nil document id")
	}
	if doc.ShipmentID != spec.ShipmentID {
		t.Fatal("shipment id mismatch")
	}
	if doc.FromRoutePointID != spec.FromRoutePointID || doc.ToRoutePointID != spec.ToRoutePointID {
		t.Fatal("route point ids mismatch")
	}
	if doc.DocumentType != values.DocumentTypeWaybillForward {
		t.Fatalf("expected FORWARD, got %v", doc.DocumentType)
	}
	if doc.EventUUID != "evt-1" {
		t.Fatalf("expected evt-1, got %q", doc.EventUUID)
	}
	if len(doc.PackageIDs) != 2 {
		t.Fatalf("expected 2 package ids, got %d", len(doc.PackageIDs))
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("expected valid document, got %v", err)
	}
}

func TestNewDocument_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*values.DocumentSpec)
	}{
		{name: "missing shipment id", mutate: func(s *values.DocumentSpec) { s.ShipmentID = values.NilID }},
		{name: "unspecified document type", mutate: func(s *values.DocumentSpec) { s.DocumentType = values.DocumentTypeUnspecified }},
		{name: "empty event uuid", mutate: func(s *values.DocumentSpec) { s.EventUUID = "" }},
		{name: "no package ids", mutate: func(s *values.DocumentSpec) { s.PackageIDs = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := validSpec()
			tt.mutate(&spec)

			if _, err := entities.NewDocument(spec); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDocument_PackageIDsAreCopied(t *testing.T) {
	spec := validSpec()
	doc, err := entities.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}

	spec.PackageIDs[0] = values.PackageID(values.NewID())
	if doc.PackageIDs[0] == spec.PackageIDs[0] {
		t.Fatal("package ids must be copied, not aliased")
	}
}

func TestDocument_AssignTransitions(t *testing.T) {
	doc, err := entities.NewDocument(validSpec())
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.AssignExternalID(99); err != nil {
		t.Fatalf("expected first assign to succeed, got %v", err)
	}
	if doc.ExternalID != 99 {
		t.Fatalf("expected external id 99, got %d", doc.ExternalID)
	}
	if err := doc.AssignExternalID(100); err == nil {
		t.Fatal("expected second assign to fail")
	}

	if err := doc.AssignDataPrintTask("print-1"); err != nil {
		t.Fatalf("expected first print task assign to succeed, got %v", err)
	}
	if doc.DataPrintTaskID != "print-1" {
		t.Fatalf("expected print-1, got %q", doc.DataPrintTaskID)
	}
	if err := doc.AssignDataPrintTask("print-2"); err == nil {
		t.Fatal("expected second print task assign to fail")
	}
}