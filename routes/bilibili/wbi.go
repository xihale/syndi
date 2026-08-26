package routes

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ctxpkg "github.com/xihale/syndi/pkg/context"
)

// bilibiliMixinKeyTab is the fixed permutation table published by
// SocialSisterYi/bilibili-API-collect; it selects 32 chars from the
// concatenated img_key+sub_key of the nav API to build the signing key.
var bilibiliMixinKeyTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
	61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
	36, 20, 34, 44, 52,
}

type bilibiliNavResp struct {
	biliResp
	Data struct {
		WbiImg struct {
			ImgURL string `json:"img_url"`
			SubURL string `json:"sub_url"`
		} `json:"wbi_img"`
	} `json:"data"`
}

func bilibiliWbiKeyName(raw string) string {
	if i := strings.LastIndexByte(raw, '/'); i >= 0 {
		raw = raw[i+1:]
	}
	if i := strings.IndexByte(raw, '.'); i > 0 {
		raw = raw[:i]
	}
	return raw
}

// bilibiliWbiKeys fetches (and caches for ~6h) the current wbi img/sub keys.
func bilibiliWbiKeys(c *ctxpkg.Context) (string, string, error) {
	fetch := func() ([2]string, error) {
		var resp bilibiliNavResp
		if err := bilibiliJSONProfile().Fetch("https://api.bilibili.com/x/web-interface/nav").
			GetJSON(c.Parent(), c.Client(), &resp); err != nil {
			return [2]string{}, err
		}
		img := bilibiliWbiKeyName(resp.Data.WbiImg.ImgURL)
		sub := bilibiliWbiKeyName(resp.Data.WbiImg.SubURL)
		if img == "" || sub == "" {
			return [2]string{}, fmt.Errorf("bilibili: empty wbi keys from nav api")
		}
		return [2]string{img, sub}, nil
	}

	var pair [2]string
	var err error
	if c.Cache() == nil {
		pair, err = fetch()
	} else {
		var v interface{}
		v, err = c.CacheTryGet("bilibili-wbi-keys", 6*time.Hour, func() (interface{}, error) {
			return fetch()
		})
		if err == nil {
			var ok bool
			pair, ok = v.([2]string)
			if !ok {
				err = fmt.Errorf("bilibili: invalid cached wbi keys")
			}
		}
	}
	if err != nil {
		return "", "", err
	}
	return pair[0], pair[1], nil
}

// bilibiliSignWbi appends the w_rid/wts signature to params, mirroring
// upstream utils.addWbiVerifyInfo: md5(sorted(params)+"&wts="+ts+mixinKey).
func bilibiliSignWbi(params url.Values, imgKey, subKey string, nowSec int64) url.Values {
	mixinRaw := imgKey + subKey
	mixin := make([]byte, 0, 32)
	for _, idx := range bilibiliMixinKeyTab {
		if idx < len(mixinRaw) && len(mixin) < 32 {
			mixin = append(mixin, mixinRaw[idx])
		}
	}

	sorted := make([]string, 0, len(params))
	for k := range params {
		if k == "w_rid" || k == "wts" {
			continue
		}
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	base := url.Values{}
	for _, k := range sorted {
		base.Set(k, params.Get(k))
	}
	hashInput := base.Encode() + "&wts=" + strconv.FormatInt(nowSec, 10) + string(mixin)
	sum := md5.Sum([]byte(hashInput))

	out := url.Values{}
	for _, k := range sorted {
		out.Set(k, params.Get(k))
	}
	out.Set("w_rid", hex.EncodeToString(sum[:]))
	out.Set("wts", strconv.FormatInt(nowSec, 10))
	return out
}

// bilibiliDmImgList generates a plausible dm_img_list payload used by the
// space web client (randomized gaussian values like upstream utils).
func bilibiliDmImgList() string {
	x := gaussClamped(1245, 5)
	y := gaussClamped(1285, 5)
	return fmt.Sprintf(`[{"x":%d,"y":%d,"z":0,"timestamp":%d,"type":0}]`,
		3*x+2*y, 4*x-5*y, gaussClamped(30, 5))
}

// gaussClamped approximates upstream's box-muller gaussian integer.
func gaussClamped(mean, std int) int {
	v := int(math.Round(rand.NormFloat64()*float64(std) + float64(mean)))
	if v < 0 {
		return 0
	}
	return v
}
