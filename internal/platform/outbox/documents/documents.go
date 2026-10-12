// Package documents is the documents outbox: it persists
// "documents.publish_after_start_shipment" messages into lmt.outbox (writer)
// and delivers them to cargo-documents CreateEventByType (reader) with
// at-least-once semantics.
package documents

// MessageTypePublishAfterStartShipment is the single outbox message type used
// for both the shipment-start and the intermediate-point-departure triggers
// (ARCHITECTURE.md §22.1). It covers the forward and the conditional return
// waybill flow.
const MessageTypePublishAfterStartShipment = "documents.publish_after_start_shipment"