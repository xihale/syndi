package routes

import (
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	"github.com/xihale/syndi/pkg/models"
)

// bilibiliSocialCategories is the shared category list for this namespace.
func bilibiliSocialCategories() []models.Category {
	return []models.Category{{Name: "social-media"}}
}

// bilibiliEmbedParam documents the optional embed-disable path parameter.
func bilibiliEmbedParam() models.Parameter {
	return routeutils.OptionalParam("embed", "默认为开启内嵌视频, 任意值为关闭 (any value disables inline players)")
}

// bilibiliWithEmbedPath clones a spec registering the deeper "/:embed"
// variant so both /path/xxx and /path/xxx/embed work (gin has no optional
// trailing segments).
func bilibiliWithEmbedPath(spec routeutils.RouteSpec) routeutils.RouteSpec {
	clone := spec
	if !strings.HasSuffix(clone.Path, "/:embed") {
		clone.Path += "/:embed"
		clone = routeutils.RequireParams(clone, "embed")
	}
	return clone
}

// Routes lists all bilibili route specs in this package.
var Routes = []routeutils.RouteSpec{
	bilibiliPopularRoute,
	bilibiliHotSearchRoute,
	bilibiliRankingRoute,
	bilibiliRankingZoneRoute,

	bilibiliUserVideoRoute,
	bilibiliUserVideoEmbedRoute,
	bilibiliUserDynamicRoute,
	bilibiliUserDynamicParamsRoute,
	bilibiliVideoPageRoute,
	bilibiliVideoPageEmbedRoute,
	bilibiliVideoReplyRoute,
	bilibiliPartionRoute,
	bilibiliPartionEmbedRoute,
	bilibiliPartionRankingRoute,
	bilibiliPartionRankingDaysRoute,
	bilibiliBangumiMediaRoute,
	bilibiliBangumiMediaEmbedRoute,
	bilibiliAudioRoute,
	bilibiliLiveRoomRoute,
	bilibiliLiveSearchRoute,
	bilibiliReadlistRoute,
	bilibiliVsearchRoute,
	bilibiliVsearchOrderRoute,
	bilibiliWeeklyRoute,
	bilibiliWeeklyEmbedRoute,
}

var bilibiliUserVideoRoute = routeutils.RouteSpec{
	Path:        "user/video/:uid",
	Name:        "UP 主投稿",
	Example:     "bilibili/user/video/946974",
	URL:         "https://space.bilibili.com/946974",
	Maintainers: []string{"xihale"},
	Description: "Bilibili user uploaded videos (wbi-signed space arc/search API)",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("uid", "用户 id, 可在 UP 主主页中找到"),
		routeutils.OptionalParam("embed", bilibiliEmbedParam().Description),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  BilibiliUserVideoHandler,
}

