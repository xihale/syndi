package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

type miuiCommunityResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Entity  struct {
		Records []miuiPost `json:"records"`
	} `json:"entity"`
}

type miuiPost struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	TextContent string `json:"textContent"` // rich-text HTML
	Pic         string `json:"pic"`
	Cover       string `json:"cover"`
	CreateTime  int64  `json:"createTime"` // unix millis
	Author      struct {
		Name string `json:"name"`
	} `json:"author"`
}

var miuiCommunityUserRoute = routeutils.RouteSpec{
	Path:        "community/user/:uid",
	Name:        "Mi Community User Posts",
	Example:     "miui/community/user/1200057564",
	Maintainers: []string{"xihale"},
	Description: "Latest posts of a Xiaomi Community (小米社区) user",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("uid", "Xiaomi user uid, from the uid query param of the community profile page"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  MiuiCommunityUserHandler,
}

// MiuiCommunityUserHandler handles /miui/community/user/:uid.
// Note: the legacy /miui/firmware/:device route is not ported because
// update.miui.com/updates/miota-fullrom.php now returns an empty body.
func MiuiCommunityUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	uid := strings.TrimSpace(c.Param("uid"))
	userLink := "https://web.vip.miui.com/page/info/mio/mio/homePage?uid=" + uid

	var resp miuiCommunityResp
	if err := routeutils.GetJSONWithHeaders(c.Parent(), c.Client(),
		fmt.Sprintf("https://api.vip.miui.com/api/community/user/announce/list?uid=%s&limit=10", uid),
		map[string]string{"Referer": userLink}, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 && len(resp.Entity.Records) == 0 {
		return nil, fmt.Errorf("xiaomi community API error: %s", resp.Message)
	}

	author := ""
	if len(resp.Entity.Records) > 0 {
		author = resp.Entity.Records[0].Author.Name
	}
	feed := routeutils.NewFeed(
		fmt.Sprintf("小米社区 - %s 的发帖", author),
		userLink,
		fmt.Sprintf("%s 的发帖", author),
	)
	for _, p := range resp.Entity.Records {
		title := p.Title
		if title == "" {
			if author == "" {
				continue
			}
			title = author + " 的动态"
		}
		desc := p.TextContent
		if desc == "" {
			img := firstNonEmpty(p.Pic, p.Cover)
			if img == "" {
				continue
			}
			desc = fmt.Sprintf(`<img src="%s">`, img)
		} else if img := firstNonEmpty(p.Pic, p.Cover); img != "" && !strings.Contains(desc, "<img") {
			desc += fmt.Sprintf(`<br><img src="%s">`, img)
		}
		itemLink := fmt.Sprintf("https://web.vip.miui.com/page/info/mio/mio/detail?postId=%s", p.ID)
		item := routeutils.NewItem(title, itemLink, desc, time.UnixMilli(p.CreateTime))
		item.GUID = p.ID
		routeutils.SetItemAuthor(item, p.Author.Name, "", userLink)
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
