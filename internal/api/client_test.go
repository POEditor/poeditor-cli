package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func server(t *testing.T, handler http.HandlerFunc) *api.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return api.NewClient(srv.URL+"/v2/", "tok", "poeditor-cli/test")
}

func TestPost_sendsFormWithToken(t *testing.T) {
	var gotPath, gotUA, gotType string
	var gotForm url.Values
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotUA, gotType = r.URL.Path, r.UserAgent(), r.Header.Get("Content-Type")
		_ = r.ParseForm()
		gotForm = r.PostForm
		io.WriteString(w, `{"response":{"status":"success","code":"200","message":"OK"}}`)
	})

	err := client.PostForTest(context.Background(), "terms/list", url.Values{"id": {"7717"}}, nil)

	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v2/terms/list" {
		t.Errorf("path = %q", gotPath)
	}
	if gotType != "application/x-www-form-urlencoded" {
		t.Errorf("content type = %q", gotType)
	}
	if gotUA != "poeditor-cli/test" {
		t.Errorf("user agent = %q", gotUA)
	}
	if gotForm.Get("api_token") != "tok" || gotForm.Get("id") != "7717" {
		t.Errorf("form = %v", gotForm)
	}
}

func TestPost_decodesResult(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"response":{"status":"success","code":"200","message":"OK"},"result":{"n":3}}`)
	})
	var result struct{ N int }

	if err := client.PostForTest(context.Background(), "x", nil, &result); err != nil {
		t.Fatal(err)
	}
	if result.N != 3 {
		t.Errorf("N = %d, want 3", result.N)
	}
}

func TestPost_failResponseIsAPIError(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"response":{"status":"fail","code":"4011","message":"Invalid API Token"}}`)
	})

	err := client.PostForTest(context.Background(), "x", nil, nil)

	var apiErr *api.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "4011" || apiErr.Message != "Invalid API Token" {
		t.Fatalf("got %v", err)
	}
}

func TestPost_nonJSONResponse(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "<html>502 Bad Gateway</html>")
	})

	err := client.PostForTest(context.Background(), "x", nil, nil)

	if err == nil || !strings.Contains(err.Error(), "HTTP 502") || !strings.Contains(err.Error(), "Bad Gateway") {
		t.Fatalf("got %v", err)
	}
}

func TestNewClient_defaultsToProduction(t *testing.T) {
	if c := api.NewClient("", "t", ""); c.BaseURL != api.DefaultBaseURL {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

func TestDownload(t *testing.T) {
	var gotToken, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotToken = r.Method, r.URL.Query().Get("api_token")
		if r.URL.Path == "/gone" {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "expired")
			return
		}
		io.WriteString(w, `{"hello":"world"}`)
	}))
	t.Cleanup(srv.Close)
	client := api.NewClient(srv.URL+"/v2/", "tok", "ua")

	body, err := client.Download(context.Background(), srv.URL+"/file")

	if err != nil || string(body) != `{"hello":"world"}` {
		t.Fatalf("got %q, %v", body, err)
	}
	if gotMethod != http.MethodGet || gotToken != "" {
		t.Errorf("method %s, token %q: want a plain GET without the token", gotMethod, gotToken)
	}
	if _, err := client.Download(context.Background(), srv.URL+"/gone"); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("404: %v", err)
	}
}

func TestUpload_multipart(t *testing.T) {
	var gotForm url.Values
	var gotFile, gotName string
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("not multipart: %v", err)
		}
		gotForm = url.Values(r.MultipartForm.Value)
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Errorf("no file: %v", err)
		} else {
			data, _ := io.ReadAll(f)
			gotFile, gotName = string(data), hdr.Filename
		}
		io.WriteString(w, `{"response":{"status":"success","code":"200","message":"OK"},"result":{"terms":{"added":2}}}`)
	})
	var result struct {
		Terms struct{ Added int } `json:"terms"`
	}

	err := client.UploadForTest(context.Background(), "projects/upload", url.Values{"id": {"7"}, "updating": {"terms"}}, "en.json", []byte(`{"a":"b"}`), &result)

	if err != nil {
		t.Fatal(err)
	}
	if gotForm.Get("api_token") != "tok" || gotForm.Get("id") != "7" || gotForm.Get("updating") != "terms" {
		t.Errorf("fields = %v", gotForm)
	}
	if gotFile != `{"a":"b"}` || gotName != "en.json" {
		t.Errorf("file %q named %q", gotFile, gotName)
	}
	if result.Terms.Added != 2 {
		t.Errorf("result = %+v", result)
	}
}

func TestRaw_returnsBodyAndResult(t *testing.T) {
	body := `{"response":{"status":"success","code":"200","message":"OK"},"result":{"n":3}}`
	client := server(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })

	resp, err := client.Raw(context.Background(), "x/y", url.Values{"id": {"1"}}, nil)

	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Body) != body || string(resp.Result) != `{"n":3}` {
		t.Errorf("resp = %s / %s", resp.Body, resp.Result)
	}
}

func TestRaw_failureKeepsBody(t *testing.T) {
	body := `{"response":{"status":"fail","code":"4012","message":"Invalid project"}}`
	client := server(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })

	resp, err := client.Raw(context.Background(), "projects/view", nil, nil)

	var apiErr *api.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("err = %v", err)
	}
	if string(resp.Body) != body || resp.Result != nil {
		t.Errorf("resp = %s / %s", resp.Body, resp.Result)
	}
}

func TestRaw_multipartFieldName(t *testing.T) {
	var gotField, gotToken string
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		for name := range r.MultipartForm.File {
			gotField = name
		}
		gotToken = r.MultipartForm.Value["api_token"][0]
		io.WriteString(w, `{"response":{"status":"success","code":"200","message":"OK"}}`)
	})

	_, err := client.Raw(context.Background(), "x/y", nil, &api.File{Field: "attachment", Name: "a.txt", Data: []byte("hi")})

	if err != nil {
		t.Fatal(err)
	}
	if gotField != "attachment" || gotToken != "tok" {
		t.Errorf("field = %q, token = %q", gotField, gotToken)
	}
}
