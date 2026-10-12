-- +goose Up

-- Replace ambiguous route_point_id with clear from/to semantics.
-- route_point_id is dropped since no code reads or writes it anymore.
ALTER TABLE lmt.documents
    ADD COLUMN from_route_point_id UUID REFERENCES lmt.route_point(route_point_id),
    ADD COLUMN to_route_point_id UUID REFERENCES lmt.route_point(route_point_id);

UPDATE lmt.documents SET to_route_point_id = route_point_id WHERE route_point_id IS NOT NULL;

DROP INDEX IF EXISTS lmt.idx_documents_route_point;
ALTER TABLE lmt.documents DROP COLUMN IF EXISTS route_point_id;

CREATE INDEX idx_documents_from_route_point ON lmt.documents (from_route_point_id) WHERE from_route_point_id IS NOT NULL;
CREATE INDEX idx_documents_to_route_point ON lmt.documents (to_route_point_id) WHERE to_route_point_id IS NOT NULL;

-- Create document_package join table linking documents to packages.
CREATE TABLE lmt.document_package (
    document_id UUID NOT NULL REFERENCES lmt.documents(id) ON DELETE CASCADE,
    package_id UUID NOT NULL,
    PRIMARY KEY (document_id, package_id)
);

CREATE INDEX idx_document_package_package ON lmt.document_package (package_id);

COMMENT ON TABLE lmt.document_package IS 'Связь документа с грузоместами (тарами), входящими в этот документ';

-- +goose Down

DROP TABLE IF EXISTS lmt.document_package;

ALTER TABLE lmt.documents ADD COLUMN IF NOT EXISTS route_point_id UUID REFERENCES lmt.route_point(route_point_id);
CREATE INDEX IF NOT EXISTS idx_documents_route_point ON lmt.documents (route_point_id) WHERE route_point_id IS NOT NULL;

DROP INDEX IF EXISTS lmt.idx_documents_from_route_point;
DROP INDEX IF EXISTS lmt.idx_documents_to_route_point;

ALTER TABLE lmt.documents
    DROP COLUMN IF EXISTS to_route_point_id,
    DROP COLUMN IF EXISTS from_route_point_id;
