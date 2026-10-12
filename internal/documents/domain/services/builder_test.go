package services_test

import (
	"testing"

	"github.com/google/uuid"

	"last-mile-task/internal/documents/domain/directions"
	"last-mile-task/internal/documents/domain/services"
	"last-mile-task/internal/documents/domain/values"
)

func buildInput(documentType values.DocumentType, pkgs ...directions.Package) values.BuildInput {
	return values.BuildInput{
		ShipmentID:       values.NewID(),
		DocumentType:     documentType,
		FromRoutePointID: values.NewID(),
		ToRoutePointID:   values.NewID(),
		Packages:         pkgs,
		Driver:           values.Driver{ID: "d-1", Name: "Ivan Petrov", Phone: "+7-900"},
		RouteCarType:     "van",
		OfficePoint:      values.OfficePoint{PointID: 101, CountryID: 643},
		EventUUID:        "evt-1",
	}
}

func TestBuild_Forward(t *testing.T) {
	from := values.NewID()
	to := values.NewID()
	pkgs := []directions.Package{
		{ID: uuid.New(), TareID: 1, TareType: "BOX", TareSticker: "BOX1", Action: directions.PackageActionLoad, From: from, To: to},
		{ID: uuid.New(), TareID: 2, TareType: "BOX", TareSticker: "BOX2", Action: directions.PackageActionLoad, From: from, To: to},
	}

	s := services.NewDocumentBuilderService()
	out, err := s.Build(buildInput(values.DocumentTypeWaybillForward, pkgs...))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if out.DocumentType != values.DocumentTypeWaybillForward {
		t.Fatalf("expected FORWARD, got %v", out.DocumentType)
	}
	if out.FromRoutePointID != from || out.ToRoutePointID != to {
		t.Fatal("route points mismatch")
	}
	if len(out.PackageIDs) != 2 {
		t.Fatalf("expected 2 package ids, got %d", len(out.PackageIDs))
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 content items, got %d", len(out.Items))
	}
	if out.Items[0].Good.Article != "BOX1" {
		t.Fatalf("expected BOX1, got %q", out.Items[0].Good.Article)
	}
	if out.Items[0].Quantity.Value != 1 || out.Items[0].Quantity.Unit != "pkg" {
		t.Fatalf("unexpected quantity: %+v", out.Items[0].Quantity)
	}
	if out.Meta.Driver.Name != "Ivan Petrov" {
		t.Fatalf("expected driver name, got %q", out.Meta.Driver.Name)
	}
	if out.Meta.RouteCarType != "van" {
		t.Fatalf("expected van, got %q", out.Meta.RouteCarType)
	}
	if out.OfficePoint.PointID != 101 {
		t.Fatalf("expected office point 101, got %d", out.OfficePoint.PointID)
	}
}

func TestBuild_Return(t *testing.T) {
	pkgs := []directions.Package{
		{ID: uuid.New(), TareID: 5, TareType: "BOX", TareSticker: "BOX5", Action: directions.PackageActionLoad},
	}

	out, err := services.NewDocumentBuilderService().Build(buildInput(values.DocumentTypeWaybillReturn, pkgs...))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if out.DocumentType != values.DocumentTypeWaybillReturn {
		t.Fatalf("expected RETURN, got %v", out.DocumentType)
	}
}

func TestBuild_NoPackages(t *testing.T) {
	_, err := services.NewDocumentBuilderService().Build(buildInput(values.DocumentTypeWaybillForward))
	if err == nil {
		t.Fatal("expected error for empty packages")
	}
}