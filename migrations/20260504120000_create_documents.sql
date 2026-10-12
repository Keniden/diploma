-- +goose Up
CREATE TABLE lmt.documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id UUID NOT NULL REFERENCES lmt.shipment (shipment_id),
    event_uuid TEXT NOT NULL,
    document_id BIGINT NOT NULL,
    data_print_task_uuid TEXT NOT NULL,
    generate_task_uuid TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX documents_shipment_id_idx ON lmt.documents (shipment_id);

COMMENT ON TABLE lmt.documents IS 'Связь отгрузки с cargo-documents: event, слепок данных и задача генерации';

-- +goose Down
DROP TABLE IF EXISTS lmt.documents;
