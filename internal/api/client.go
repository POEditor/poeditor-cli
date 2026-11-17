// Package api is the POEditor API v2 client. Every endpoint is a form POST to
// <base>/<endpoint> with an api_token field; every response is wrapped in
//
//	{"response": {"status": "success|fail", "code": "200", "message": "OK"}, "result": {...}}
//
// Layout: client.go is the transport (post, upload, Download, *Error); each
// API group has its own file (projects.go, languages.go…) with one exported
// method per endpoint, named after it (projects/list → ProjectsList), plus its
// result types and an options struct when the endpoint has optional
// parameters. The transport is unexported, so commands can only call the API
// through those methods.
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

// File is a file sent as a multipart form field.
type File struct {
	Field string // form field name; projects/upload expects "file"
	Name  string // file name sent to the server
	Data  []byte
}

// post calls endpoint (e.g. "projects/list") with params and decodes the
// "result" object into result (which may be nil).
func (c *Client) post(ctx context.Context, endpoint string, params url.Values, result any) error {
	env, err := c.send(ctx, endpoint, params, nil)
	if err != nil {
		return err
	}
	return decodeResult(endpoint, env, result)
}

// upload is post as multipart/form-data, with one file.
func (c *Client) upload(ctx context.Context, endpoint string, params url.Values, file File, result any) error {
	env, err := c.send(ctx, endpoint, params, &file)
	if err != nil {
		return err
	}
	return decodeResult(endpoint, env, result)
}

func decodeResult(endpoint string, env envelope, result any) error {
	if result == nil || len(env.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Result, result); err != nil {
		return fmt.Errorf("unexpected API result for %s: %w", endpoint, err)
	}
	return nil
}

// send POSTs params plus api_token (a url-encoded form, or multipart when
// file is set) and returns the decoded envelope. A "fail" response is an
// *Error.
func (c *Client) send(ctx context.Context, endpoint string, params url.Values, file *File) (envelope, error) {
	contentType, reqBody, err := c.encode(params, file)
	if err != nil {
		return envelope{}, err
	}
	endpointURL := strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, reqBody)
	if err != nil {
		return envelope{}, err
	}
	req.Header.Set("Content-Type", contentType)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return envelope{}, fmt.Errorf("cannot reach the POEditor API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return envelope{}, fmt.Errorf("cannot read the API response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return envelope{}, fmt.Errorf("unexpected API response (HTTP %d): %s", resp.StatusCode, snippet(body))
	}
	if env.Response.Status != "success" {
		return env, &Error{Code: env.Response.Code, Message: env.Response.Message}
	}
	return env, nil
}

// encode builds the request body: params plus api_token, url-encoded, or
// multipart/form-data when there is a file.
func (c *Client) encode(params url.Values, file *File) (contentType string, body io.Reader, err error) {
	form := url.Values{}
	for key, values := range params {
		form[key] = values
	}
	form.Set("api_token", c.Token)
	if file == nil {
		return "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), nil
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for key, values := range form {
		for _, v := range values {
			if err := mw.WriteField(key, v); err != nil {
				return "", nil, err
			}
		}
	}
	part, err := mw.CreateFormFile(file.Field, file.Name)
	if err != nil {
		return "", nil, err
	}
	if _, err := part.Write(file.Data); err != nil {
		return "", nil, err
	}
	if err := mw.Close(); err != nil {
		return "", nil, err
	}
	return mw.FormDataContentType(), &buf, nil
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
