// Package routes ports the RSSHub zhihu (知乎) namespace.
//
// Authentication: most www.zhihu.com /api/v4 endpoints return 403 without a
// logged-in session. Set the ZHIHU_COOKIES environment variable to a browser
// cookie string containing at least z_c0, e.g.:
//
//	ZHIHU_COOKIES="z_c0=2|1:0|..."
//
// The name matches upstream RSSHub's config so users can reuse the same value.
// Routes that work anonymously (hotlist, daily) send the cookie only when set.
package routes

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/pkg/registry"
)

const zhihuCookiesEnv = "ZHIHU_COOKIES"

func init() {
	// Surface the credential requirement on the docs frontend so users can
	// see at a glance whether this deployment can serve login-gated routes.
	registry.RegisterNamespaceEnv("zhihu", registry.EnvRequirement{
		Key:         zhihuCookiesEnv,
		Description: "配置后解锁需登录的知乎路由",
		Scope:       "部分路由（登录类）",
		Fields: []registry.EnvField{
			{Name: "z_c0", Note: "浏览器登录知乎后，在 DevTools → Cookies 里复制"},
		},
	})
}

// zhihuInt64 accepts both JSON numbers and quoted strings. Zhihu serializes
// ids above 2^53 as strings to avoid JS precision loss, so every id field
// needs this wrapper.
type zhihuInt64 int64

func (n *zhihuInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*n = zhihuInt64(v)
	return nil
}

func (n zhihuInt64) Int64() int64 { return int64(n) }

func (n zhihuInt64) String() string { return strconv.FormatInt(int64(n), 10) }

// A single stable desktop-Chrome UA: logged-in zhihu sessions are sensitive to
// fingerprint changes, so unlike public-site profiles we never rotate agents.
const zhihuUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// zhihuCookies returns the configured login cookie string ("" when unset).
func zhihuCookies() string {
	return strings.TrimSpace(os.Getenv(zhihuCookiesEnv))
}

// requireZhihuCookies fails fast with an actionable message for routes that
// always need a logged-in session.
func requireZhihuCookies() error {
	if zhihuCookies() == "" {
		return fmt.Errorf("知乎该路由需要登录 Cookie：请设置环境变量 %s（至少包含 z_c0，如 %s=\"z_c0=<值>\"）后重试", zhihuCookiesEnv, zhihuCookiesEnv)
	}
	return nil
}

// zhihuProfile builds an XHR-like request profile for zhihu endpoints.
// The cookie is attached only when configured; anonymous endpoints stay clean.
func zhihuProfile(referer string) *disguise.Profile {
	p := disguise.Custom(zhihuUserAgent).
		JSONAccept().
		Lang("zh-CN,zh;q=0.9,en;q=0.8").
		Referer(referer)
	if ck := zhihuCookies(); ck != "" {
		p = p.Cookie(ck)
	}
	return p
}

// zhihuWebProfile is the HTML-page variant (document Accept header).
func zhihuWebProfile(referer string) *disguise.Profile {
	p := disguise.Custom(zhihuUserAgent).
		Lang("zh-CN,zh;q=0.9,en;q=0.8").
		Referer(referer)
	if ck := zhihuCookies(); ck != "" {
		p = p.Cookie(ck)
	}
	return p
}

var zhihuMCNLinkRe = regexp.MustCompile(`(?i)<a[^>]*data-draft-type="mcn-link-card"[^>]*>.*?</a>`)

// processZhihuContent cleans zhihu rich-text HTML for feed consumption,
// mirroring upstream's processImage helper:
//   - drops <noscript> and mcn-link-card promo anchors
//   - unwraps link.zhihu.com redirect targets
//   - promotes lazy-loaded image sources (data-actualsrc/data-original)
func processZhihuContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	content = zhihuMCNLinkRe.ReplaceAllString(content, "")

	doc, err := parser.LoadString(content)
	if err != nil {
		return content
	}

	doc.Find("noscript").Each(func(_ int, s *goquery.Selection) { _ = s.Remove() })

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		for _, prefix := range []string{"http://link.zhihu.com/?target=", "https://link.zhihu.com/?target="} {
			if strings.HasPrefix(href, prefix) {
				target := strings.TrimPrefix(href, prefix)
				if decoded, err := url.QueryUnescape(target); err == nil && decoded != "" {
					s.SetAttr("href", decoded)
				}
				break
			}
		}
	})

	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		src := ""
		if v, ok := s.Attr("data-actualsrc"); ok && v != "" {
			src = v
			s.RemoveAttr("data-actualsrc")
		} else if v, ok := s.Attr("data-original"); ok && v != "" {
			src = v
			s.RemoveAttr("data-original")
		}
		if src != "" {
			fixed := fixZhihuImageURL(src)
			s.SetAttr("src", fixed)
			s.RemoveAttr("width")
			s.RemoveAttr("height")
		} else if v, ok := s.Attr("src"); ok && v != "" {
			s.SetAttr("src", fixZhihuImageURL(v))
		}
	})

	// Return only the fragment inside <body>, without the document wrapper
	// goquery adds around parsed fragments.
	html, err := doc.Find("body").Html()
	if err != nil {
		return content
	}
	return strings.TrimSpace(html)
}

// fixZhihuImageURL strips resize/query suffixes so feeds embed full images.
func fixZhihuImageURL(raw string) string {
	if i := strings.Index(raw, "?"); i >= 0 {
		raw = raw[:i]
	}
	replacer := strings.NewReplacer("_b.jpg", ".jpg", "_r.jpg", ".jpg", "_720w.jpg", ".jpg")
	return replacer.Replace(raw)
}
