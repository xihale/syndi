package routes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var telegramChannelPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,64}$`)

var telegramChannelRoute = routeutils.RouteSpec{
	Path:        "channel/:channel",
	Name:        "Telegram Channel",
	Example:     "telegram/channel/durov",
	Maintainers: []string{"xihale"},
	Description: "Public preview posts of a Telegram channel",
	Categories:  []models.Category{{Name: "social-media"}},
	Features:    models.Features{},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("channel", "Channel username without @, e.g. durov"),
	},
	CacheTTL: 15 * time.Minute,
	Handler:  TelegramChannelHandler,
}

// TelegramChannelHandler handles /telegram/channel/:channel
func TelegramChannelHandler(c *ctxpkg.Context) (*models.Feed, error) {
	channel := c.Param("channel")
	if !telegramChannelPattern.MatchString(channel) {
		return nil, fmt.Errorf("invalid channel name %q", channel)
	}
	ctx := c.Parent()

	url := fmt.Sprintf("https://t.me/s/%s", channel)
	doc, err := routeutils.GetHTML(ctx, c.Client(), url)
	if err != nil {
		return nil, err
	}

	channelTitle := telegramChannelTitle(doc, channel)
	feed := routeutils.NewFeed(
		fmt.Sprintf("Telegram %s", channelTitle),
		fmt.Sprintf("https://t.me/%s", channel),
		fmt.Sprintf("Public posts of Telegram channel %s", channelTitle),
	)

	doc.Each("div.tgme_widget_message_wrap", func(i int, wrap *parser.Selection) {
		item := parseTelegramMessage(wrap, channel, channelTitle)
		routeutils.AddItem(feed, item)
	})

	return feed, nil
}

func parseTelegramMessage(wrap *parser.Selection, channel, channelTitle string) *models.Item {
	msg := wrap.Find("div.tgme_widget_message")
	if msg.Length() == 0 {
		return nil
	}

	// Messages without a text node (pure media posts) are skipped.
	textSel := msg.Find("div.tgme_widget_message_text")
	if textSel.Length() == 0 {
		return nil
	}

	link := ""
	if dateLink := msg.Find("a.tgme_widget_message_date"); dateLink.Length() > 0 {
		link = dateLink.AttrOr("href", "")
	}
	if link == "" {
		if dataPost := msg.AttrOr("data-post", ""); dataPost != "" {
			link = fmt.Sprintf("https://t.me/%s", dataPost)
		} else {
			link = fmt.Sprintf("https://t.me/%s", channel)
		}
	}

	description := ""
	if htmlStr, err := textSel.Html(); err == nil {
		description = strings.TrimSpace(htmlStr)
	}
	if description == "" {
		description = textSel.TextTrim()
	}
	if description == "" {
		return nil
	}

	title := truncateText(extractPlainText(description), 80)
	if title == "" {
		return nil
	}

	var pubDate time.Time
	if datetime := msg.Find("time").AttrOr("datetime", ""); datetime != "" {
		if parsed, err := time.Parse(time.RFC3339, datetime); err == nil {
			pubDate = parsed
		}
	}

	item := routeutils.NewItem(title, link, description, pubDate)
	if dataPost := msg.AttrOr("data-post", ""); dataPost != "" {
		item.GUID = "telegram-" + dataPost
	} else {
		item.GUID = link
	}

	author := msg.Find("span.tgme_widget_message_owner_name").TextTrim()
	if author == "" {
		author = channelTitle
	}
	if author != "" {
		routeutils.SetAuthor(item, author, routeutils.WithAuthorURI("https://t.me/"+channel))
	}
	return item
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
