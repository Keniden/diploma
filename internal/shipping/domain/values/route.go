package values

type RouteTemplate struct {
	RouteTemplateID RouteTemplateID
	Distance        int64
	Duration        int64
	Points          []RoutePointTemplate
}

type RoutePointTemplate struct {
	RoutePointTemplateID PointID
	Distance             int64
	SeqNum               int32
}
