package services

import (
	"fmt"

	"last-mile-task/internal/documents/domain/directions"
	"last-mile-task/internal/documents/domain/values"
)

// DocumentBuilderService assembles the short-goods content and metadata of a
// waybill document from the packages of a single direction.
type DocumentBuilderService struct{}

// NewDocumentBuilderService builds the document builder service.
func NewDocumentBuilderService() *DocumentBuilderService {
	return &DocumentBuilderService{}
}

// Build groups the input packages by direction, then assembles the goods
// lines and metadata for the waybill. The input is scoped to a single
// direction (From→To); Build derives the direction key from its packages and
// produces the WithShortGoods view of that direction.
func (s *DocumentBuilderService) Build(in values.BuildInput) (values.WithShortGoods, error) {
	if len(in.Packages) == 0 {
		return values.WithShortGoods{}, fmt.Errorf("build: no packages")
	}

	groups := directions.GroupByDirection(in.Packages)
	if len(groups) != 1 {
		return values.WithShortGoods{}, fmt.Errorf("build: expected a single direction, got %d", len(groups))
	}

	var (
		direction directions.DirectionKey
		pkgs      []directions.Package
	)
	for key, group := range groups {
		direction = key
		pkgs = group
		break
	}

	packageIDs := make([]values.PackageID, 0, len(pkgs))
	items := make([]values.ContentItem, 0, len(pkgs))

	for _, pkg := range pkgs {
		packageIDs = append(packageIDs, values.PackageID(pkg.ID))

		items = append(items, values.ContentItem{
			Good: values.Good{
				Article: pkg.TareSticker,
			},
			Quantity: values.Quantity{
				Value: 1,
				Unit:  "pkg",
			},
			Price: values.Price{Currency: "RUB"},
		})
	}

	if len(in.PackageIDs) > 0 {
		packageIDs = in.PackageIDs
	}

	return values.WithShortGoods{
		ShipmentID:       in.ShipmentID,
		DocumentType:     in.DocumentType,
		FromRoutePointID: direction.From,
		ToRoutePointID:   direction.To,
		PackageIDs:       packageIDs,
		EventUUID:        in.EventUUID,
		ExternalID:       in.ExternalID,
		DataPrintTaskID:  in.DataPrintTaskID,
		GenerateTaskID:   in.GenerateTaskID,
		Meta:             values.Meta{Driver: in.Driver, RouteCarType: in.RouteCarType},
		OfficePoint:      in.OfficePoint,
		Items:            items,
	}, nil
}