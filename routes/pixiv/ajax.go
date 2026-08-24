package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xihale/syndi/internal/client"
)

// pixivAjaxResp is the shared envelope of /ajax/* endpoints.
type pixivAjaxResp struct {
	Error   bool            `json:"error"`
	Message string          `json:"message"`
	Body    json.RawMessage `json:"body"`
}

// pixivIllustDetail is the payload of /ajax/illust/{id}.
type pixivIllustDetail struct {
	IllustID      string    `json:"illustId"`
	Title         string    `json:"illustTitle"`
	Comment       string    `json:"illustComment"` // caption HTML
	UserID        string    `json:"userId"`
	UserName      string    `json:"userName"`
	CreateDate    pixivTime `json:"createDate"`
	PageCount     int       `json:"pageCount"`
	ViewCount     int       `json:"viewCount"`
	BookmarkCount int       `json:"bookmarkCount"`
	LikeCount     int       `json:"likeCount"`
	IllustType    int       `json:"illustType"`
	XRestrict     int       `json:"xRestrict"` // 1 = R-18, 2 = R-18G
	Tags          struct {
		Tags []struct {
			Tag string `json:"tag"`
		} `json:"tags"`
	} `json:"tags"`
}

// pixivTime is a tolerant RFC3339 timestamp ("2026-08-24T20:43:42+09:00");
// null/"" decode to the zero time instead of failing the document.
type pixivTime struct {
	time.Time
}

func (t *pixivTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// fetchPixivIllustDetails retrieves /ajax/illust/{id} for each id with a
// bounded worker pool. Failures degrade gracefully: failed ids stay absent
// from the returned map (keyed by input id).
func fetchPixivIllustDetails(ctx context.Context, cl *client.Client, ids []string) map[string]*pixivIllustDetail {
	const workers = 5
	out := make(map[string]*pixivIllustDetail, len(ids))
	if len(ids) == 0 {
		return out
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var resp pixivAjaxResp
			url := fmt.Sprintf("%s/ajax/illust/%s?lang=en", pixivBaseURL, id)
			if err := pixivProfile(pixivReferer).Fetch(url).GetJSON(ctx, cl, &resp); err != nil || resp.Error {
				return
			}
			var detail pixivIllustDetail
			if err := json.Unmarshal(resp.Body, &detail); err != nil || detail.IllustID == "" {
				return
			}
			mu.Lock()
			out[id] = &detail
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}

// orderedProfileIllustIDs extracts the artwork ids of body.illusts from a
// /ajax/user/{id}/profile/all payload in document order (newest first).
//
// encoding/json maps lose insertion order, so we walk raw decoder tokens and
// collect keys of the "illusts" object as they appear.
func orderedProfileIllustIDs(raw []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))

	wantObject := func(dec *json.Decoder) bool {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		delim, ok := tok.(json.Delim)
		return ok && delim == '{'
	}
	readKey := func(dec *json.Decoder) (string, bool) {
		tok, err := dec.Token()
		if err != nil {
			return "", false
		}
		key, ok := tok.(string)
		return key, ok
	}
	skipValue := func(dec *json.Decoder) error {
		var v json.RawMessage
		return dec.Decode(&v)
	}

	// top-level object
	if !wantObject(dec) {
		return nil, fmt.Errorf("pixiv: profile/all payload is not a JSON object")
	}
	inBody := false
	for dec.More() {
		key, ok := readKey(dec)
		if !ok {
			return nil, fmt.Errorf("pixiv: malformed profile/all payload")
		}
		switch {
		case key == "body" && !inBody:
			if !wantObject(dec) {
				return nil, fmt.Errorf("pixiv: profile/all body is not a JSON object")
			}
			inBody = true
		case key == "illusts" && inBody:
			if !wantObject(dec) {
				return nil, fmt.Errorf("pixiv: profile/all illusts is not a JSON object")
			}
			var ids []string
			for dec.More() {
				id, ok := readKey(dec)
				if !ok {
					return nil, fmt.Errorf("pixiv: malformed illusts map")
				}
				ids = append(ids, id)
				if err := skipValue(dec); err != nil { // value is null or an object
					return nil, err
				}
			}
			// closing '}'
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return ids, nil
		default:
			if err := skipValue(dec); err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}
