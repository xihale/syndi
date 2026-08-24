// Package testutil provides helpers for live-testing route handlers.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/xihale/syndi/internal/client"
	ctxpkg "github.com/xihale/syndi/pkg/context"
	"github.com/xihale/syndi/pkg/models"
)

// RunHandler invokes a route handler with a real HTTP context and client.
// pathParams maps :param names to values; the URL path is synthetic.
func RunHandler(handler models.HandlerFunc, pathParams map[string]string) (*models.Feed, error) {
	req := httptest.NewRequest(http.MethodGet, "/live-test", nil)
	rec := httptest.NewRecorder()
	c := ctxpkg.NewContext(rec, req)
	if len(pathParams) > 0 {
		c.SetParams(pathParams)
	}
	cl := client.New(
		client.WithUserAgent("Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0"),
		client.WithTimeout(30*time.Second),
	)
	c.SetClient(cl)
	return handler(c)
}
