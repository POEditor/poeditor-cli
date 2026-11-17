package api

import (
	"context"
	"net/url"
	"sort"
)

// Error codes of the language parameter (server: ApiV2::getProjectLanguage,
// ApiV2_Languages::getLanguageByCode, Project::addNewLanguage).
const (
	CodeUnknownLanguage      = "4043" // not a language code POEditor knows
	CodeLanguageNotInProject = "4044"
	CodeLanguageInProject    = "4050" // languages/add: already there
)

// LanguageRef names a language wherever the API nests one: languages/available,
// a term's source-language block (terms/list "drl"), per-language upload stats.
type LanguageRef struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Language is a project language: one entry of languages/list
// (server: ApiV2_Languages_List::prepareLanguageResponse).
type Language struct {
	LanguageRef
	Translations int        `json:"translations"` // translated terms (intval)
	Percentage   Percentage `json:"percentage"`   // translated share of the terms
	Updated      Time       `json:"updated"`      // last translation added; zero when none
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

// LanguagesAvailable calls languages/available: every language POEditor
// offers to the account, its own custom languages included
// (server: ApiV2_Languages_Available, IsoLanguages::findAllByUser).
func (c *Client) LanguagesAvailable(ctx context.Context) ([]LanguageRef, error) {
	var result struct {
		Languages []LanguageRef `json:"languages"`
	}
	if err := c.post(ctx, "languages/available", url.Values{}, &result); err != nil {
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

// LanguagesAdd calls languages/add: adds one language to a project. A
// language already in the project is an *Error with CodeLanguageInProject.
func (c *Client) LanguagesAdd(ctx context.Context, projectID int, code string) error {
	return c.post(ctx, "languages/add", languageParams(projectID, code), nil)
}

// LanguagesDelete calls languages/delete: removes a language from a
// project, with all its translations.
func (c *Client) LanguagesDelete(ctx context.Context, projectID int, code string) error {
	return c.post(ctx, "languages/delete", languageParams(projectID, code), nil)
}

// LanguageUpdateOptions are languages/update's optional parameters.
type LanguageUpdateOptions struct {
	// FuzzyTrigger marks the translations of the other languages fuzzy
	// where this update added or changed one (fuzzy_trigger).
	FuzzyTrigger bool
}

// LanguagesUpdate calls languages/update: writes the translations of one
// language, overwriting existing ones (server: ApiV2_Languages_Update,
// import type import_def). The result counts are in Stats.Translations.
// Rows must not be empty: the server takes an empty list for invalid data.
func (c *Client) LanguagesUpdate(ctx context.Context, projectID int, code string, rows []TranslationRow, opts LanguageUpdateOptions) (Stats, error) {
	extra := url.Values{"language": {code}}
	if opts.FuzzyTrigger {
		extra.Set("fuzzy_trigger", "1")
	}
	return c.postRows(ctx, "languages/update", projectID, rows, len(rows), extra)
}

func languageParams(projectID int, code string) url.Values {
	return url.Values{"id": {idParam(projectID)}, "language": {code}}
}
