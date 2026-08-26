package routes

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// urlSegment percent-encodes a single path/query segment value.
func urlSegment(raw string) string {
	return url.QueryEscape(raw)
}

// --- bangumi/media ---

type bilibiliPgcMediaResp struct {
	biliResp
	Result struct {
		MediaID  int64  `json:"media_id"`
		SeasonID int64  `json:"season_id"`
		Title    string `json:"title"`
		Cover    string `json:"cover"`
		ShareURL string `json:"share_url"`
		Evaluate string `json:"evaluate"`
		TypeName string `json:"type_name"`
	} `json:"result"`
}

type bilibiliPgcEpisode struct {
	ID       int64  `json:"id"`
	Aid      int64  `json:"aid"`
	Bvid     string `json:"bvid"`
	Cid      int64  `json:"cid"`
	Title    string `json:"title"`
	LongFn   string `json:"long_title"`
	Cover    string `json:"cover"`
	ShareURL string `json:"share_url"`
}

type bilibiliPgcSectionResp struct {
	biliResp
	Result struct {
		MainSection struct {
			Episodes []bilibiliPgcEpisode `json:"episodes"`
		} `json:"main_section"`
		Section []struct {
			Title    string               `json:"title"`
			Episodes []bilibiliPgcEpisode `json:"episodes"`
		} `json:"section"`
	} `json:"result"`
}

// BilibiliBangumiMediaHandler handles /bilibili/bangumi/media/:mediaid/:embed?
// Media metadata via pgc/view/web/media; episodes via pgc/web/season/section.
func BilibiliBangumiMediaHandler(c *ctxpkg.Context) (*models.Feed, error) {
	mediaID := strings.TrimSpace(c.Param("mediaid"))
	mediaID = strings.TrimPrefix(strings.TrimPrefix(mediaID, "md"), "ss")
	if mediaID == "" {
		return nil, fmt.Errorf("bilibili: missing media id")
	}
	if _, err := strconv.ParseInt(mediaID, 10, 64); err != nil {
		return nil, fmt.Errorf("bilibili: invalid media id %q", c.Param("mediaid"))
	}
	embed := bilibiliEmbedEnabled(c.Param("embed"))
	ctx := c.Parent()

	var media bilibiliPgcMediaResp
	if err := bilibiliJSONProfile().Fetch("https://api.bilibili.com/pgc/view/web/media?media_id="+urlSegment(mediaID)).
		GetJSON(ctx, c.Client(), &media); err != nil {
		return nil, err
	}
	if err := media.Err(); err != nil {
		// pgc/view/web/media is intermittent for some regions; fall back to
		// the review endpoint which carries title/season/share/cover.
		var fallback bilibiliPgcMediaResp
		if ferr := bilibiliJSONProfile().Fetch("https://api.bilibili.com/pgc/review/user?media_id="+urlSegment(mediaID)).
			GetJSON(ctx, c.Client(), &fallback); ferr == nil && fallback.Err() == nil && fallback.Result.SeasonID != 0 {
			media = fallback
		} else {
			return nil, media.Err()
		}
	}
	if media.Result.ShareURL == "" {
		media.Result.ShareURL = fmt.Sprintf("https://www.bilibili.com/bangumi/media/md%d", media.Result.MediaID)
	}

	var season bilibiliPgcSectionResp
	apiURL := "https://api.bilibili.com/pgc/web/season/section?season_id=" + strconv.FormatInt(media.Result.SeasonID, 10)
	if err := bilibiliJSONProfile().
		Referer(media.Result.ShareURL).
		Fetch(apiURL).GetJSON(ctx, c.Client(), &season); err != nil {
		return nil, err
	}
	if err := season.Err(); err != nil {
		return nil, err
	}
	return bilibiliBangumiFeed(&media, &season, embed), nil
}

