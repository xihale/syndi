package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const jikeMobileBaseURL = "https://m.okjike.com"

// jikeCST formats display dates in China Standard Time.
var jikeCST = time.FixedZone("CST", 8*3600)

// jikeJSONScriptRe extracts the first embedded application/json script block
// from Jike's server-rendered pages.
var jikeJSONScriptRe = regexp.MustCompile(`(?s)<script[^>]*type="application/json"[^>]*>(.*?)</script>`)

// jikeThumbnailRe strips image thumbnail processing parameters.
var jikeThumbnailRe = regexp.MustCompile(`/thumbnail/.+`)

// jikeTrailingBreaksRe removes trailing line breaks and whitespace.
var jikeTrailingBreaksRe = regexp.MustCompile(`(?:<br/>|\s)+$`)

func jikeProfile(pageURL string) *disguise.Profile {
	return disguise.Chrome().Lang("zh-CN,zh;q=0.9").Referer(pageURL)
}

// Routes lists all Jike route specs in this package.
var Routes = []routeutils.RouteSpec{
	jikeTopicRoute,
	jikeTopicTextRoute,
	jikeUserRoute,
}

var jikeTopicRoute = routeutils.RouteSpec{
	Path:        "topic/:id",
	Name:        "Jike Topic",
	Example:     "jike/topic/556688fae4b00c57d9dd46ee",
	Maintainers: []string{"xihale"},
	Description: "Posts of a Jike (即刻) topic, including text, pictures and linked media. Topic id comes from the share URL m.okjike.com/topics/:id",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Topic id from the topic page URL, e.g. 553870e8e4b0cafb0a1bef68"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  JikeTopicHandler,
}

var jikeTopicTextRoute = routeutils.RouteSpec{
	Path:        "topic/text/:id",
	Name:        "Jike Topic Text Only",
	Example:     "jike/topic/text/553870e8e4b0cafb0a1bef68",
	Maintainers: []string{"xihale"},
	Description: "Text-only posts of a Jike (即刻) topic, ideal for daily news digest topics",
	Categories:  []models.Category{{Name: "social-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "Topic id from the topic page URL, e.g. 553870e8e4b0cafb0a1bef68"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  JikeTopicTextHandler,
}

var jikeUserRoute = routeutils.RouteSpec{
	Path:        "user/:id",
	Name:        "Jike User Timeline",
	Example:     "jike/user/3EE02BC9-C5B3-4209-8750-4ED1EE0F67BB",
	Maintainers: []string{"xihale"},
	Description: "Timeline of a Jike (即刻) user including original posts and reposts. User id comes from the personal page URL m.okjike.com/users/:id",
	Categories:  []models.Category{{Name: "social-media"}},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "User id from the user profile page URL"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  JikeUserHandler,
}

// JikeTopicHandler handles /jike/topic/:id.
func JikeTopicHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	props, err := jikeFetchPageProps(c.Parent(), c.Client(), jikeMobileBaseURL+"/topics/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	if len(props.Posts) == 0 {
		return nil, fmt.Errorf("topic %s does not exist or has no posts", id)
	}
	topicName := props.Topic.Content
	if topicName == "" {
		topicName = "即刻圈子 " + id
	}
	feed := routeutils.NewFeed(
		topicName+" - 即刻圈子",
		jikeMobileBaseURL+"/topics/"+id,
		props.Topic.BriefIntro,
	)
	routeutils.AppendMappedItems(feed, props.Posts, 0, func(p jikePost) *models.Item {
		return jikeBuildPostItem(p)
	})
	return feed, nil
}

// JikeTopicTextHandler handles /jike/topic/text/:id.
func JikeTopicTextHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	props, err := jikeFetchPageProps(c.Parent(), c.Client(), jikeMobileBaseURL+"/topics/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	if len(props.Posts) == 0 {
		return nil, fmt.Errorf("topic %s does not exist or has no posts", id)
	}
	topicName := props.Topic.Content
	if topicName == "" {
		topicName = "即刻圈子 " + id
	}
	feed := routeutils.NewFeed(
		topicName+" - 即刻圈子",
		jikeMobileBaseURL+"/topics/"+id,
		props.Topic.BriefIntro,
	)
	routeutils.AppendMappedItems(feed, props.Posts, 0, func(p jikePost) *models.Item {
		content := strings.TrimSpace(p.Content)
		if content == "" || p.ID == "" {
			return nil
		}
		title := fmt.Sprintf("%s %s", topicName, p.CreatedAt.In(jikeCST).Format("01月02日"))
		link := jikePostLink(p)
		item := routeutils.NewItem(title, link,
			strings.ReplaceAll(html.EscapeString(content), "\n", "<br/>"), p.CreatedAt)
		item.GUID = link
		return item
	})
	return feed, nil
}

