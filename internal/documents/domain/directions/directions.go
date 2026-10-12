package directions

import "github.com/google/uuid"

// PackageAction is the direction of a package within a route point pair.
type PackageAction int

const (
	// PackageActionUnspecified is the zero value.
	PackageActionUnspecified PackageAction = 0
	// PackageActionLoad means the package is loaded at From and carried to To.
	PackageActionLoad PackageAction = 1
	// PackageActionUnload means the package is unloaded at To.
	PackageActionUnload PackageAction = 2
)

// RoutePointID identifies a route point in a direction.
type RoutePointID = uuid.UUID

// DirectionKey identifies a "From → To" route-point pair: the direction along
// which goods move (forward or return).
type DirectionKey struct {
	From RoutePointID
	To   RoutePointID
}

// RoutePoint is a route point participating in a direction.
type RoutePoint struct {
	ID         RoutePointID
	ExternalID int64
}

// Package is a cargo unit with its direction (load point → unload point).
type Package struct {
	ID          uuid.UUID
	TareID      int64
	TareType    string
	TareSticker string
	Action      PackageAction
	From        RoutePointID
	To          RoutePointID
}

// GroupByDirection groups packages by their load→unload route-point pair,
// producing non-empty groups keyed by DirectionKey.
func GroupByDirection(pkgs []Package) map[DirectionKey][]Package {
	groups := make(map[DirectionKey][]Package)
	for _, pkg := range pkgs {
		key := DirectionKey{From: pkg.From, To: pkg.To}
		groups[key] = append(groups[key], pkg)
	}
	return groups
}

// HasAction reports whether any package matches the given action.
func HasAction(pkgs []Package, action PackageAction) bool {
	for _, pkg := range pkgs {
		if pkg.Action == action {
			return true
		}
	}
	return false
}

// IsEmpty reports whether the group has no packages.
func IsEmpty(pkgs []Package) bool {
	return len(pkgs) == 0
}