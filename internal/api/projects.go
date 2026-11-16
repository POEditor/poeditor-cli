package api

import (
	"context"
	"errors"
	"net/url"
)

// Project is the short project shape: one entry of projects/list
// (server: ApiV2_Projects_List::formatResponse).
type Project struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Public  int    `json:"public"`
	Open    int    `json:"open"`
	Created string `json:"created"`
}

// ProjectsList calls projects/list: every project the token can access.
func (c *Client) ProjectsList(ctx context.Context) ([]Project, error) {
	var result struct {
		Projects []Project `json:"projects"`
	}
	if err := c.post(ctx, "projects/list", nil, &result); err != nil {
		return nil, err
	}
	return result.Projects, nil
}

// ProjectDetails is the full project shape: the short one plus settings and
// term counts. Returned by projects/view, projects/add and projects/update
// (server: ApiV2_Projects::generateProjectArray).
type ProjectDetails struct {
	Project
	Description       string `json:"description"`
	ReferenceLanguage string `json:"reference_language"` // code, or ""
	FallbackLanguage  string `json:"fallback_language"`  // code, or ""
	Terms             int    `json:"terms"`
}

// ProjectsView calls projects/view: one project's details.
func (c *Client) ProjectsView(ctx context.Context, projectID int) (ProjectDetails, error) {
	var result struct {
		Project ProjectDetails `json:"project"`
	}
	params := url.Values{"id": {idParam(projectID)}}
	err := c.post(ctx, "projects/view", params, &result)
	return result.Project, err
}

// ExportOptions are the projects/export parameters besides the project id.
// Empty fields are not sent.
type ExportOptions struct {
	Language         string
	Type             string   // file format, API name (key_value_json, apple_strings…)
	Filters          []string // translated, untranslated, fuzzy…
	Tags             []string
	TagsMatch        string // all | any | exclude; sent only with Tags
	Order            string
	FallbackLanguage string
}

func (o ExportOptions) values(projectID int) url.Values {
	params := url.Values{
		"id":       {idParam(projectID)},
		"language": {o.Language},
		"type":     {o.Type},
	}
	if len(o.Filters) > 0 {
		params.Set("filters", jsonParam(o.Filters))
	}
	if len(o.Tags) > 0 {
		params.Set("tags", jsonParam(o.Tags))
		if o.TagsMatch != "" {
			params.Set("tags_match", o.TagsMatch)
		}
	}
	if o.Order != "" {
		params.Set("order", o.Order)
	}
	if o.FallbackLanguage != "" {
		params.Set("fallback_language", o.FallbackLanguage)
	}
	return params
}

// ProjectsExport calls projects/export and returns the download link
// (fetch it with Download).
func (c *Client) ProjectsExport(ctx context.Context, projectID int, opts ExportOptions) (string, error) {
	var result struct {
		URL string `json:"url"`
	}
	if err := c.post(ctx, "projects/export", opts.values(projectID), &result); err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", errors.New("the API returned no download link")
	}
	return result.URL, nil
}

// UploadOptions are the projects/upload parameters besides the project id
// and the file. False and empty fields are not sent.
type UploadOptions struct {
	Updating       string // terms | terms_translations | translations
	Language       string
	Overwrite      bool
	SyncTerms      bool
	ReadFromSource bool
	FuzzyTrigger   bool
	Tags           UploadTags
}

// UploadTags are the tag groups of projects/upload; empty groups are not sent.
type UploadTags struct {
	All                     []string `json:"all,omitempty"`
	New                     []string `json:"new,omitempty"`
	Obsolete                []string `json:"obsolete,omitempty"`
	OverwrittenTranslations []string `json:"overwritten_translations,omitempty"`
}

// IsZero reports whether no group has tags.
func (t UploadTags) IsZero() bool {
	return len(t.All)+len(t.New)+len(t.Obsolete)+len(t.OverwrittenTranslations) == 0
}

// String is the "tags" parameter as sent: a JSON object of the non-empty groups.
func (t UploadTags) String() string {
	if t.IsZero() {
		return ""
	}
	return jsonParam(t)
}

func (o UploadOptions) values(projectID int) url.Values {
	params := url.Values{
		"id":       {idParam(projectID)},
		"updating": {o.Updating},
		"language": {o.Language},
	}
	for name, on := range map[string]bool{
		"overwrite":        o.Overwrite,
		"sync_terms":       o.SyncTerms,
		"read_from_source": o.ReadFromSource,
		"fuzzy_trigger":    o.FuzzyTrigger,
	} {
		if on {
			params.Set(name, "1")
		}
	}
	if !o.Tags.IsZero() {
		params.Set("tags", o.Tags.String())
	}
	return params
}

// CodeTooManyUploads is the projects/upload rate-limit code
// ("Too many upload requests": one upload every 10s paid, 20s free).
const CodeTooManyUploads = "4048"

// ProjectsUpload calls projects/upload with one file (multipart field "file").
func (c *Client) ProjectsUpload(ctx context.Context, projectID int, fileName string, data []byte, opts UploadOptions) (Stats, error) {
	var result Stats
	err := c.upload(ctx, "projects/upload", opts.values(projectID), fileName, data, &result)
	return result, err
}