// JikeUserHandler handles /jike/user/:id.
func JikeUserHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	pageURL := jikeMobileBaseURL + "/users/" + url.PathEscape(id)
	props, err := jikeFetchPageProps(c.Parent(), c.Client(), pageURL)
	if err != nil {
		return nil, err
	}
	if len(props.Posts) == 0 {
		return nil, fmt.Errorf("user %s does not exist or has no posts", id)
	}
	screenName := ""
	bio := ""
	if props.User != nil {
		screenName = props.User.ScreenName
		bio = props.User.Bio
	}
	if screenName == "" {
		screenName = "即刻用户"
	}
	feed := routeutils.NewFeed(screenName+"的即刻动态", pageURL, bio)
	routeutils.AppendMappedItems(feed, props.Posts, 0, func(p jikePost) *models.Item {
		return jikeBuildUserItem(id, p)
	})
	return feed, nil
}

// jikeFetchPageProps loads a mobile page and unmarshals its embedded JSON state.
func jikeFetchPageProps(ctx context.Context, cl *client.Client, pageURL string) (*jikePageProps, error) {
	body, err := jikeProfile(pageURL).Fetch(pageURL).GetString(ctx, cl)
	if err != nil {
		return nil, err
	}
	m := jikeJSONScriptRe.FindStringSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("embedded JSON not found on %s; page layout may have changed", pageURL)
	}
	var props struct {
		Props struct {
			PageProps jikePageProps `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal([]byte(m[1]), &props); err != nil {
		return nil, fmt.Errorf("failed to parse embedded JSON of %s: %w", pageURL, err)
	}
	return &props.Props.PageProps, nil
}

// jikePostLink returns the canonical external link of a post. WeChat articles
// captured by Jike link out to mp.weixin.qq.com directly.
func jikePostLink(p jikePost) string {
	link := fmt.Sprintf("%s/originalPosts/%s", jikeMobileBaseURL, p.ID)
	if p.LinkInfo != nil {
		ext := p.LinkInfo.OriginalLinkURL
		if ext == "" {
			ext = p.LinkInfo.LinkURL
		}
		if u, err := url.Parse(ext); err == nil && u.Host == "mp.weixin.qq.com" {
			return ext
		}
	}
	return link
}

// jikeBuildPostItem maps a topic post to a feed item with full media markup.
func jikeBuildPostItem(p jikePost) *models.Item {
	content := strings.TrimSpace(p.Content)
	linkInfoBlock := ""
	audioName := ""
	videoName := ""

	if li := p.LinkInfo; li != nil && li.LinkURL != "" {
		linkURL := li.LinkURL
		if li.OriginalLinkURL != "" {
			linkURL = li.OriginalLinkURL
		}
		audio := li.Audio
		if audio == nil {
			audio = p.Audio
		}
		video := li.Video
		if video == nil {
			video = p.Video
		}
		if audio != nil && audio.Title != "" {
			img := ""
			if audio.Image != nil {
				img = jikeFirstStr(audio.Image.PicURL, audio.Image.ThumbnailURL)
			}
			audioName = fmt.Sprintf("%s - %s", audio.Title, audio.Author)
			if img != "" {
				linkInfoBlock += fmt.Sprintf(`<img src="%s"/>`, html.EscapeString(img))
			}
			linkInfoBlock += fmt.Sprintf(`<p><a href="%s">%s</a></p>`,
				html.EscapeString(linkURL), html.EscapeString(audioName))
		} else if video != nil && video.URL != "" {
			img := ""
			if video.Image != nil {
				img = jikeFirstStr(video.Image.PicURL, video.Image.ThumbnailURL)
			}
			minutes := video.Duration.Int64() / 60000
			videoName = li.Title
			if videoName == "" {
				videoName = video.Title
			}
			label := videoName
			if label == "" {
				label = "观看视频"
			}
			if img != "" {
				linkInfoBlock += fmt.Sprintf(`<img src="%s"/>`, html.EscapeString(img))
			}
			linkInfoBlock += fmt.Sprintf(`<p><a href="%s">%s - 约%d分钟</a></p>`,
				html.EscapeString(linkURL), html.EscapeString(label), minutes)
		} else if linkURL != "" {
			label := li.Title
			if label == "" {
				label = "访问原文"
			}
			if li.PictureURL != "" {
				linkInfoBlock += fmt.Sprintf(`<img src="%s"/>`, html.EscapeString(li.PictureURL))
			}
			linkInfoBlock += fmt.Sprintf(`<p><a href="%s">%s</a></p>`,
				html.EscapeString(linkURL), html.EscapeString(label))
		}
	}

	description := linkInfoBlock
	bodyHTML := strings.ReplaceAll(html.EscapeString(content), "\n", "<br/>")
	description += bodyHTML

	for _, pic := range p.Pictures {
		imgURL := pic.PicURL
		if idx := strings.Index(imgURL, "?imageMogr2/"); idx >= 0 {
			imgURL = imgURL[:idx]
		} else if pic.Format != "gif" {
			imgURL = jikeThumbnailRe.ReplaceAllString(imgURL, "")
		}
		description += fmt.Sprintf(`<br/><img src="%s" referrerpolicy="no-referrer"/>`, html.EscapeString(imgURL))
	}
	if p.Video != nil && p.Video.URL != "" && p.LinkInfo == nil {
		description += fmt.Sprintf(`<br/><video src="%s" controls></video>`, html.EscapeString(p.Video.URL))
	}

	title := audioName
	if title == "" {
		title = videoName
	}
	if title == "" {
		title = collapseNewlines(content)
	}
	if title == "" && p.LinkInfo != nil {
		title = p.LinkInfo.Title
	}
	if title == "" {
		title = "无题"
	}

	link := jikePostLink(p)
	item := routeutils.NewItem(title, link, description, p.CreatedAt)
	item.GUID = link
	if p.User != nil && p.User.ScreenName != "" {
		routeutils.SetAuthor(item, p.User.ScreenName,
			routeutils.WithAuthorURI(jikeMobileBaseURL+"/users/"+p.User.Username))
	}
	return item
}

// jikeBuildUserItem maps a user timeline entry to a feed item.
func jikeBuildUserItem(ownerID string, p jikePost) *models.Item {
	typeMap := map[string]string{
		"ORIGINAL_POST":   "发布",
		"REPOST":          "转发",
		"ANSWER":          "回答",
		"QUESTION":        "提问",
		"PERSONAL_UPDATE": "创建新主题",
	}
	verb := typeMap[p.Type]
	if verb == "" {
		verb = "发布"
	}

	linkTemplate := ""
	if p.LinkInfo != nil && p.LinkInfo.LinkURL != "" {
		linkTemplate = fmt.Sprintf(`<p><a href="%s">%s</a></p>`,
			html.EscapeString(p.LinkInfo.LinkURL), html.EscapeString(jikeFirstStr(p.LinkInfo.Title, "访问原文")))
	}

	content := p.Content
	if content == "" && p.LinkInfo != nil {
		content = p.LinkInfo.Title
	}
	if content == "" && p.Question != nil {
		content = p.Question.Title
	}
	if content == "" {
		content = p.Title
	}
	contentHTML := strings.ReplaceAll(html.EscapeString(content), "\r\n", "<br/>")
	contentHTML = strings.ReplaceAll(contentHTML, "\n", "<br/>")
	contentHTML = strings.ReplaceAll(contentHTML, "\r", "<br/>")

	shortenTitle := "一条动态"
	if content != "" {
		shortenTitle = regexp.MustCompile(`(<br/>)+`).ReplaceAllString(contentHTML, " ")
		shortenTitle = strings.TrimSpace(shortenTitle)
	}

	if p.Type == "REPOST" && p.Target != nil {
		target := p.Target
		screenNameAnchor := ""
		if target.User != nil {
			name := html.EscapeString(target.User.ScreenName)
			uri := html.EscapeString(jikeMobileBaseURL + "/users/" + target.User.Username)
			screenNameAnchor = fmt.Sprintf(`<a href="%s">@%s</a>`, uri, name)
		}
		repostBody := strings.ReplaceAll(html.EscapeString(target.Content), "\n", "<br/>")
		for _, pic := range target.Pictures {
			repostBody += fmt.Sprintf(`<br/><img src="%s" referrerpolicy="no-referrer"/>`,
				html.EscapeString(jikeFirstStr(pic.ThumbnailURL, pic.PicURL)))
		}
		contentHTML += fmt.Sprintf("<blockquote>转发 %s: %s</blockquote>", screenNameAnchor, repostBody)
	}

	description := contentHTML + "<br/><br/>" + linkTemplate
	for _, pic := range p.Pictures {
		description += fmt.Sprintf(`<br/><img src="%s" referrerpolicy="no-referrer"/>`, html.EscapeString(pic.PicURL))
	}
	description = jikeTrailingBreaksRe.ReplaceAllString(description, "")

	title := fmt.Sprintf("%s了: %s", verb, shortenTitle)

	link := jikePostLink(p)
	switch p.Type {
	case "REPOST":
		link = fmt.Sprintf("%s/reposts/%s", jikeMobileBaseURL, p.ID)
	case "MEDIUM":
		link = "https://www.okjike.com/medium/" + p.ID
	}
	item := routeutils.NewItem(title, link, description, p.CreatedAt)
	item.GUID = link
	if p.User != nil && p.User.ScreenName != "" {
		routeutils.SetAuthor(item, p.User.ScreenName,
			routeutils.WithAuthorURI(jikeMobileBaseURL+"/users/"+p.User.Username))
	}
	return item
}

// jikeFirstStr returns the first non-empty string.
func jikeFirstStr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// collapseNewlines turns newlines into single spaces for use inside a title.
func collapseNewlines(s string) string {
	fields := strings.Fields(strings.ReplaceAll(s, "\u00a0", " "))
	return strings.Join(fields, " ")
}

type jikePageProps struct {
	Posts []jikePost `json:"posts"`
	Topic jikeTopic  `json:"topic"`
	User  *jikeUser  `json:"user"`
}

type jikeTopic struct {
	ID            string       `json:"id"`
	Content       string       `json:"content"`
	BriefIntro    string       `json:"briefIntro"`
	SquarePicture jikeImageSet `json:"squarePicture"`
}

type jikeUser struct {
	Username    string       `json:"username"`
	ScreenName  string       `json:"screenName"`
	Bio         string       `json:"bio"`
	AvatarImage jikeImageSet `json:"avatarImage"`
}

type jikeImageSet struct {
	PicURL       string `json:"picUrl"`
	MiddlePicURL string `json:"middlePicUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type jikePost struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"`
	Content   string        `json:"content"`
	Title     string        `json:"title"`
	CreatedAt time.Time     `json:"createdAt"`
	User      *jikeUser     `json:"user"`
	Pictures  []jikePicture `json:"pictures"`
	LinkInfo  *jikeLinkInfo `json:"linkInfo"`
	Video     *jikeVideo    `json:"video"`
	Audio     *jikeAudio    `json:"audio"`
	Question  *jikeQuestion `json:"question"`
	Target    *jikePost     `json:"target"`
}

type jikePicture struct {
	PicURL       string `json:"picUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Format       string `json:"format"`
}

type jikeLinkInfo struct {
	LinkURL         string     `json:"linkUrl"`
	OriginalLinkURL string     `json:"originalLinkUrl"`
	Title           string     `json:"title"`
	PictureURL      string     `json:"pictureUrl"`
	Video           *jikeVideo `json:"video"`
	Audio           *jikeAudio `json:"audio"`
}

type jikeMediaImage struct {
	PicURL       string `json:"picUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type jikeAudio struct {
	Title  string          `json:"title"`
	Author string          `json:"author"`
	Image  *jikeMediaImage `json:"image"`
}

type jikeVideo struct {
	URL      string          `json:"url"`
	Title    string          `json:"title"`
	Duration jkFlexInt64     `json:"duration"`
	Image    *jikeMediaImage `json:"image"`
}

type jikeQuestion struct {
	Title string `json:"title"`
}
