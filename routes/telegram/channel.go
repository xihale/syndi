package routes

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/disguise"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

const telegramBaseURL = "https://t.me"

// telegramChannelDataURL points at the public web preview of a channel.
// Tests swap it for a local fixture server. t.me rejects default tool
// User-Agents, so requests always go through the disguise profile below.
var telegramChannelDataURL = telegramBaseURL

var (
	telegramChannelPattern  = regexp.MustCompile(`^[a-zA-Z0-9_]{3,64}$`)
	telegramBackgroundURLRe = regexp.MustCompile(`url\('([^']*)'\)`)
)

// telegramRouteParamsRe recognizes the supported route-params syntax; any
// other tail is treated as a search query (upstream backward compatibility).
var telegramRouteParamsRe = regexp.MustCompile(`(^|&)(showLinkPreview|showMessageMedia|searchQuery)=`)

type telegramChannelOptions struct {
	showLinkPreview  bool   // embed the link-preview blockquote
	showMessageMedia bool   // embed photos/stickers/video thumbnails
	searchQuery      string // search inside the channel (?q=)
}

func parseTelegramRouteParams(raw string) telegramChannelOptions {
	opts := telegramChannelOptions{showLinkPreview: true, showMessageMedia: true}
	raw = strings.TrimPrefix(raw, "/")
	if raw == "" {
		return opts
	}
	if !telegramRouteParamsRe.MatchString(raw) {
		opts.searchQuery = raw
		return opts
	}
	for _, kv := range strings.Split(raw, "&") {
		key, value, _ := strings.Cut(kv, "=")
		switch key {
		case "showLinkPreview":
			opts.showLinkPreview = routeutils.ParseBool(value, true)
		case "showMessageMedia":
			opts.showMessageMedia = routeutils.ParseBool(value, true)
		case "searchQuery":
			opts.searchQuery = value
		}
	}
	return opts
}

var telegramChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:username",
	Name:        "Telegram Channel",
	Example:     "telegram/channel/durov",
	Maintainers: []string{"xihale"},
	Description: "Public preview posts of a Telegram channel via t.me",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "Channel username without @, e.g. durov"),
	},
	CacheTTL: 10 * time.Minute,
	Handler:  TelegramChannelHandler,
}

// telegramChannelParamsRoute serves the same feed with optional trailing route
// parameters (/channel/:username/:routeParams). Registered as a catch-all
// because Gin has no optional path segments.
var telegramChannelParamsRoute = routeutils.RouteSpec{
	Path:        "channel/:username/*routeParams",
	Name:        "Telegram Channel",
	Example:     "telegram/channel/durov/showLinkPreview=0&showMessageMedia=0",
	Maintainers: []string{"xihale"},
	Description: "Public preview posts of a Telegram channel with extra switches",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("username", "Channel username without @"),
		routeutils.OptionalParam("routeParams", `key=value pairs joined by '&': showLinkPreview (0/1, default 1) embed the link preview; showMessageMedia (0/1, default 1) embed photos/stickers/video thumbnails; searchQuery=<text> search the channel. Any unknown tail is treated as a search query`),
	},
	CacheTTL: 10 * time.Minute,
	Handler:  TelegramChannelHandler,
}

// TelegramChannelHandler handles /telegram/channel/:username[/*routeParams].
func TelegramChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	channel := strings.TrimPrefix(strings.TrimSpace(c.Param("username")), "@")
	if !telegramChannelPattern.MatchString(channel) {
		return nil, fmt.Errorf("invalid channel name %q", channel)
	}
	opts := parseTelegramRouteParams(c.Param("routeParams"))
	ctx := c.Parent()

	resourceURL := fmt.Sprintf("%s/s/%s", telegramChannelDataURL, channel)
	feedLink := fmt.Sprintf("%s/s/%s", telegramBaseURL, channel)
	if opts.searchQuery != "" {
		resourceURL += "?q=" + url.QueryEscape(opts.searchQuery)
		feedLink += "?q=" + url.QueryEscape(opts.searchQuery)
	}

	doc, err := fetchTelegramChannelPage(ctx, c.Client(), resourceURL)
	if err != nil {
		return nil, err
	}

	channelTitle := telegramChannelTitle(doc, channel)
	feedTitle := fmt.Sprintf("%s - Telegram Channel", channelTitle)
	if opts.searchQuery != "" {
		feedTitle = fmt.Sprintf("%q - %s", opts.searchQuery, feedTitle)
	}
	feed := routeutils.NewFeed(
		feedTitle,
		feedLink,
		strings.TrimSpace(doc.Text("div.tgme_channel_info_description")),
	)

	var wraps []*parser.Selection
	doc.FindSelector("div.tgme_widget_message_wrap").Each(func(_ int, wrap *parser.Selection) {
		wraps = append(wraps, wrap)
	})
	// The preview page lists messages newest first; feeds read oldest first.
	for i := len(wraps) - 1; i >= 0; i-- {
		routeutils.AddItem(feed, parseTelegramMessage(wraps[i], channel, channelTitle, opts))
	}

	return feed, nil
}

