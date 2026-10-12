package scan

import "last-mile-task/internal/shipping/domain/values"

// RoutePointTree
//
//	 -> RoutePoint
//		-> ActionTypeLoad
//		  -> Package #1
//		  -> Package #2
//		-> ActionTypeUnload
//		  -> Package #3
type RoutePointTree map[values.RoutePointID]map[values.PackageActionType]map[values.PackageID]struct{}

func NewRoutePointTree(pkgs []Package) RoutePointTree {
	tree := make(RoutePointTree)
	for _, pkg := range pkgs {
		routePoint, ok := tree[pkg.RoutePointID]
		if !ok {
			routePoint = make(map[values.PackageActionType]map[values.PackageID]struct{})
		}

		packages, ok := routePoint[pkg.ActionType]
		if !ok {
			packages = map[values.PackageID]struct{}{}
		}

		packages[pkg.ID] = struct{}{}

		routePoint[pkg.ActionType] = packages
		tree[pkg.RoutePointID] = routePoint
	}
	return tree
}

// ShipmentTree
//
//	 -> Shipment #1
//		-> Package #1
//		-> Package #2
//	 -> Shipment #2
//		-> Package #3
type ShipmentTree map[values.ShipmentID][]Package

func NewShipmentTree(pkgs []Package) ShipmentTree {
	tree := make(ShipmentTree)
	for _, pkg := range pkgs {
		tree[pkg.ShipmentID] = append(tree[pkg.ShipmentID], pkg)
	}
	return tree
}
