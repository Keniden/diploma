// Package cargo is the cargo-core outbox: it persists "cargo.packages.*"
// messages into lmt.outbox (writers) and delivers them to the cargo-core API
// (readers).
//
// The real repository re-uses the cargo-core client types:
// cargo_core_client_v1.UpdatePackagesStatusPackage. The skeleton's
// internal/clients/cargo-core/v1 is not ported yet, so the wire DTO and the
// action constants are mirrored here (field names and JSON layout are
// identical: ID, Action, CurrentPointUUID, Dt). TODO(LASTMILE-053): switch
// to the client types once the cargo-core client is ported.
package cargo

import (
	"time"

	"github.com/google/uuid"
)

const (
	MessageTypeScannedPackages       = "cargo.packages.scanned"
	MessageTypePackagesStatusUpdated = "cargo.packages.status.updated"
)

type UpdatePackagesStatusActionType string

const (
	UpdatePackagesStatusActionTypeUnspecified UpdatePackagesStatusActionType = ""
	UpdatePackagesStatusActionTypeLoad        UpdatePackagesStatusActionType = "LOAD"
	UpdatePackagesStatusActionTypeUnload      UpdatePackagesStatusActionType = "UNLOAD"
	UpdatePackagesStatusActionTypeDepart      UpdatePackagesStatusActionType = "DEPART"
)

// UpdatePackagesStatusPackage mirrors
// cargo_core_client_v1.UpdatePackagesStatusPackage. TODO(LASTMILE-053).
type UpdatePackagesStatusPackage struct {
	ID               uuid.UUID
	Action           UpdatePackagesStatusActionType
	CurrentPointUUID uuid.UUID
	Dt               time.Time
}

type ScannedPackagesMessage struct {
	Packages []UpdatePackagesStatusPackage `json:"packages"`
}

type PackagesStatusMessage struct {
	Packages []UpdatePackagesStatusPackage `json:"packages"`
}
