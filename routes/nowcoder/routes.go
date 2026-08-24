package routes

import "github.com/xihale/syndi/internal/routeutils"

// Routes exports the nowcoder RouteSpecs for registration.
var Routes = []routeutils.RouteSpec{
	nowCoderHotsRoute,
	nowCoderHotsTypeRoute,
	nowCoderScheduleRoute,
	nowCoderSchedulePropertyRoute,
	nowCoderSchedulePropertyTypeRoute,
	nowCoderInterviewRoute,
}
