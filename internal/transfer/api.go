package transfer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// The API calls of pull/push and of `poeditor api projects export|upload`:
// one implementation, so the two spellings cannot drift.

const (
	// UploadDelay is the wait between uploads: the paid-plan limit. Free
	// accounts (20s) get the rest through the 4048 retries.
	UploadDelay = 10 * time.Second
	// MaxRetries is how many times one upload is retried after 4048.
	MaxRetries = 3
)

// Export calls projects/export and downloads the file it links to.
func Export(ctx context.Context, client *api.Client, projectID int, opts api.ExportOptions) ([]byte, error) {
	link, err := client.ProjectsExport(ctx, projectID, opts)
	if err != nil {
		return nil, err
	}
	return client.Download(ctx, link)
}

// Upload calls projects/upload, waiting and retrying while the API says it
// is too early (4048); each wait is announced on stderr.
func Upload(ctx context.Context, f *cmdutil.Factory, client *api.Client, projectID int, name string, data []byte, opts api.UploadOptions) (api.Stats, error) {
	for attempt := 0; ; attempt++ {
		counts, err := client.ProjectsUpload(ctx, projectID, name, data, opts)
		var apiErr *api.Error
		if err == nil || !errors.As(err, &apiErr) || apiErr.Code != api.CodeTooManyUploads {
			return counts, err
		}
		if attempt == MaxRetries {
			return counts, fmt.Errorf("%w (still refused after %d retries; try again in a minute)", err, MaxRetries)
		}
		if err := Wait(ctx, f, UploadDelay, "and retrying: POEditor allows one upload every 10–20s"); err != nil {
			return counts, err
		}
	}
}

// Wait sleeps d (through f.Sleep: tests don't wait), saying why on stderr.
func Wait(ctx context.Context, f *cmdutil.Factory, d time.Duration, why string) error {
	fmt.Fprintf(f.IO.ErrOut, "waiting %ds %s…\n", int(d.Seconds()), why)
	return f.Sleep(ctx, d)
}

// UploadCounts are an upload's counts as --json reports them (push, and
// `poeditor api projects upload`): our own shape, not the API's.
type UploadCounts struct {
	Terms        *TermCounts        `json:"terms,omitempty"`
	Translations *TranslationCounts `json:"translations,omitempty"`
}

// TermCounts are the term counts of one upload.
type TermCounts struct {
	Parsed  int `json:"parsed"`
	Added   int `json:"added"`
	Deleted int `json:"deleted"`
}

// TranslationCounts are the translation counts of one upload.
type TranslationCounts struct {
	Parsed  int `json:"parsed"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
}

// CountsOf copies what --json reports from the API's stats.
func CountsOf(s api.Stats) UploadCounts {
	var c UploadCounts
	if t := s.Terms; t != nil {
		c.Terms = &TermCounts{Parsed: t.Parsed, Added: t.Added, Deleted: t.Deleted}
	}
	if t := s.Translations; t != nil {
		c.Translations = &TranslationCounts{Parsed: t.Parsed, Added: t.Added, Updated: t.Updated}
	}
	return c
}

// String is the counts on one line, for stderr:
// "terms: 3 parsed, 2 added, 0 deleted · translations: 2 added, 1 updated".
func (c UploadCounts) String() string {
	var parts []string
	if t := c.Terms; t != nil {
		parts = append(parts, fmt.Sprintf("terms: %d parsed, %d added, %d deleted", t.Parsed, t.Added, t.Deleted))
	}
	if t := c.Translations; t != nil {
		parts = append(parts, fmt.Sprintf("translations: %d added, %d updated", t.Added, t.Updated))
	}
	return strings.Join(parts, " · ")
}
