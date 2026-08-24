package routes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xihale/syndi/internal/routeutils"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

var ieeeTopicSlug = regexp.MustCompile(`^[a-z0-9-]+$`)

var ieeeSpectrumTopicRoute = routeutils.RouteSpec{
	Path:        "topic/:topic",
	Name:        "IEEE Spectrum Topic",
	Example:     "ieee-spectrum/topic/artificial-intelligence",
	Maintainers: []string{"xihale"},
	Description: "IEEE Spectrum news for a topic (e.g. artificial-intelligence, robotics, computing, biomedical, energy)",
	Categories:  []models.Category{{Name: "technology"}},
	Features:    models.Features{SupportRadar: true},
	Parameters: []models.Parameter{
		routeutils.RequiredParam("topic", "topic slug, e.g. artificial-intelligence or robotics"),
	},
	CacheTTL: 1 * time.Hour,
	Handler:  IEEESpectrumTopicHandler,
}

// IEEESpectrumTopicHandler handles /ieee-spectrum/topic/:topic
func IEEESpectrumTopicHandler(c *ctxpkg.Context) (*models.Feed, error) {
	topic := strings.TrimSpace(c.Param("topic"))
	if !ieeeTopicSlug.MatchString(topic) {
		return nil, fmt.Errorf("invalid topic slug %q (lowercase letters, digits and dashes only)", topic)
	}

	feed, err := routeutils.GetFeed(c.Parent(), c.Client(), fmt.Sprintf("https://spectrum.ieee.org/feeds/topic/%s.rss", topic))
	if err != nil {
		return nil, err
	}
	feed.Title = fmt.Sprintf("IEEE Spectrum: %s", topic)
	feed.Link = fmt.Sprintf("https://spectrum.ieee.org/topic/%s/", topic)
	feed.Description = fmt.Sprintf("Latest IEEE Spectrum coverage of %s", topic)
	return feed, nil
}