var bilibiliUserDynamicRoute = routeutils.RouteSpec{
	Path:        "user/dynamic/:uid",
	Name:        "UP 主动态",
	Example:     "bilibili/user/dynamic/946974",
	URL:         "https://space.bilibili.com/946974/dynamic",
	Maintainers: []string{"xihale"},
	Description: "Bilibili user dynamics/timeline; optional RSSHub-style fragment e.g. embed=0&showEmoji=1",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("uid", "用户 id, 可在 UP 主主页中找到"),
		routeutils.RequiredParam("routeParams", "键值参数片段, 如 showEmoji=1&embed=0&useAvid=1"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  BilibiliUserDynamicHandler,
}

var bilibiliVideoPageRoute = routeutils.RouteSpec{
	Path:        "video/page/:bvid",
	Name:        "视频选集列表",
	Example:     "bilibili/video/page/BV1i7411M7N9",
	URL:         "https://www.bilibili.com/video/BV1i7411M7N9",
	Maintainers: []string{"xihale"},
	Description: "Bilibili video episodes/pages list (query limit supported)",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("bvid", "可在视频页 URL 中找到, 也支持 av 号"),
		routeutils.OptionalParam("embed", bilibiliEmbedParam().Description),
		routeutils.OptionalParam("limit", "Maximum number of items, default 10, max 100"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  BilibiliVideoPageHandler,
}

var bilibiliVideoPageEmbedRoute = bilibiliWithEmbedPath(bilibiliVideoPageRoute)

var bilibiliVideoReplyRoute = routeutils.RouteSpec{
	Path:        "video/reply/:bvid",
	Name:        "视频评论",
	Example:     "bilibili/video/reply/BV1i7411M7N9",
	URL:         "https://www.bilibili.com/video/BV1i7411M7N9",
	Maintainers: []string{"xihale"},
	Description: "Bilibili video comments (x/v2/reply/main); query mode=2 for latest, default hot",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("bvid", "可在视频页 URL 中找到, 也支持 av 号"),
		routeutils.OptionalParam("mode", "排序, 3=热门(默认) 2=最新 1=按时间"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  BilibiliVideoReplyHandler,
}

var bilibiliPartionRoute = routeutils.RouteSpec{
	Path:        "partion/:tid",
	Name:        "分区视频",
	Example:     "bilibili/partion/33",
	Maintainers: []string{"xihale"},
	Description: "Bilibili newest videos of a category zone (33 连载动画, 4 游戏, 17 单机, 171 电竞...)",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("tid", "分区 id"),
		routeutils.OptionalParam("embed", bilibiliEmbedParam().Description),
		routeutils.OptionalParam("limit", "Maximum number of items, default 30, max 100"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  BilibiliPartionHandler,
}

var bilibiliPartionEmbedRoute = bilibiliWithEmbedPath(bilibiliPartionRoute)

var bilibiliPartionRankingRoute = routeutils.RouteSpec{
	Path:        "partion/ranking/:tid",
	Name:        "分区视频排行榜",
	Example:     "bilibili/partion/ranking/171",
	Maintainers: []string{"xihale"},
	Description: "Bilibili category hot-rank videos, last 7 days by default",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("tid", "分区 id"),
		routeutils.RequiredParam("days", "最近多少天内的热度排序, 缺省为 7, 支持 1/3/7/30/90/120"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  BilibiliPartionRankingHandler,
}

var bilibiliPartionRankingDaysRoute = func() routeutils.RouteSpec {
	clone := bilibiliPartionRankingRoute
	clone.Path = "partion/ranking/:tid/:days"
	return clone
}()

var bilibiliBangumiMediaRoute = routeutils.RouteSpec{
	Path:        "bangumi/media/:mediaid",
	Name:        "番剧",
	Example:     "bilibili/bangumi/media/9192",
	URL:         "https://www.bilibili.com/bangumi/media/md9192",
	Maintainers: []string{"xihale"},
	Description: "Bilibili bangumi (OGV) season details and episode list",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("mediaid", "番剧媒体 id, 番剧主页 URL 中获取 (md/ss 前缀可选)"),
		routeutils.OptionalParam("embed", bilibiliEmbedParam().Description),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  BilibiliBangumiMediaHandler,
}

var bilibiliBangumiMediaEmbedRoute = bilibiliWithEmbedPath(bilibiliBangumiMediaRoute)

var bilibiliAudioRoute = routeutils.RouteSpec{
	Path:        "audio/:id",
	Name:        "歌单",
	Example:     "bilibili/audio/10624",
	URL:         "https://www.bilibili.com/audio/am10624",
	Maintainers: []string{"xihale"},
	Description: "Bilibili audio menu/playlist songs",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "歌单 id, 可在歌单页 URL 中找到"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  BilibiliAudioHandler,
}

var bilibiliLiveRoomRoute = routeutils.RouteSpec{
	Path:        "live/room/:roomID",
	Name:        "直播开播",
	Example:     "bilibili/live/room/3",
	URL:         "https://live.bilibili.com/3",
	Maintainers: []string{"xihale"},
	Description: "Bilibili live room status; emits one item while live, empty when offline",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("roomID", "房间号, 可在直播间 URL 中找到, 长短号均可"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  BilibiliLiveRoomHandler,
}

var bilibiliLiveSearchRoute = routeutils.RouteSpec{
	Path:        "live/search/:key",
	Name:        "直播间搜索",
	Example:     "bilibili/live/search/解密",
	URL:         "https://search.bilibili.com/live?keyword=解密",
	Maintainers: []string{"xihale"},
	Description: "Bilibili live room search via the public search/type endpoint",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("key", "检索关键字"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  BilibiliLiveSearchHandler,
}

var bilibiliReadlistRoute = routeutils.RouteSpec{
	Path:        "readlist/:listid",
	Name:        "专栏文集",
	Example:     "bilibili/readlist/25611",
	URL:         "https://www.bilibili.com/read/readlist/rl25611",
	Maintainers: []string{"xihale"},
	Description: "Bilibili article collection (readlist) with its articles",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("listid", "文集 id, 可在专栏文集 URL 中找到"),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  BilibiliReadlistHandler,
}

var bilibiliVsearchRoute = routeutils.RouteSpec{
	Path:        "vsearch/:kw",
	Name:        "视频搜索",
	Example:     "bilibili/vsearch/RSSHub",
	URL:         "https://search.bilibili.com/all?keyword=RSSHub",
	Maintainers: []string{"xihale"},
	Description: "Bilibili video search; supports query params order/embed/tid as well",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("kw", "检索关键字"),
		routeutils.RequiredParam("order", "排序方式, 综合:totalrank 最多点击:click 最新发布:pubdate(缺省) 最多弹幕:dm 最多收藏:stow"),
		routeutils.OptionalParam("embed", bilibiliEmbedParam().Description),
		routeutils.OptionalParam("tid", "分区 id"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  BilibiliVsearchHandler,
}

var bilibiliVsearchOrderRoute = func() routeutils.RouteSpec {
	clone := bilibiliVsearchRoute
	clone.Path = "vsearch/:kw/:order"
	return clone
}()

var bilibiliWeeklyRoute = routeutils.RouteSpec{
	Path:        "weekly",
	Name:        "B 站每周必看",
	Example:     "bilibili/weekly",
	URL:         "https://www.bilibili.com/h5/weekly-recommend",
	Maintainers: []string{"xihale"},
	Description: "Bilibili weekly popular selection",
	Categories:  bilibiliSocialCategories(),
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("embed", bilibiliEmbedParam().Description),
	},
	CacheTTL: 6 * time.Hour,
	Handler:  BilibiliWeeklyHandler,
}

var bilibiliWeeklyEmbedRoute = bilibiliWithEmbedPath(bilibiliWeeklyRoute)

// Deeper-path clones registered alongside their base specs above.
var bilibiliUserVideoEmbedRoute = bilibiliWithEmbedPath(bilibiliUserVideoRoute)

var bilibiliUserDynamicParamsRoute = func() routeutils.RouteSpec {
	clone := bilibiliUserDynamicRoute
	clone.Path = "user/dynamic/:uid/:routeParams"
	return clone
}()