// fetchTelegramChannelPage loads the t.me web preview with a browser disguise
// and falls back to the telegram.me mirror when the primary host is
// unreachable from the instance network.
func fetchTelegramChannelPage(ctx context.Context, cl *client.Client, resourceURL string) (*parser.Document, error) {
	doc, err := disguise.Chrome().Fetch(resourceURL).GetHTML(ctx, cl)
	if err == nil {
		return doc, nil
	}
	mirror := strings.Replace(resourceURL, "://t.me/", "://telegram.me/", 1)
	if mirror == resourceURL {
		return nil, err
	}
	retryDoc, retryErr := disguise.Chrome().Fetch(mirror).GetHTML(ctx, cl)
	if retryErr != nil {
		return nil, err
	}
	return retryDoc, nil
}

// cleanTelegramFragment runs Telegram-authored message fragments through the
// shared sanitizer pipeline: scripts dropped, relative links resolved against
// the t.me base, referrer policy added to embedded media.
func cleanTelegramFragment(fragment string) string {
	cleaned, err := routeutils.CleanDescription(fragment, telegramBaseURL+"/", routeutils.CleanOptions{
		BaseURL:        telegramBaseURL + "/",
		ReferrerPolicy: true,
		RemoveScripts:  true,
		FixLazyImages:  true,
		ResolveLinks:   true,
	})
	if err != nil {
		return html.EscapeString(routeutils.ExtractText(fragment))
	}
	return cleaned
}

// telegramBackgroundImage extracts a background-image url('...') value.
func telegramBackgroundImage(style string) string {
	if m := telegramBackgroundURLRe.FindStringSubmatch(style); len(m) > 1 {
		return m[1]
	}
	return ""
}

func parseTelegramMessage(wrap *parser.Selection, channel, channelTitle string, opts telegramChannelOptions) *models.Item {
	msg := wrap.Find("div.tgme_widget_message")
	if msg.Length() == 0 {
		return nil
	}
	// Service notices ("channel photo updated", pinned banners) and the
	// "no posts found" placeholder are not real entries.
	if wrap.Find(".service_message").Length() > 0 || wrap.Find(".tme_no_messages_found").Length() > 0 {
		return nil
	}

	dataPost := msg.AttrOr("data-post", "")
	link := msg.Find("a.tgme_widget_message_date").AttrOr("href", "")
	if link == "" && dataPost != "" {
		link = fmt.Sprintf("%s/%s", telegramBaseURL, dataPost)
	}
	if link == "" {
		link = fmt.Sprintf("%s/%s", telegramBaseURL, channel)
	}

	var pubDate time.Time
	if datetime := msg.Find("time").AttrOr("datetime", ""); datetime != "" {
		if parsed, err := time.Parse(time.RFC3339, datetime); err == nil {
			pubDate = parsed
		}
	}

	messageTags := telegramMediaTags(msg)

	mediaHTML := ""
	if opts.showMessageMedia {
		mediaHTML = telegramMessageMedia(msg, link)
	}

	description := ""
	titleText := ""
	// Current t.me pages wrap partially-supported messages in
	// .media_supported_cont; plain posts keep the text as a direct bubble
	// child (mirrors upstream's dual selector).
	textSel := msg.Find(".media_supported_cont > .tgme_widget_message_text")
	if textSel.Length() == 0 {
		textSel = msg.Find("div.tgme_widget_message_bubble > div.tgme_widget_message_text")
	}
	if textSel.Length() > 0 {
		textSel = textSel.First()
		if fragment, err := textSel.Html(); err == nil && strings.TrimSpace(fragment) != "" {
			description += "<p>" + cleanTelegramFragment(strings.TrimSpace(fragment)) + "</p>"
		}
		titleText = truncateText(extractPlainText(textSel.TextTrim()), 80)
	}

	title := titleText
	if title == "" {
		title = messageTags
	}
	if messageTags != "" && titleText != "" && opts.showMessageMedia {
		title = messageTags + " " + titleText
	}
	if title == "" && !pubDate.IsZero() {
		title = pubDate.UTC().Format("2006-01-02 15:04")
	}
	if title == "" {
		return nil
	}

	if opts.showLinkPreview {
		description += telegramLinkPreview(msg)
	}
	description += mediaHTML
	if description == "" {
		return nil
	}

	item := routeutils.NewItem(title, link, description, pubDate)
	if dataPost != "" {
		item.GUID = "telegram-channel-" + dataPost
	} else {
		item.GUID = link
	}

	author := msg.Find(".tgme_widget_message_from_author").First().TextTrim()
	if author == "" {
		author = channelTitle
	}
	routeutils.SetAuthor(item, author, routeutils.WithAuthorURI(telegramBaseURL+"/"+channel))
	return item
}

