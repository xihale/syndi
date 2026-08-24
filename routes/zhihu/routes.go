package routes

import (
	"github.com/xihale/rsshub-go/internal/routeutils"
)

// Routes lists all zhihu routes registered under /zhihu.
//
// Not ported from upstream (with reasons):
//   - topic/:id        需要 x-zse-96 请求签名（JS VM 解 __zse_ck 挑战），纯 Go 暂未实现
//   - timeline         登录用户私人时间线，依赖签名接口
//   - all-collections  用户本人全部收藏夹，依赖登录态与签名接口
//   - pins             想法热榜/用户想法，依赖签名接口
//   - check-cookie     上游为 JSON API 路由；本项目暂无该路由类型，
//     可用任一 cookie 路由是否报错代替验证
var Routes = []routeutils.RouteSpec{
	zhihuHotRoute,
	zhihuDailyRoute,
	zhihuDailySectionRoute,
	zhihuZhuanlanRoute,
	zhihuQuestionRoute,
	zhihuPeopleAnswersRoute,
	zhihuPostsRoute,
	zhihuActivitiesRoute,
	zhihuCollectionRoute,
	zhihuWeeklyRoute,
}
