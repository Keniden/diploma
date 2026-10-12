// Package publicdriver contains use cases for the driver mobile client:
// package scanning at route points and active shipment retrieval.
//
// TODO(LASTMILE-080): this is a slim port of the real public_driver module —
// only the SetPackagesStatus + GetActiveShipmentByDriverID slice is ported
// (scan_packages, sync_packages, get_active_shipment_id,
// get_shipment_by_id). The remaining use cases (create_shipment,
// list_items/documents, basket, reserves, showcase, packages listing, point
// lifecycle) and their transport methods are not ported and return
// Unimplemented.
package publicdriver
