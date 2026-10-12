package documentspg

import (
	"github.com/google/uuid"

	"last-mile-task/internal/documents/domain/entities"
	"last-mile-task/internal/documents/domain/values"
)

// documentFromRecord maps a lmt.documents row and its package ids into the
// Document aggregate.
func documentFromRecord(rec documentRecord, packageIDs []uuid.UUID) *entities.Document {
	var from, to values.RoutePointID
	if rec.FromRoutePointID != nil {
		from = *rec.FromRoutePointID
	}
	if rec.ToRoutePointID != nil {
		to = *rec.ToRoutePointID
	}

	ids := make([]values.PackageID, 0, len(packageIDs))
	for _, id := range packageIDs {
		ids = append(ids, values.PackageID(id))
	}

	return &entities.Document{
		ID:               rec.ID,
		ShipmentID:       rec.ShipmentID,
		FromRoutePointID: from,
		ToRoutePointID:   to,
		DocumentType:     values.DocumentType(rec.DocumentType),
		EventUUID:        rec.EventUUID,
		ExternalID:       rec.DocumentID,
		DataPrintTaskID:  rec.DataPrintTaskID,
		GenerateTaskID:   rec.GenerateTaskID,
		PackageIDs:       ids,
	}
}