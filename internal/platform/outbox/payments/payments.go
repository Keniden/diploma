// Package payments is the payments outbox: it persists
// "payments.last_mile_shipment_accrual" messages into lmt.outbox (writer)
// inside the shipment completion transaction and delivers them to the
// payments service (reader) with at-least-once semantics.
package payments

// MessageTypeLastMileShipmentAccrual is the single outbox message type used
// to accrue a completed last-mile shipment (ARCHITECTURE.md §23 flow 8).
const MessageTypeLastMileShipmentAccrual = "payments.last_mile_shipment_accrual"