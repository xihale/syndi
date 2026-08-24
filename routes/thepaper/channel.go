package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const thepaperBaseURL = "https://m.thepaper.cn"

func thepaperProfile() *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(thepaperBaseURL + "/")
}

// thepaperChannelNames maps well-known channel ids to display names.
var thepaperChannelNames = map[string]string{
	"26916":  "视频",
	"108856": "战疫",
	"25950":  "时事",
	"25951":  "财经",
	"36079":  "澎湃号",
	"119908": "科技",
	"25952":  "思想",
	"119489": "智库",
	"25953":  "生活",
	"26161":  "问吧",
	"122908": "国际",
	"-21":    "体育",
	"-24":    "评论",
}

func thepaperChannelName(id string) string {
	if name, ok := thepaperChannelNames[id]; ok {
		return name
	}
	return "频道 " + id
}

type thepaperChannelItem struct {
	ContID          string `json:"contId"`
	Link            string `json:"link"`
	Name            string `json:"name"`
	Pic             string `json:"pic"`
	PubTimeLong     int64  `json:"pubTimeLong"`
	CornerLabelDesc string `json:"cornerLabelDesc"`
	NodeInfo        struct {
		Name string `json:"name"`
	} `json:"nodeInfo"`
}

type thepaperChannelResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List []thepaperChannelItem `json:"list"`
	} `json:"data"`
}

type thepaperDetailResp struct {
	Props struct {
		PageProps struct {
			DetailData struct {
				ContentDetail struct {
					Name        string `json:"name"`
					Summary     string `json:"summary"`
					Author      string `json:"author"`
					Content     string `json:"content"`
					PublishTime int64  `json:"publishTime"`
				} `json:"contentDetail"`
			} `json:"detailData"`
		} `json:"pageProps"`
	} `json:"props"`
}

var thepaperNextDataRe = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

var thepaperChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:id",
	Name:        "ThePaper Channel",
	Example:     "thepaper/channel/25950",
	Maintainers: []string{"xihale"},
	Description: "ThePaper (澎湃新闻) news channel feed. Ids: 25950 时事, 25951 财经, 119908 科技, 122908 国际, -21 体育, -24 评论, 26916 视频, etc.",
	Categories:  []models.Category{{Name: "new-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Channel id from the channel page URL"),
		routeutils.OptionalParam("limit", "Max items, default 20"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  ThePaperChannelHandler,
}

// ThePaperChannelHandler handles /thepaper/channel/:id
func ThePaperChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 40)
	ctx := c.Parent()

	var resp thepaperChannelResp
	req := map[string]any{"channelId": id}
	if err := thepaperProfile().JSONAccept().
		PostJSON("https://api.thepaper.cn/contentapi/nodeCont/getByChannelId", req).
		GetJSON(ctx, c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("thepaper api code %d: %s", resp.Code, resp.Msg)
	}

	feed := routeutils.NewFeed(
		"澎湃新闻频道 - "+thepaperChannelName(id),
		thepaperBaseURL+"/channel/"+id,
		"澎湃新闻"+thepaperChannelName(id)+"频道最新内容",
	)

	n := 0
	for _, entry := range resp.Data.List {
		if n >= limit {
			break
		}
		item := thepaperBuildItem(ctx, c.Client(), entry)
		if item == nil || item.Title == "" {
			continue
		}
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

func thepaperBuildItem(ctx context.Context, cl *client.Client, entry thepaperChannelItem) *models.Item {
	link := entry.Link
	internal := false
	if link == "" {
		link = thepaperBaseURL + "/detail/" + entry.ContID
		internal = true
	}

	title := entry.Name
	desc := "<p>" + html.EscapeString(entry.Name) + "</p>"
	var pubDate time.Time
	if entry.PubTimeLong > 0 {
		pubDate = time.UnixMilli(entry.PubTimeLong)
	}
	author := ""

	// Internal articles: fetch detail page for full content and a reliable date.
	if internal && entry.ContID != "" {
		body, err := thepaperProfile().Fetch(thepaperBaseURL+"/detail/"+entry.ContID).GetString(ctx, cl)
		if err == nil {
			if m := thepaperNextDataRe.FindStringSubmatch(body); m != nil {
				var detail thepaperDetailResp
				if json.Unmarshal([]byte(m[1]), &detail) == nil {
					cd := detail.Props.PageProps.DetailData.ContentDetail
					if cd.Name != "" {
						title = cd.Name
					}
					content := cd.Content
					if content == "" {
						content = cd.Summary
					}
					if content != "" {
						desc = content
					}
					author = cd.Author
					if cd.PublishTime > 0 {
						pubDate = time.UnixMilli(cd.PublishTime)
					}
				}
			}
		}
	}

	item := routeutils.NewItem(title, link, desc, pubDate)
	if item == nil {
		return nil
	}
	if entry.ContID != "" {
		item.GUID = entry.ContID
	}
	if author != "" {
		routeutils.SetItemAuthor(item, author, "", "")
	}
	if entry.NodeInfo.Name != "" {
		routeutils.SetCategories(item, entry.NodeInfo.Name)
	}
	return item
}
