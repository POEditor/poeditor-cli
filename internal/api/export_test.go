package api

import (
	"context"
	"net/url"
)

// The raw transport is unexported so commands can only reach the API through
// the typed endpoint methods; client_test.go tests it through these.

func (c *Client) PostForTest(ctx context.Context, endpoint string, params url.Values, result any) error {
	return c.post(ctx, endpoint, params, result)
}

func (c *Client) UploadForTest(ctx context.Context, endpoint string, params url.Values, fileName string, data []byte, result any) error {
	return c.upload(ctx, endpoint, params, File{Field: "file", Name: fileName, Data: data}, result)
}