// telegramMediaTags renders short labels for the media kinds carried by a
// message ([Photo]/[Video]/...), mirroring upstream's media tags.
func telegramMediaTags(msg *parser.Selection) string {
	var sb strings.Builder
	add := func(cond bool, tag string) {
		if cond {
			sb.WriteString(tag)
		}
	}
	add(msg.Find(".tgme_widget_message_photo,.tgme_widget_message_photo_wrap").Length() > 0, "[Photo]")
	add(msg.Find(".tgme_widget_message_video_player").Length() > 0, "[Video]")
	add(msg.Find("[data-webp],.tgme_widget_message_sticker").Length() > 0, "[Sticker]")
	add(msg.Find("audio.tgme_widget_message_voice").Length() > 0, "[Voice]")
	add(msg.Find(".tgme_widget_message_document").Length() > 0, "[Document]")
	return sb.String()
}

// telegramMessageMedia builds <img> markup for photos, stickers and video
// posters. t.me exposes them as CSS background images; only URLs harvested
// from attributes are re-emitted, always escaped.
func telegramMessageMedia(msg *parser.Selection, postLink string) string {
	var sb strings.Builder
	appendImage := func(src, href string) {
		src = html.EscapeString(src)
		href = html.EscapeString(href)
		sb.WriteString(fmt.Sprintf(`<a href="%s"><img src="%s" referrerpolicy="no-referrer"></a>`, href, src))
	}

	// Photos / album items render as anchors with a background image.
	msg.Find("a.tgme_widget_message_photo_wrap").Each(func(_ int, photo *parser.Selection) {
		src := telegramBackgroundImage(photo.AttrOr("style", ""))
		if src == "" {
			return
		}
		href := photo.AttrOr("href", postLink)
		appendImage(src, href)
	})

	// WebP stickers carry their URL in an attribute.
	msg.Find("[data-webp]").Each(func(_ int, sticker *parser.Selection) {
		src := sticker.AttrOr("data-webp", "")
		if src == "" {
			return
		}
		appendImage(src, postLink)
	})

	// Videos: only the poster frame exists in static HTML, link back to post.
	msg.Find(".tgme_widget_message_video_player").Each(func(_ int, player *parser.Selection) {
		thumb := telegramBackgroundImage(player.Find(".tgme_widget_message_video_thumb").AttrOr("style", ""))
		if thumb == "" {
			return
		}
		appendImage(thumb, postLink)
	})

	return sb.String()
}

// telegramLinkPreview rebuilds the link-preview card attached to messages.
// All text passes through html.EscapeString and the image URL comes from the
// background-image attribute.
func telegramLinkPreview(msg *parser.Selection) string {
	container := msg.Find("a.tgme_widget_message_link_preview").First()
	site := msg.Find(".link_preview_site_name").First().TextTrim()
	titleNode := msg.Find(".link_preview_title").First()
	desc := msg.Find(".link_preview_description").First().TextTrim()

	href := container.AttrOr("href", "")
	imageSrc := ""
	for _, sel := range []string{".link_preview_image", ".link_preview_right_image"} {
		img := msg.Find(sel).First()
		if img.Length() > 0 {
			imageSrc = telegramBackgroundImage(img.AttrOr("style", ""))
			break
		}
	}

	if site == "" && titleNode.Length() == 0 && desc == "" && imageSrc == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<blockquote>")
	if site != "" {
		fmt.Fprintf(&sb, "<b>%s</b><br>", html.EscapeString(site))
	}
	if titleNode.Length() > 0 {
		title := html.EscapeString(titleNode.TextTrim())
		if href != "" {
			fmt.Fprintf(&sb, `<b><a href="%s">%s</a></b><br>`, html.EscapeString(href), title)
		} else {
			fmt.Fprintf(&sb, "<b>%s</b><br>", title)
		}
	}
	if desc != "" {
		fmt.Fprintf(&sb, "<p>%s</p>", html.EscapeString(desc))
	}
	if imageSrc != "" {
		fmt.Fprintf(&sb, `<img src="%s" referrerpolicy="no-referrer">`, html.EscapeString(imageSrc))
	}
	sb.WriteString("</blockquote>")
	return sb.String()
}

func telegramChannelTitle(doc *parser.Document, channel string) string {
	title := strings.Join(strings.Fields(doc.Text("div.tgme_channel_info_header_title span")), " ")
	if title == "" {
		title = strings.TrimSpace(doc.Text("div.tgme_channel_info_header_title"))
	}
	if title == "" {
		title = channel
	}
	return title
}
