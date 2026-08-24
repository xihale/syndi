// Package routes ports the RSSHub pixiv namespace.
//
// Authentication: pixiv web AJAX endpoints degrade unpredictably without a
// session (R-18 works filtered out, some regions get empty payloads), and
// upstream RSSHub likewise refuses to serve pixiv routes unauthenticated.
// Set the PIXIV_COOKIES environment variable to a browser cookie string,
// e.g.:
//
//	PIXIV_COOKIES="PHPSESSID=...; device_token=...; first_visit_datetime=..."
//
// The name matches upstream RSSHub's config spirit so users can reuse the
// same logged-in browser cookie. Every route fails fast with an actionable
// message when the variable is unset.
package routes

import (
	"fmt"
	"os"
	"strings"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/pkg/registry"
)

const pixivCookiesEnv = "PIXIV_COOKIES"

func init() {
	// Surface the credential requirement on the docs frontend so users can
	// see at a glance whether this deployment can serve pixiv routes.
	registry.RegisterNamespaceEnv("pixiv", registry.EnvRequirement{
		Key:         pixivCookiesEnv,
		Description: "配置后解锁 pixiv 路由（全部需要登录 Cookie）",
		Scope:       "全部路由（登录类）",
		Fields: []registry.EnvField{
			{Name: "PHPSESSID", Note: "浏览器登录 pixiv 后，在 DevTools → Cookies 里复制整串"},
		},
	})
}

// pixivCookies returns the configured login cookie string ("" when unset).
func pixivCookies() string {
	return strings.TrimSpace(os.Getenv(pixivCookiesEnv))
}

// requirePixivCookies fails fast with an actionable message.
func requirePixivCookies() error {
	if pixivCookies() == "" {
		return fmt.Errorf("pixiv 该路由需要登录 Cookie：请设置环境变量 %s（如 %s=\"PHPSESSID=<值>\"）后重试", pixivCookiesEnv, pixivCookiesEnv)
	}
	return nil
}

// A single stable desktop-Chrome UA: pixiv ties ajax sessions to fingerprint
// consistency, so unlike public-site profiles we never rotate agents.
const pixivUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// pixivProfile builds an XHR-like request profile for www.pixiv.net/ajax
// endpoints. The login cookie is attached only when configured.
func pixivProfile(referer string) *disguise.Profile {
	p := disguise.Custom(pixivUserAgent).
		JSONAccept().
		Lang("en-US,en;q=0.9,ja;q=0.8,zh-CN;q=0.7").
		Referer(referer)
	if ck := pixivCookies(); ck != "" {
		p = p.Cookie(ck)
	}
	return p
}

const (
	pixivBaseURL   = "https://www.pixiv.net"
	pixivReferer   = pixivBaseURL + "/"
	pixivImageHost = "https://i.pximg.net" // hotlink-blocked for feed readers
)

// pixivEmbedImageURL returns the public social-embed image for an artwork.
// i.pximg.net requires a same-site Referer that feed readers never send, so
// descriptions embed embed.pixiv.net instead, which serves anonymously and is
// exactly what pixiv itself uses for OGP cards.
func pixivEmbedImageURL(illustID string) string {
	return fmt.Sprintf("https://embed.pixiv.net/decorate.php?illust_id=%s&mode=sns-ogp", illustID)
}
