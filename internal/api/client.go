// Package api is the POEditor API v2 client. Every endpoint is a form POST to
// <base>/<endpoint> with an api_token field; every response is wrapped in
//
//	{"response": {"status": "success|fail", "code": "200", "message": "OK"}, "result": {...}}
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API v2 root.
const DefaultBaseURL = "https://api.poeditor.com/v2/"

// Client talks to API v2 with one token.
type Client struct {
	BaseURL   string
	Token     string
	UserAgent string
	HTTP      *http.Client
}

// NewClient returns a client for baseURL ("" means production).
func NewClient(baseURL, token, userAgent string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:   baseURL,
		Token:     token,
		UserAgent: userAgent,
		HTTP:      &http.Client{Timeout: 60 * time.Second},
	}
}

// Error is a "fail" response from the API.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("API error %s: %s", e.Code, e.Message)
}

type envelope struct {
	Response struct {
		Status  string `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"response"`
	Result json.RawMessage `json:"result"`
}

// Post calls endpoint (e.g. "projects/list") with params and decodes the
// "result" object into result (which may be nil).
func (c *Client) Post(ctx context.Context, endpoint string, params url.Values, result any) error {
	form := url.Values{}
	for key, values := range params {
		form[key] = values
	}
	form.Set("api_token", c.Token)
	return c.do(ctx, endpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), result)
}

// Upload calls endpoint as multipart/form-data with params plus one file in
// the form field "file" (projects/upload).
func (c *Client) Upload(ctx context.Context, endpoint string, params url.Values, fileName string, data []byte, result any) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for key, values := range params {
		for _, v := range values {
			if err := mw.WriteField(key, v); err != nil {
				return err
			}
		}
	}
	if err := mw.WriteField("api_token", c.Token); err != nil {
		return err
	}
	part, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	return c.do(ctx, endpoint, mw.FormDataContentType(), &body, result)
}

func (c *Client) do(ctx context.Context, endpoint, contentType string, reqBody io.Reader, result any) error {
	endpointURL := strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach the POEditor API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot read the API response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("unexpected API response (HTTP %d): %s", resp.StatusCode, snippet(body))
	}
	if env.Response.Status != "success" {
		return &Error{Code: env.Response.Code, Message: env.Response.Message}
	}
	if result == nil || len(env.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Result, result); err != nil {
		return fmt.Errorf("unexpected API result for %s: %w", endpoint, err)
	}
	return nil
}

func snippet(body []byte) string {
	const max = 200
	s := strings.TrimSpace(string(body))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

// Download fetches a file the API handed out (e.g. projects/export's url).
// Those links carry their own one-time hash, so no token is sent.
func (c *Client) Download(ctx context.Context, fileURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot download the exported file: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read the exported file: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading the exported file failed (HTTP %d): %s", resp.StatusCode, snippet(body))
	}
	return body, nil
}