// bilibiliBangumiFeed renders the episode list for a media/section pair.
func bilibiliBangumiFeed(media *bilibiliPgcMediaResp, season *bilibiliPgcSectionResp, embed bool) *models.Feed {
	feed := routeutils.NewFeed(media.Result.Title, media.Result.ShareURL, html.EscapeString(media.Result.Evaluate))
	episodes := append([]bilibiliPgcEpisode{}, season.Result.MainSection.Episodes...)
	for _, sec := range season.Result.Section {
		episodes = append(episodes, sec.Episodes...)
	}
	routeutils.AppendMappedItems(feed, episodes, 0, func(e bilibiliPgcEpisode) *models.Item {
		title := fmt.Sprintf("第%s话 %s", e.Title, e.LongFn)
		desc := renderBilibiliOGVDescription(embed, bilibiliFeedImage(e.Cover), html.EscapeString(e.LongFn),
			media.Result.SeasonID, e.ID)
		item := routeutils.NewItem(title, e.ShareURL, desc, time.Time{})
		if item == nil {
			return nil
		}
		item.GUID = fmt.Sprintf("bilibili-bangumi-%d-%d", media.Result.SeasonID, e.ID)
		return item
	})
	return feed
}

// renderBilibiliOGVDescription mirrors upstream renderOGVDescription.
func renderBilibiliOGVDescription(embed bool, img, description string, seasonID int64, episodeID int64) string {
	var b strings.Builder
	if embed && seasonID != 0 && episodeID != 0 {
		b.WriteString(fmt.Sprintf(
			`<iframe width="640" height="360" src="https://www.bilibili.com/blackboard/html5mobileplayer.html?seasonId=%d&amp;episodeId=%d" frameborder="0" allowfullscreen></iframe><br/>`,
			seasonID, episodeID))
	}
	if img != "" {
		b.WriteString(fmt.Sprintf(`<img src="%s"/><br/>`, html.EscapeString(img)))
	}
	b.WriteString(description)
	return b.String()
}

// --- audio (歌单) ---

type bilibiliAudioMenuResp struct {
	biliResp
	Data struct {
		MenuID int64  `json:"menuId"`
		Title  string `json:"title"`
		Intro  string `json:"intro"`
		Uname  string `json:"uname"`
	} `json:"data"`
}

type bilibiliAudioSong struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Uno       string `json:"uname"`
	Intro     string `json:"intro"`
	Cover     string `json:"cover"`
	Passtime  int64  `json:"passtime"`
	Duration  int64  `json:"duration"`
	Statistic struct {
		SID int64 `json:"sid"`
	} `json:"statistic"`
}

type bilibiliAudioSongsResp struct {
	biliResp
	Data struct {
		TotalSize int64               `json:"totalSize"`
		Data      []bilibiliAudioSong `json:"data"`
	} `json:"data"`
}

const bilibiliAudioBaseURL = "https://www.bilibili.com/audio"

// BilibiliAudioHandler handles /bilibili/audio/:id (audio menu songs).
func BilibiliAudioHandler(c *ctxpkg.Context) (*models.Feed, error) {
	sid := c.Param("id")
	if _, err := strconv.ParseInt(sid, 10, 64); err != nil {
		return nil, fmt.Errorf("bilibili: invalid audio menu id %q", sid)
	}
	ctx := c.Parent()

	var menu bilibiliAudioMenuResp
	if err := bilibiliJSONProfile().Fetch("https://www.bilibili.com/audio/music-service-c/web/menu/info?sid="+urlSegment(sid)).
		GetJSON(ctx, c.Client(), &menu); err != nil {
		return nil, err
	}
	if err := menu.Err(); err != nil {
		return nil, err
	}

	var songs bilibiliAudioSongsResp
	if err := bilibiliJSONProfile().Fetch(fmt.Sprintf(
		"https://www.bilibili.com/audio/music-service-c/web/song/of-menu?sid=%s&pn=1&ps=100", urlSegment(sid))).
		GetJSON(ctx, c.Client(), &songs); err != nil {
		return nil, err
	}
	if err := songs.Err(); err != nil {
		return nil, err
	}
	return bilibiliAudioFeed(&menu, &songs), nil
}

// bilibiliAudioFeed renders the song list of an audio menu.
func bilibiliAudioFeed(menu *bilibiliAudioMenuResp, songs *bilibiliAudioSongsResp) *models.Feed {
	link := fmt.Sprintf("%s/am%d", bilibiliAudioBaseURL, menu.Data.MenuID)
	feed := routeutils.NewFeed(menu.Data.Title, link, html.EscapeString(menu.Data.Intro))
	routeutils.SetFeedAuthor(feed, menu.Data.Uname)
	routeutils.AppendMappedItems(feed, songs.Data.Data, 0, mapBilibiliAudioSong)
	return feed
}

