package entities

import (
	"slices"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/optimal"
)

const (
	// SourceSeqNum value from which every route's source point starts
	SourceSeqNum = 0

	MinimumRouteLength = 2
)

// Route is aggregate
type Route struct {
	ID         values.RouteID
	ShipmentID values.ShipmentID
	TemplateID values.RouteTemplateID
	Distance   int64
	Duration   int64
	Points     []*RoutePoint
}

func NewRouteFromTemplate(template values.RouteTemplate) (*Route, error) {
	if len(template.Points) < MinimumRouteLength {
		return nil, lmterrors.NewInvalidRouteTooShortError("route is too short").WithAttributes(
			lmterrors.Attributes{
				"required": MinimumRouteLength,
				"got":      len(template.Points),
			},
		)
	}

	// TODO(LASTMILE-021): the real repository uses utils.Map
	// (internal/utils), which is not ported yet; map inline instead.
	points := make([]*RoutePoint, 0, len(template.Points))
	for _, tp := range template.Points {
		points = append(points, newRoutePointFromTemplate(tp))
	}

	route := Route{
		ID:         values.NewID(),
		TemplateID: template.RouteTemplateID,
		Distance:   template.Distance,
		Duration:   template.Duration,
		Points:     points,
	}

	route.RebuildSequence()

	return &route, nil
}

func (r *Route) SetDistance(distance int64) {
	r.Distance = distance
}

func (r *Route) SetDuration(duration int64) {
	r.Duration = duration
}

func assertRouteLengthGreater(r *Route, length int) error {
	if len(r.Points) < length {
		return lmterrors.NewInvalidRouteTooShortError("route is too short").WithAttributes(
			lmterrors.Attributes{
				"required": length,
			},
		)
	}
	return nil
}

func assertRouteLengthValid(r *Route) error {
	return assertRouteLengthGreater(r, MinimumRouteLength)
}

func (r *Route) Source() (*RoutePoint, error) {
	if err := assertRouteLengthValid(r); err != nil {
		return nil, err
	}
	return r.Points[0], nil
}

func (r *Route) Target() (*RoutePoint, error) {
	if err := assertRouteLengthValid(r); err != nil {
		return nil, err
	}

	return r.Points[len(r.Points)-1], nil
}

func (r *Route) AssignToShipment(id values.ShipmentID) {
	r.ShipmentID = id
}

// MiddlePoints returns all route points except source and target
func (r *Route) MiddlePoints() ([]*RoutePoint, error) {
	if err := assertRouteLengthValid(r); err != nil {
		return nil, err
	}

	return r.Points[1 : len(r.Points)-1], nil
}

// Loading can be handled at any route point except the last one
func (r *Route) RoutePointsForLoading() ([]*RoutePoint, error) {
	if err := assertRouteLengthValid(r); err != nil {
		return nil, err
	}

	return r.Points[:len(r.Points)-1], nil
}

// Unloading can be handled at any route point except the first one
func (r *Route) RoutePointsForUnloading() ([]*RoutePoint, error) {
	if err := assertRouteLengthValid(r); err != nil {
		return nil, err
	}

	return r.Points[1:], nil
}

func (r *Route) ApplyOptimization(optimalRoute optimal.Route) error {
	r.SetDistance(optimalRoute.Distance)
	r.SetDuration(optimalRoute.Duration)

	middlePoints, err := r.MiddlePoints()
	if err != nil {
		return err
	}

	optimalMiddlePoints, err := optimalRoute.MiddlePoints()
	if err != nil {
		return err
	}

	optimalSequence := make(map[values.PointID]optimal.Point)
	for _, optimalRoutePoint := range optimalMiddlePoints {
		optimalSequence[optimalRoutePoint.ID] = optimalRoutePoint
	}

	for _, routePoint := range middlePoints {
		optimalRoutePoint, ok := optimalSequence[routePoint.PointID]
		if !ok {
			return lmterrors.NewInvalidRoutePointNotFoundError("optimized route point not found")
		}

		routePoint.SetSequenceNumber(optimalRoutePoint.SeqNum)
		routePoint.SetDistance(optimalRoutePoint.Distance)
		routePoint.SetDuration(optimalRoutePoint.Duration)
	}

	r.RebuildSequence()

	return nil
}

func (r *Route) StripPointsByLoadedPackages(pkgs []*Package) error {
	loaded := make(map[values.PackageID]struct{})
	for _, pkg := range pkgs {
		if pkg.IsCarried() {
			loaded[pkg.ID] = struct{}{}
		}
	}

	pointsToPreserve := make(map[values.RoutePointID]struct{})

	for _, pkg := range pkgs {
		for _, action := range pkg.Actions {
			switch action.Type {
			case values.RoutePointActionTypeLoad:
				// If there are any packages to load
				// then preserve this route point
				pointsToPreserve[action.RoutePointID] = struct{}{}
			case values.RoutePointActionTypeUnload:
				// If there are already loaded packages to unload
				// then preserve this route point
				_, ok := loaded[pkg.ID]
				if ok {
					pointsToPreserve[action.RoutePointID] = struct{}{}
				}
			}
		}
	}

	// Always preserve source and target route points
	if len(r.Points) > 0 {
		pointsToPreserve[r.Points[0].ID] = struct{}{}
		if len(r.Points) > 1 {
			pointsToPreserve[r.Points[len(r.Points)-1].ID] = struct{}{}
		}
	}

	stripped := make([]*RoutePoint, 0, len(pointsToPreserve))

	for _, rp := range r.Points {
		_, ok := pointsToPreserve[rp.ID]
		if !ok {
			continue
		}

		stripped = append(stripped, rp)
	}

	r.Points = stripped

	return nil
}

// FilterPointFunc filters out route points when fn(rp) returns true
func (r *Route) FilterPointFunc(fn func(rp *RoutePoint) bool) {
	r.Points = slices.DeleteFunc(r.Points, fn)
	r.RebuildSequence()
}

func (r *Route) RebuildSequence() {
	r.sortPointsBySequence()
	for i, rp := range r.Points {
		rp.SeqNum = int32(SourceSeqNum + i)
	}
}

func (r *Route) sortPointsBySequence() {
	// sort by sequence number to preserve route points slice in order
	slices.SortStableFunc(
		r.Points,
		func(a *RoutePoint, b *RoutePoint) int {
			return int(a.SeqNum - b.SeqNum)
		},
	)
}

func (r *Route) FindPointByID(id values.RoutePointID) (*RoutePoint, bool) {
	for _, rp := range r.Points {
		if rp.ID == id {
			return rp, true
		}
	}

	return nil, false
}

func (r *Route) Tail(seqNum int32) []*RoutePoint {
	tail := make([]*RoutePoint, 0)

	for _, rp := range r.Points {
		if rp.SeqNum > seqNum {
			tail = append(tail, rp)
		}
	}

	return tail
}
