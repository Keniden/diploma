// Handlers of the showcase + basket RPCs. The skeleton contract is
// Empty→Empty (see service.proto and TODO(LASTMILE-008)): the request inputs
// (point_id, route_id, item_id, page size/token) travel through gRPC metadata
// while the driver id is resolved from the interceptor context by the use
// cases.
package drivergrpc

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	lmterrors "last-mile-task/internal/errors"
)

const (
	defaultPageSize = 20

	metaPointID   = "point_id"
	metaRouteID   = "route_id"
	metaItemID    = "item_id"
	metaPageSize  = "page_size"
	metaPageToken = "page_token"
)

// ListPoints implements [driver.DriverServer]: the showcase point page.
func (s *Server) ListPoints(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	size, token, err := pageFromMetadata(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	if _, _, err := s.app.ListPoints(ctx, publicdriver.ListPointsRequest{
		PageSize:  size,
		PageToken: token,
	}); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// ListRoutesByPoint implements [driver.DriverServer]: the route page for a
// point.
func (s *Server) ListRoutesByPoint(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	pointID, err := metadataInt64(ctx, metaPointID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	size, token, err := pageFromMetadata(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	if _, _, err := s.app.ListRoutesByPoint(ctx, publicdriver.ListRoutesByPointRequest{
		PointID:   pointID,
		PageSize:  size,
		PageToken: token,
	}); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// ListItems implements [driver.DriverServer]: the current driver's basket.
func (s *Server) ListItems(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if _, err := s.app.ListItems(ctx); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// AddItem implements [driver.DriverServer]: adds a route to the basket.
func (s *Server) AddItem(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	routeID, err := metadataInt64(ctx, metaRouteID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	if err := s.app.AddItem(ctx, routeID); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// DeleteItem implements [driver.DriverServer]: removes an item from the basket.
func (s *Server) DeleteItem(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	itemID, err := metadataUUID(ctx, metaItemID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	if err := s.app.DeleteItem(ctx, itemID); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// ClearBasket implements [driver.DriverServer]: clears the driver's basket.
func (s *Server) ClearBasket(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.app.ClearBasket(ctx); err != nil {
		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// pageFromMetadata reads the cursor page size/token from gRPC metadata,
// falling back to defaults when absent.
//
// TODO(LASTMILE-008): once the RPCs carry page fields, read them from the
// request instead of the metadata.
func pageFromMetadata(ctx context.Context) (size, token int64, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return defaultPageSize, 0, nil
	}

	size = defaultPageSize
	if vals := md.Get(metaPageSize); len(vals) > 0 && vals[0] != "" {
		parsed, parseErr := strconv.ParseInt(vals[0], 10, 64)
		if parseErr != nil {
			return 0, 0, lmterrors.NewBadRequestError("invalid page_size")
		}
		size = parsed
	}

	if vals := md.Get(metaPageToken); len(vals) > 0 && vals[0] != "" {
		parsed, parseErr := strconv.ParseInt(vals[0], 10, 64)
		if parseErr != nil {
			return 0, 0, lmterrors.NewBadRequestError("invalid page_token")
		}
		token = parsed
	}

	return size, token, nil
}

// metadataInt64 reads and parses an int64 metadata value.
func metadataInt64(ctx context.Context, key string) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get(key)) == 0 || md.Get(key)[0] == "" {
		return 0, lmterrors.NewBadRequestError(key + " is required")
	}
	parsed, err := strconv.ParseInt(md.Get(key)[0], 10, 64)
	if err != nil {
		return 0, lmterrors.NewBadRequestError("invalid " + key)
	}
	return parsed, nil
}

// metadataUUID reads and parses a UUID metadata value.
func metadataUUID(ctx context.Context, key string) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get(key)) == 0 || md.Get(key)[0] == "" {
		return uuid.Nil, lmterrors.NewBadRequestError(key + " is required")
	}
	parsed, err := uuid.Parse(md.Get(key)[0])
	if err != nil {
		return uuid.Nil, lmterrors.NewBadRequestError("invalid " + key)
	}
	return parsed, nil
}