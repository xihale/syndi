package routes

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const qqFactBaseURL = "https://vp.fact.qq.com"

// qqFactRequestToken mirrors the upstream JS: CryptoJS.DES.encrypt(
// `${Date.now()}-sgn51n6r6q97o6g3`, "jzhotdata") which is OpenSSL-style
// salted DES-CBC with an MD5-based EVP_BytesToKey derivation.
func qqFactRequestToken() string {
	salt := make([]byte, 8)
	_, _ = rand.Read(salt)
	key, iv := qqFactDeriveKeyIV([]byte("jzhotdata"), salt)
	block, err := des.NewCipher(key)
	if err != nil {
		return ""
	}
	plaintext := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), "sgn51n6r6q97o6g3")
	data := qqFactPKCS7Pad([]byte(plaintext), block.BlockSize())
	enc := make([]byte, len(data))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(enc, data)

	var buf bytes.Buffer
	buf.WriteString("Salted__")
	buf.Write(salt)
	buf.Write(enc)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// qqFactDeriveKeyIV implements OpenSSL EVP_BytesToKey with MD5 for DES
// (8-byte key + 8-byte IV).
func qqFactDeriveKeyIV(password, salt []byte) (key, iv []byte) {
	var out, prev []byte
	for len(out) < 16 {
		h := md5.New()
		h.Write(prev)
		h.Write(password)
		h.Write(salt)
		prev = h.Sum(nil)
		out = append(out, prev...)
	}
	return out[:8], out[8:16]
}

func qqFactPKCS7Pad(b []byte, blockSize int) []byte {
	pad := blockSize - len(b)%blockSize
	return append(b, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

type qqFactListResp struct {
	Code int `json:"code"`
	Data struct {
		List []qqFactItem `json:"list"`
	} `json:"data"`
}

type qqFactAuthor struct {
	Name string
	Desc string
}

// UnmarshalJSON tolerates Author being either an object or a plain string.
func (a *qqFactAuthor) UnmarshalJSON(data []byte) error {
	var obj struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && (obj.Name != "" || obj.Desc != "") {
		a.Name, a.Desc = obj.Name, obj.Desc
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.Name = strings.TrimSpace(s)
	}
	return nil
}

type qqFactItem struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Explain  string       `json:"explain"`
	Result   string       `json:"result"`
	Date     string       `json:"date"`
	Abstract string       `json:"abstract"`
	Cover    string       `json:"coverrect"`
	Tag      []string     `json:"tag"`
	Author   qqFactAuthor `json:"Author"`
}

var qqFactRoute = routeutils.RouteSpec{
	Path:        "fact",
	Name:        "QQ Fact Check",
	Example:     "qq/fact",
	Maintainers: []string{"xihale"},
	Description: "Tencent Fact Check (腾讯较真/辟谣) latest rumor-refuting articles",
	Categories:  []models.Category{{Name: "new-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.OptionalParam("limit", "Max items, default 10"),
	},
	CacheTTL: 60 * time.Minute,
	Handler:  QQFactHandler,
}

// QQFactHandler handles /qq/fact
func QQFactHandler(c *ctxpkg.Context) (*models.Feed, error) {
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 10, 30)

	token := qqFactRequestToken()
	apiURL := fmt.Sprintf("%s/api/article/list?page=1&locale=zh-CN&token=%s", qqFactBaseURL, url.QueryEscape(token))
	var resp qqFactListResp
	if err := disguise.Chrome().Lang("zh-CN,zh;q=0.9").
		Referer(qqFactBaseURL+"/home").JSONAccept().
		Fetch(apiURL).GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("qq fact api code %d", resp.Code)
	}

	feed := routeutils.NewFeed(
		"较真查证平台 - 腾讯新闻",
		qqFactBaseURL+"/home",
		"腾讯新闻较真平台最新辟谣",
	)

	n := 0
	for _, it := range resp.Data.List {
		if n >= limit {
			break
		}
		if it.Title == "" || it.ID == "" {
			continue
		}
		link := fmt.Sprintf("%s/article?id=%s", qqFactBaseURL, it.ID)

		var b strings.Builder
		if it.Cover != "" {
			b.WriteString(`<img src="` + html.EscapeString(qqFactAbsURL(it.Cover)) + `"/><br/>`)
		}
		if it.Abstract != "" {
			b.WriteString("<p>" + html.EscapeString(it.Abstract) + "</p>")
		}
		meta := make([]string, 0, 3)
		if it.Result != "" {
			meta = append(meta, "查证结果："+it.Result)
		}
		if len(it.Tag) > 0 {
			meta = append(meta, "标签："+strings.Join(it.Tag, "、"))
		}
		if len(meta) > 0 {
			b.WriteString("<p>" + html.EscapeString(strings.Join(meta, " | ")) + "</p>")
		}

		title := it.Title
		if it.Explain != "" {
			title = "【" + it.Explain + "】" + it.Title
		}
		item := routeutils.NewItem(title, link, b.String(), qqFactParseDate(it.Date))
		if item == nil {
			continue
		}
		item.GUID = it.ID
		if it.Author.Name != "" {
			author := it.Author.Name
			if it.Author.Desc != "" {
				author += " - " + it.Author.Desc
			}
			routeutils.SetItemAuthor(item, author, "", "")
		}
		routeutils.SetCategories(item, it.Tag...)
		routeutils.AddItem(feed, item)
		n++
	}
	return feed, nil
}

func qqFactAbsURL(u string) string {
	switch {
	case strings.HasPrefix(u, "//"):
		return "https:" + u
	case strings.HasPrefix(u, "http"):
		return u
	case u == "":
		return ""
	default:
		return "https://" + u
	}
}

func qqFactParseDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.FixedZone("CST", 8*60*60))
	if err != nil {
		return time.Time{}
	}
	return t
}
