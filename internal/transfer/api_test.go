package transfer_test

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/testutil"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

// factory records the waits instead of sleeping.
func factory(waits *[]time.Duration) (*cmdutil.Factory, *bytes.Buffer) {
	ios, _, _, stderr := iostreams.Test()
	f := &cmdutil.Factory{IO: ios, Sleep: func(_ context.Context, d time.Duration) error {
		*waits = append(*waits, d)
		return nil
	}}
	return f, stderr
}

func TestExport(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/export", testutil.OK(map[string]string{"url": fake.File("fr.po", []byte(`msgid "a"`))}))
	client := api.NewClient(fake.URL, testutil.TestToken, "test")

	data, err := transfer.Export(context.Background(), client, 7717, api.ExportOptions{Language: "fr", Type: "po"})

	if err != nil || string(data) != `msgid "a"` {
		t.Fatalf("data = %q, err = %v", data, err)
	}
	if req := fake.Requests()[0]; req.Endpoint != "projects/export" || req.Form.Get("type") != "po" {
		t.Errorf("request = %s %v", req.Endpoint, req.Form)
	}
}

func TestUpload_retriesWhileTooEarly(t *testing.T) {
	tests := []struct {
		name      string
		refusals  int // 4048 answers before success
		wantWaits int
		wantErr   string
	}{
		{"at once", 0, 0, ""},
		{"after two refusals", 2, 2, ""},
		{"gives up", transfer.MaxRetries + 1, transfer.MaxRetries, "still refused after 3 retries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			calls := 0
			fake.StubFunc("projects/upload", func(url.Values) testutil.Stub {
				calls++
				if calls <= tt.refusals {
					return testutil.Fail(api.CodeTooManyUploads, "Too many upload requests")
				}
				return testutil.OK(map[string]any{"terms": map[string]int{"parsed": 2, "added": 1}})
			})
			client := api.NewClient(fake.URL, testutil.TestToken, "test")
			var waits []time.Duration
			f, stderr := factory(&waits)

			stats, err := transfer.Upload(context.Background(), f, client, 7717, "en.json", []byte("{}"), api.UploadOptions{Updating: "terms"})

			if tt.wantErr != "" {
				var apiErr *api.Error
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || !errors.As(err, &apiErr) {
					t.Errorf("error = %v", err)
				}
			} else if err != nil || stats.Terms == nil || stats.Terms.Added != 1 {
				t.Errorf("stats = %+v, err = %v", stats.Terms, err)
			}
			if len(waits) != tt.wantWaits {
				t.Errorf("waits = %v, want %d", waits, tt.wantWaits)
			}
			for _, w := range waits {
				if w != transfer.UploadDelay {
					t.Errorf("waited %v", w)
				}
			}
			if n := strings.Count(stderr.String(), "waiting 10s and retrying"); n != tt.wantWaits {
				t.Errorf("stderr = %q", stderr.String())
			}
		})
	}
}

// Another API error is not retried.
func TestUpload_otherErrors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/upload", testutil.Fail("4046", "could not parse"))
	client := api.NewClient(fake.URL, testutil.TestToken, "test")
	var waits []time.Duration
	f, _ := factory(&waits)

	_, err := transfer.Upload(context.Background(), f, client, 7717, "en.json", []byte("{}"), api.UploadOptions{})

	var apiErr *api.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "4046" || len(waits) != 0 || len(fake.Requests()) != 1 {
		t.Errorf("err = %v, waits = %v, %d requests", err, waits, len(fake.Requests()))
	}
}

func TestCountsOf(t *testing.T) {
	tests := []struct {
		stats api.Stats
		want  string
	}{
		{api.Stats{}, ""},
		{api.Stats{Terms: &api.TermStats{Parsed: 3, Added: 2, Deleted: 1, Updated: 9}}, "terms: 3 parsed, 2 added, 1 deleted"},
		{api.Stats{Terms: &api.TermStats{Parsed: 3, Added: 2}, Translations: &api.TranslationStats{Parsed: 3, Added: 2, Updated: 1}},
			"terms: 3 parsed, 2 added, 0 deleted · translations: 2 added, 1 updated"},
	}
	for _, tt := range tests {
		c := transfer.CountsOf(tt.stats)
		if got := c.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
		if (c.Terms == nil) != (tt.stats.Terms == nil) || (c.Translations == nil) != (tt.stats.Translations == nil) {
			t.Errorf("counts = %+v", c)
		}
	}
}
