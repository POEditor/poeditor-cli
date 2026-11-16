package api

import (
	"context"
	"net/url"
	"sort"
)

// Language is one entry of languages/list
// (server: ApiV2_Languages_List::generateLanguageArray).
type Language struct {
	Name         string `json:"name"`
	Code         string `json:"code"`
	Translations int    `json:"translations"`
}

// LanguagesList calls languages/list: the languages of a project.
func (c *Client) LanguagesList(ctx context.Context, projectID int) ([]Language, error) {
	var result struct {
		Languages []Language `json:"languages"`
	}
	params := url.Values{"id": {idParam(projectID)}}
	if err := c.post(ctx, "languages/list", params, &result); err != nil {
		return nil, err
	}
	return result.Languages, nil
}

// LanguageCodes calls languages/list and returns only the codes, sorted.
func (c *Client) LanguageCodes(ctx context.Context, projectID int) ([]string, error) {
	langs, err := c.LanguagesList(ctx, projectID)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(langs))
	for _, l := range langs {
		codes = append(codes, l.Code)
	}
	sort.Strings(codes)
	return codes, nil
}
