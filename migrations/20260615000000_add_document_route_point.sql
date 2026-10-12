-- +goose Up
-- Add route_point_id and document_type to lmt.documents for per-point document association.
ALTER TABLE lmt.documents
    ADD COLUMN route_point_id UUID REFERENCES lmt.route_point(route_point_id),
    ADD COLUMN document_type SMALLINT NOT NULL DEFAULT 0;

CREATE INDEX idx_documents_route_point ON lmt.documents (route_point_id) WHERE route_point_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS lmt.idx_documents_route_point;
ALTER TABLE lmt.documents DROP COLUMN IF EXISTS document_type;
ALTER TABLE lmt.documents DROP COLUMN IF EXISTS route_point_id;
