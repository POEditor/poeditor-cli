// Package testutil is the test kit for commands: a fake API v2 server, a
// runner that executes the real command tree against it, golden files, and
// the option-coverage check every command package's TestMain runs.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// TestToken is the API token the runner configures against the fake API.
const TestToken = "test-token"

// Request is one call the fake API received.
type Request struct {
	Endpoint string     // e.g. "projects/list"
	Form     url.Values // includes api_token
}

// Stub is a canned API response.
type Stub struct {
	status  string
	code    string
	message string
	result  any
}

// OK is a success response whose "result" is result (nil = no result key).
func OK(result any) Stub {
	return Stub{status: "success", code: "200", message: "OK", result: result}
}

// Fail is a failure response, e.g. Fail("4011", "Invalid API Token").
func Fail(code, message string) Stub {
	return Stub{status: "fail", code: code, message: message}
}

// FakeAPI is an httptest server speaking the API v2 envelope.
type FakeAPI struct {
	URL string

	t        testing.TB
	mu       sync.Mutex
	stubs    map[string]func(url.Values) Stub
	requests []Request
}

// NewFakeAPI starts a fake API that is closed when the test ends.
func NewFakeAPI(t testing.TB) *FakeAPI {
	t.Helper()
	fake := &FakeAPI{t: t, stubs: map[string]func(url.Values) Stub{}}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)
	fake.URL = server.URL + "/v2/"
	return fake
}

// Stub sets the response for endpoint ("projects/list").
func (f *FakeAPI) Stub(endpoint string, stub Stub) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stubs[endpoint] = func(url.Values) Stub { return stub }
}

// StubFunc sets a response that depends on the request, e.g. on the project id:
//
//	fake.StubFunc("projects/view", func(form url.Values) testutil.Stub { ... form.Get("id") ... })
func (f *FakeAPI) StubFunc(endpoint string, fn func(form url.Values) Stub) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stubs[endpoint] = fn
}

// Requests returns every request received so far, in order.
func (f *FakeAPI) Requests() []Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Request(nil), f.requests...)
}

// OnlyRequest asserts exactly one request was made and returns it.
func (f *FakeAPI) OnlyRequest(t testing.TB) Request {
	t.Helper()
	reqs := f.Requests()
	if len(reqs) != 1 {
		t.Fatalf("expected exactly 1 API request, got %d: %+v", len(reqs), reqs)
	}
	return reqs[0]
}

func (f *FakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	endpoint := strings.TrimPrefix(r.URL.Path, "/v2/")
	if r.Method != http.MethodPost {
		f.t.Errorf("fake API: %s %s, API v2 only accepts POST", r.Method, endpoint)
	}
	if err := r.ParseForm(); err != nil {
		f.t.Errorf("fake API: bad form for %s: %v", endpoint, err)
	}

	f.mu.Lock()
	f.requests = append(f.requests, Request{Endpoint: endpoint, Form: r.PostForm})
	stubFn, ok := f.stubs[endpoint]
	f.mu.Unlock()

	var stub Stub
	if ok {
		stub = stubFn(r.PostForm)
	} else {
		f.t.Errorf("fake API: no stub for endpoint %q", endpoint)
		stub = Fail("404", "endpoint not stubbed in test")
	}

	body := map[string]any{
		"response": map[string]string{"status": stub.status, "code": stub.code, "message": stub.message},
	}
	if stub.result != nil {
		body["result"] = stub.result
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		f.t.Errorf("fake API: encoding response: %v", err)
	}
}
