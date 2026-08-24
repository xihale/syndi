package disguise

import (
	"context"
	"encoding/json"
	"encoding/xml"

	"github.com/xihale/syndi/internal/client"
	"github.com/xihale/syndi/internal/parser"
	"github.com/xihale/syndi/internal/parser/rssfeed"
	"github.com/xihale/syndi/pkg/models"
)

// Request binds a profile to one target fetch. Obtain via (*Profile).Fetch or
// use the Profile's Get* helpers directly.
type Request struct {
	p    *Profile
	url  string
	body interface{} // non-nil => POST JSON
}

// Fetch starts a disguised GET request to url.
func (p *Profile) Fetch(url string) *Request { return &Request{p: p, url: url} }

// PostJSON starts a disguised POST with a JSON-serializable body.
func (p *Profile) PostJSON(url string, body interface{}) *Request {
	return &Request{p: p, url: url, body: body}
}

// GetBytes performs the request and returns the raw response body.
func (r *Request) GetBytes(ctx context.Context, cl *client.Client) ([]byte, error) {
	r.p.nap()
	headers := r.p.Headers(r.url)

	if r.body != nil {
		payload, err := json.Marshal(r.body)
		if err != nil {
			return nil, err
		}
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
		return cl.PostWithHeaders(ctx, r.url, payload, headers)
	}
	return cl.GetWithHeaders(ctx, r.url, headers)
}

// GetString performs the request and returns the body as a string.
func (r *Request) GetString(ctx context.Context, cl *client.Client) (string, error) {
	data, err := r.GetBytes(ctx, cl)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetJSON performs the request and unmarshals the JSON response into target.
func (r *Request) GetJSON(ctx context.Context, cl *client.Client, target interface{}) error {
	data, err := r.GetBytes(ctx, cl)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return &jsonError{url: r.url, err: err}
	}
	return nil
}

// GetHTML performs the request and parses the response as HTML.
func (r *Request) GetHTML(ctx context.Context, cl *client.Client) (*parser.Document, error) {
	data, err := r.GetBytes(ctx, cl)
	if err != nil {
		return nil, err
	}
	return parser.LoadString(string(data))
}

// GetXML performs the request and unmarshals the XML response into target.
func (r *Request) GetXML(ctx context.Context, cl *client.Client, target interface{}) error {
	data, err := r.GetBytes(ctx, cl)
	if err != nil {
		return err
	}
	if err := xml.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}

// GetFeed performs the request and parses a native RSS/Atom/RDF feed.
func (r *Request) GetFeed(ctx context.Context, cl *client.Client) (*models.Feed, error) {
	data, err := r.GetBytes(ctx, cl)
	if err != nil {
		return nil, err
	}
	return rssfeed.Parse(data)
}

type jsonError struct {
	url string
	err error
}

func (e *jsonError) Error() string {
	return "disguise: invalid JSON from " + e.url + ": " + e.err.Error()
}

func (e *jsonError) Unwrap() error { return e.err }