// mapBilibiliAudioSong maps one song record into a feed item.
func mapBilibiliAudioSong(s bilibiliAudioSong) *models.Item {
	if s.Title == "" || s.ID == 0 {
		return nil
	}
	songSID := s.Statistic.SID
	if songSID == 0 {
		songSID = s.ID
	}
	desc := html.EscapeString(s.Intro)
	if s.Cover != "" {
		desc += fmt.Sprintf(`<br/><img src="%s">`, html.EscapeString(bilibiliFeedImage(s.Cover)))
	}
	item := routeutils.NewItem(s.Title,
		fmt.Sprintf("%s/au%d", bilibiliAudioBaseURL, songSID),
		desc, time.Unix(s.Passtime, 0))
	if item == nil {
		return nil
	}
	item.GUID = fmt.Sprintf("bilibili-audio-%d", songSID)
	if author := firstNonEmpty(s.Author, s.Uno); author != "" {
		routeutils.SetAuthor(item, author)
	}
	return item
}

// --- readlist (专栏文集) ---

type bilibiliReadlistResp struct {
	biliResp
	Data struct {
		List struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			ImageURL string `json:"image_url"`
			Summary  string `json:"summary"`
		} `json:"list"`
		Author struct {
			Name string `json:"name"`
			Face string `json:"face"`
		} `json:"author"`
		Articles []struct {
			ID          int64    `json:"id"`
			Title       string   `json:"title"`
			Summary     string   `json:"summary"`
			ImageURLs   []string `json:"image_urls"`
			PublishTime int64    `json:"publish_time"`
		} `json:"articles"`
	} `json:"data"`
}

// BilibiliReadlistHandler handles /bilibili/readlist/:listid.
func BilibiliReadlistHandler(c *ctxpkg.Context) (*models.Feed, error) {
	listID := c.Param("listid")
	if _, err := strconv.ParseInt(listID, 10, 64); err != nil {
		return nil, fmt.Errorf("bilibili: invalid readlist id %q", listID)
	}
	listURL := fmt.Sprintf("https://www.bilibili.com/read/readlist/rl%s", listID)

	var resp bilibiliReadlistResp
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/article/list/web/articles?id=%s&jsonp=jsonp", urlSegment(listID))
	if err := bilibiliJSONProfile().Referer(listURL).Fetch(apiURL).
		GetJSON(c.Parent(), c.Client(), &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return bilibiliReadlistFeed(listID, listURL, &resp), nil
}

// bilibiliReadlistFeed renders article items of a readlist payload.
func bilibiliReadlistFeed(listID, listURL string, resp *bilibiliReadlistResp) *models.Feed {
	feed := routeutils.NewFeed(
		fmt.Sprintf("bilibili 专栏文集 - %s", resp.Data.List.Name), listURL,
		html.EscapeString(firstNonEmpty(resp.Data.List.Summary, "作者很懒，还木有写简介.....((/- -)/")),
	)
	routeutils.SetFeedAuthor(feed, resp.Data.Author.Name)
	for _, a := range resp.Data.Articles {
		if a.Title == "" || a.ID == 0 {
			continue
		}
		summaryText := html.EscapeString(a.Summary)
		if a.Summary != "" {
			summaryText += "…"
		}
		if len(a.ImageURLs) > 0 && a.ImageURLs[0] != "" {
			summaryText += fmt.Sprintf(`<br><img src="%s">`, html.EscapeString(bilibiliFeedImage(a.ImageURLs[0])))
		}
		item := routeutils.NewItem(a.Title,
			fmt.Sprintf("https://www.bilibili.com/read/cv%d/?from=readlist", a.ID),
			summaryText, time.Unix(a.PublishTime, 0))
		if item == nil {
			continue
		}
		item.GUID = fmt.Sprintf("bilibili-readlist-%s-%d", listID, a.ID)
		if resp.Data.Author.Name != "" {
			routeutils.SetAuthor(item, resp.Data.Author.Name)
		}
		routeutils.AddItem(feed, item)
	}
	return feed
}
