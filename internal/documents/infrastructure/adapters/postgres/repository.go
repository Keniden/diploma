package documentspg

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"

	"last-mile-task/internal/documents/domain/entities"
	"last-mile-task/internal/documents/domain/values"
	"last-mile-task/internal/platform/database"
)

// DocumentRepository persists whole Document aggregates into lmt.documents
// and lmt.document_package.
type DocumentRepository struct {
	*database.Repository
}

// NewRepository builds the documents persistence adapter.
func NewRepository(repo *database.Repository) *DocumentRepository {
	return &DocumentRepository{Repository: repo}
}

// Save persists the whole aggregate: the lmt.documents row plus the package
// bindings in lmt.document_package, all inside the context transaction (if
// any) so the write is atomic with the caller's transaction.
func (r *DocumentRepository) Save(ctx context.Context, doc *entities.Document) error {
	query, args, err := r.Builder.
		Insert("lmt.documents").
		Columns(
			"id",
			"shipment_id",
			"event_uuid",
			"document_id",
			"data_print_task_uuid",
			"generate_task_uuid",
			"from_route_point_id",
			"to_route_point_id",
			"document_type",
		).
		Values(
			doc.ID,
			doc.ShipmentID,
			doc.EventUUID,
			doc.ExternalID,
			doc.DataPrintTaskID,
			doc.GenerateTaskID,
			nullableRoutePoint(doc.FromRoutePointID),
			nullableRoutePoint(doc.ToRoutePointID),
			int16(doc.DocumentType),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert document: %w", err)
	}

	if _, err := r.Querier(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	for _, packageID := range doc.PackageIDs {
		if _, err := r.Querier(ctx).Exec(ctx,
			`INSERT INTO lmt.document_package (document_id, package_id) VALUES ($1, $2)`,
			doc.ID, packageID,
		); err != nil {
			return fmt.Errorf("insert document package %s: %w", packageID, err)
		}
	}

	return nil
}

// GetByShipmentID returns all documents of a shipment, ordered by created_at.
func (r *DocumentRepository) GetByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
) ([]*entities.Document, error) {
	query, args, err := r.Builder.
		Select(
			"id",
			"shipment_id",
			"event_uuid",
			"document_id",
			"data_print_task_uuid",
			"generate_task_uuid",
			"from_route_point_id",
			"to_route_point_id",
			"document_type",
			"created_at",
		).
		From("lmt.documents").
		Where(sq.Eq{"shipment_id": shipmentID}).
		OrderBy("created_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list documents: %w", err)
	}

	var records []documentRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	documents := make([]*entities.Document, 0, len(records))
	for _, rec := range records {
		packages, err := r.packageIDsForDocument(ctx, rec.ID)
		if err != nil {
			return nil, err
		}
		documents = append(documents, documentFromRecord(rec, packages))
	}

	return documents, nil
}

// GetByEventUUID returns the document with the given event uuid.
func (r *DocumentRepository) GetByEventUUID(
	ctx context.Context,
	eventUUID string,
) (*entities.Document, error) {
	query, args, err := r.Builder.
		Select(
			"id",
			"shipment_id",
			"event_uuid",
			"document_id",
			"data_print_task_uuid",
			"generate_task_uuid",
			"from_route_point_id",
			"to_route_point_id",
			"document_type",
			"created_at",
		).
		From("lmt.documents").
		Where(sq.Eq{"event_uuid": eventUUID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get document: %w", err)
	}

	var rec documentRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rec, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get document by event uuid %s: %w", eventUUID, entities.ErrDocumentNotFound)
		}
		return nil, fmt.Errorf("get document: %w", err)
	}

	packages, err := r.packageIDsForDocument(ctx, rec.ID)
	if err != nil {
		return nil, err
	}

	return documentFromRecord(rec, packages), nil
}

func (r *DocumentRepository) packageIDsForDocument(
	ctx context.Context,
	documentID uuid.UUID,
) ([]uuid.UUID, error) {
	query, args, err := r.Builder.
		Select("document_id", "package_id").
		From("lmt.document_package").
		Where(sq.Eq{"document_id": documentID}).
		OrderBy("package_id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list document packages: %w", err)
	}

	var records []documentPackageRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("list document packages: %w", err)
	}

	ids := make([]uuid.UUID, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.PackageID)
	}

	return ids, nil
}

func nullableRoutePoint(id values.RoutePointID) *uuid.UUID {
	if id == values.NilID {
		return nil
	}
	v := uuid.UUID(id)
	return &v
}