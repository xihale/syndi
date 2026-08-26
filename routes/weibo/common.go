// Package routes ports the RSSHub weibo (微博) namespace.
//
// Authentication: every m.weibo.cn / weibo.com API endpoint answers the Sina
// Visitor System challenge (HTTP 432) without a logged-in session. Set the
// WEIBO_COOKIES environment variable to a browser cookie string, e.g.:
//
//	WEIBO_COOKIES="SUB=...; SUBP=..."
//
// The name matches upstream RSSHub's config so users can reuse the same value.
package routes

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	"github.com/xihale/syndi/pkg/models"
	"github.com/xihale/syndi/pkg/registry"
)

const weiboCookiesEnv = "WEIBO_COOKIES"

// weiboMobileAPIBase is the m.weibo.cn origin. Tests swap it for a fixture
// server; production code must treat it as read-only.
var weiboMobileAPIBase = "https://m.weibo.cn"

func init() {
	registry.RegisterNamespaceEnv("weibo", registry.EnvRequirement{
		Key:         weiboCookiesEnv,
		Description: "配置后解锁需登录的微博路由",
		Scope:       "全部路由（登录类）",
		Fields: []registry.EnvField{
			{Name: "SUB", Note: "浏览器登录微博后，在 DevTools → Cookies 里复制整串"},
		},
	})
}

// weiboCookies returns the configured login cookie string ("" when unset).
func weiboCookies() string {
	return strings.TrimSpace(os.Getenv(weiboCookiesEnv))
}

// requireWeiboCookies fails fast with an actionable message.
func requireWeiboCookies() error {
	if weiboCookies() == "" {
		return fmt.Errorf("微博该路由需要登录 Cookie：请设置环境变量 %s（如 %s=\"SUB=<值>; SUBP=<值>\"）后重试", weiboCookiesEnv, weiboCookiesEnv)
	}
	return nil
}

const weiboMobileUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"

// weiboAPIProfile builds an m.weibo.cn XHR-like request profile. The cookie is
// attached only when configured.
func weiboAPIProfile(referer string) *disguise.Profile {
	p := disguise.Custom(weiboMobileUA).
		JSONAccept().
		Lang("zh-CN,zh;q=0.9").
		WithHeader("MWeibo-Pwa", "1").
		WithHeader("X-Requested-With", "XMLHttpRequest").
		Referer(referer)
	if ck := weiboCookies(); ck != "" {
		p = p.Cookie(ck)
	}
	return p
}

// weiboIsLoginWallErr reports whether the request was rejected by the Sina
// Visitor System (HTTP 432). The HTTP client surfaces the status as a plain
// error string, so recognition happens on the message.
func weiboIsLoginWallErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status 432")
}

// weiboAuthError reports a login-wall (ok == -100) with an actionable hint
// that differs depending on whether WEIBO_COOKIES is configured at all.
func weiboAuthError(where string, ok int) error {
	if weiboCookies() == "" {
		return fmt.Errorf("weibo: %s 被微博登录墙拦截 (ok=%d)：请设置环境变量 %s 后重试", where, ok, weiboCookiesEnv)
	}
	return fmt.Errorf("weibo: %s 不可用 (ok=%d)：WEIBO_COOKIES 可能已过期，请更新", where, ok)
}

const weiboTitleMaxRunes = 60

// weiboTitleFromHTML derives a plain, length-capped item title from mblog HTML.
func weiboTitleFromHTML(htmlText string) string {
	title := plainWeiboText(htmlText)
	if len([]rune(title)) > weiboTitleMaxRunes {
		title = string([]rune(title)[:weiboTitleMaxRunes]) + "…"
	}
	return title
}

// mbStatusID returns a stable short identifier (bid preferred) for a post.
func mbStatusID(mb *weiboMBlog) string {
	if mb.Bid != "" {
		return mb.Bid
	}
	return mb.ID
}

// appendWeiboMBlogs maps mblogs to feed items shared by the keyword and
// super-index feeds. uid pins the post author for permalink building when
// known; guidPrefix namespaces item GUIDs.
func appendWeiboMBlogs(feed *models.Feed, uid, guidPrefix string, mblogs []*weiboMBlog) {
	for _, mb := range mblogs {
		if mb == nil || (mb.Text == "" && mb.Bid == "" && mb.ID == "") {
			continue
		}
		title := weiboTitleFromHTML(mb.Text)
		if title == "" {
			title = "微博动态 " + mbStatusID(mb)
		}
		item := routeutils.NewItem(title, weiboStatusLink(uid, mb), buildWeiboDesc(mb), parseWeiboDate(mb.CreatedAt))
		item.GUID = guidPrefix + mbStatusID(mb)
		if mb.User != nil && mb.User.ScreenName != "" {
			routeutils.SetItemAuthor(item, mb.User.ScreenName, "", "")
		}
		routeutils.AddItem(feed, item)
	}
}

var weiboRelativeRe = regexp.MustCompile(`^(\d+)\s*(分钟|小时|天)前$`)

// parseWeiboDate parses mblog created_at values. Weibo mixes absolute
// timestamps ("Sun Mar 10 12:00:00 +0800 2024") with relative ones
// ("刚刚", "5分钟前", "3小时前", "昨天"); relatives are resolved against
// fetch time because upstream encodes them that way.
func parseWeiboDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse("Mon Jan 2 15:04:05 -0700 2006", raw); err == nil {
		return t
	}
	switch raw {
	case "刚刚":
		return time.Now()
	case "昨天":
		return time.Now().AddDate(0, 0, -1)
	}
	if m := weiboRelativeRe.FindStringSubmatch(raw); m != nil {
		n := 0
		for _, ch := range m[1] {
			n = n*10 + int(ch-'0')
		}
		switch m[2] {
		case "分钟":
			return time.Now().Add(-time.Duration(n) * time.Minute)
		case "小时":
			return time.Now().Add(-time.Duration(n) * time.Hour)
		case "天":
			return time.Now().AddDate(0, 0, -n)
		}
	}
	return time.Time{}
}

// weiboTextCleanup normalizes mblog HTML: drop tracking anchors and fix image
// protocol-relative URLs so feeds render them.
func weiboTextCleanup(text string) string {
	text = strings.ReplaceAll(text, `\n`, "<br>")
	text = strings.ReplaceAll(text, `src="//`, `src="https://`)
	text = strings.ReplaceAll(text, `href="//`, `href="https://`)
	return text
}
