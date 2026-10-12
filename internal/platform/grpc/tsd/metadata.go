// The TSD proto contract is simplified to Empty→Empty (TODO LASTMILE-008):
// the real scan payload is carried by the request message in the upstream repo.
// Until it lands, the transport resolves the scan parameters from the incoming
// gRPC metadata, documented here as the slice mechanism. Each metadata key uses
// a `tsd_` prefix to avoid collisions with the interceptors.
package tsd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
)

const (
	metadataQrToken    = "tsd_qr_token"
	metadataShipmentID = "tsd_shipment_id"
	metadataRoutePoint = "tsd_route_point_id"
	metadataPackages   = "tsd_packages"
)

// scanPackageMetadata is the JSON shape of a scanned package carried in the
// tsd_packages metadata entry.
type scanPackageMetadata struct {
	PackageID    string `json:"package_id"`
	RoutePointID string `json:"route_point_id"`
	PointID      int64  `json:"point_id"`
	Action       int32  `json:"action"`
	Timestamp    string `json:"timestamp"`
	TareSticker  string `json:"tare_sticker"`
}

// resolveScanShipment returns the reserve QR handshake token from the metadata.
func resolveScanShipment(ctx context.Context) (string, error) {
	md, err := incomingMetadata(ctx)
	if err != nil {
		return "", err
	}

	token, ok := mdFirst(md, metadataQrToken)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing qr token metadata")
	}

	return token, nil
}

// resolveScanPackages returns the shipment id and the scanned packages from the
// metadata.
func resolveScanPackages(ctx context.Context) (values.ShipmentID, []scan.Package, error) {
	md, err := incomingMetadata(ctx)
	if err != nil {
		return values.ShipmentID{}, nil, err
	}

	shipmentID, err := mdUUID(md, metadataShipmentID)
	if err != nil {
		return values.ShipmentID{}, nil, err
	}

	raw, ok := mdFirst(md, metadataPackages)
	if !ok {
		return values.ShipmentID{}, nil, status.Error(codes.InvalidArgument, "missing packages metadata")
	}

	var items []scanPackageMetadata
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return values.ShipmentID{}, nil, status.Errorf(codes.InvalidArgument, "parse packages metadata: %v", err)
	}

	packages := make([]scan.Package, 0, len(items))
	for _, item := range items {
		pkg, err := toScanPackage(item)
		if err != nil {
			return values.ShipmentID{}, nil, status.Errorf(codes.InvalidArgument, "parse package metadata: %v", err)
		}
		pkg.ShipmentID = shipmentID
		packages = append(packages, pkg)
	}

	return shipmentID, packages, nil
}

// resolveListPackagesToScan returns the shipment id and the route point id from
// the metadata.
func resolveListPackagesToScan(ctx context.Context) (values.ShipmentID, values.RoutePointID, error) {
	md, err := incomingMetadata(ctx)
	if err != nil {
		return values.ShipmentID{}, values.RoutePointID{}, err
	}

	shipmentID, err := mdUUID(md, metadataShipmentID)
	if err != nil {
		return values.ShipmentID{}, values.RoutePointID{}, err
	}

	routePointID, err := mdUUID(md, metadataRoutePoint)
	if err != nil {
		return values.ShipmentID{}, values.RoutePointID{}, err
	}

	return shipmentID, values.RoutePointID(routePointID), nil
}

func toScanPackage(item scanPackageMetadata) (scan.Package, error) {
	pkgID, err := uuid.Parse(item.PackageID)
	if err != nil {
		return scan.Package{}, fmt.Errorf("package id: %w", err)
	}

	routePointID, err := uuid.Parse(item.RoutePointID)
	if err != nil {
		return scan.Package{}, fmt.Errorf("route point id: %w", err)
	}

	var ts time.Time
	if item.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, item.Timestamp)
		if err != nil {
			return scan.Package{}, fmt.Errorf("timestamp: %w", err)
		}
		ts = parsed
	}

	return scan.Package{
		ID:           values.PackageID(pkgID),
		RoutePointID: values.RoutePointID(routePointID),
		PointID:      values.PointID(item.PointID),
		ActionType:   values.PackageActionType(item.Action),
		Timestamp:    ts,
		TareSticker:  item.TareSticker,
	}, nil
}

func incomingMetadata(ctx context.Context) (metadata.MD, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing incoming metadata")
	}
	return md, nil
}

func mdFirst(md metadata.MD, key string) (string, bool) {
	vals := md.Get(key)
	if len(vals) == 0 || vals[0] == "" {
		return "", false
	}
	return vals[0], true
}

func mdUUID(md metadata.MD, key string) (values.ShipmentID, error) {
	raw, ok := mdFirst(md, key)
	if !ok {
		return values.ShipmentID{}, status.Errorf(codes.InvalidArgument, "missing %s metadata", key)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return values.ShipmentID{}, status.Errorf(codes.InvalidArgument, "invalid %s metadata: %v", key, err)
	}

	return values.ShipmentID(id), nil
}
