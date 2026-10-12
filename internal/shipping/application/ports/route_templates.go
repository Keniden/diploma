package ports

import (
	"context"

	"last-mile-task/internal/shipping/domain/values"
)

// RouteTemplateProvider supplies route templates by their external template
// ID.
//
// TODO(LASTMILE-110): the real provider is backed by the route client
// (internal/clients/route, GetRoutecar) plus office points and distance
// mapping; the skeleton adapter returns ErrRouteTemplateNotFound until the
// clients are ported.
type RouteTemplateProvider interface {
	GetRouteTemplateByID(ctx context.Context, id values.RouteTemplateID) (values.RouteTemplate, error)
}
