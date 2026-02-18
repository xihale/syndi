package routeutils

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/rsshub/go/internal/client"
	"github.com/rsshub/go/internal/parser"
)

// GetJSON fetches URL and unmarshals JSON response into target
func GetJSON(ctx context.Context, client *client.Client, url string, target interface{}) error {
	return client.GetJSON(ctx, url, target)
}

// GetJSONWithHeaders fetches with custom headers and unmarshals JSON
func GetJSONWithHeaders(ctx context.Context, client *client.Client, url string, headers map[string]string, target interface{}) error {
	return client.GetJSONWithHeaders(ctx, url, headers, target)
}

// GetXML fetches and unmarshals XML
func GetXML(ctx context.Context, client *client.Client, url string, target interface{}) error {
	return client.GetXML(ctx, url, target)
}

// GetXMLWithHeaders fetches with custom headers and unmarshals XML
func GetXMLWithHeaders(ctx context.Context, client *client.Client, url string, headers map[string]string, target interface{}) error {
	return client.GetXMLWithHeaders(ctx, url, headers, target)
}

// GetHTML fetches and returns parsed Document
func GetHTML(ctx context.Context, client *client.Client, url string) (*parser.Document, error) {
	return client.GetHTML(ctx, url)
}

// GetHTMLWithHeaders fetches HTML with headers
func GetHTMLWithHeaders(ctx context.Context, client *client.Client, url string, headers map[string]string) (*parser.Document, error) {
	return client.GetHTMLWithHeaders(ctx, url, headers)
}

// UnmarshalJSON unmarshals JSON with error wrapping
func UnmarshalJSON(data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// UnmarshalXML unmarshals XML with error wrapping
func UnmarshalXML(data []byte, target interface{}) error {
	if err := xml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal XML: %w", err)
	}
	return nil
}

// FetchBytes fetches raw bytes from a URL
func FetchBytes(ctx context.Context, client *client.Client, url string) ([]byte, error) {
	return client.Get(ctx, url)
}

// FetchBytesWithHeaders fetches raw bytes with custom headers
func FetchBytesWithHeaders(ctx context.Context, client *client.Client, url string, headers map[string]string) ([]byte, error) {
	return client.GetWithHeaders(ctx, url, headers)
}

// FetchString fetches and returns response as string
func FetchString(ctx context.Context, client *client.Client, url string) (string, error) {
	data, err := client.Get(ctx, url)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FetchStringWithHeaders fetches string with custom headers
func FetchStringWithHeaders(ctx context.Context, client *client.Client, url string, headers map[string]string) (string, error) {
	data, err := client.GetWithHeaders(ctx, url, headers)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
