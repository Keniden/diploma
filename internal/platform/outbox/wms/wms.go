// Package wms is the WMS outbox: it persists "wms.tares.*" messages into
// lmt.outbox (writer) and exports them to the WMS tare API (readers) via
// pkg/wms.Client.
package wms

import (
	"last-mile-task/pkg/wms"
	"last-mile-task/pkg/wms/tares"
)

const (
	MessageTypeTaresLoad   = "wms.tares.load"
	MessageTypeTaresUnload = "wms.tares.unload"
)

type Message struct {
	Shipment wms.Shipment `json:"shipment"`
	Tares    []tares.Tare `json:"tares"`
}
