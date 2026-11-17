package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

// TranslationRow is one translation to write: the term it belongs to and its
// text (languages/update, translations/add and update). The server skips a
// row without a term (ApiV2_Languages_Update::isUserSentDataValid).
// translations/add and update ignore Fuzzy ($enable_fuzzy_flag = false).
type TranslationRow struct {
	TermKey
	Translation TranslationText `json:"translation"`
}

// TranslationText is the translation of a TranslationRow
// (server: getTranslationFromRow, getFuzzyFromRow).
type TranslationText struct {
	Content Content `json:"content"`
	Fuzzy   int     `json:"fuzzy"` // 0/1
}

// Translation is a term's text in one language: terms/list "translation"
// (server: ApiV2_Terms_List::populateTranslationArray).
type Translation struct {
	Content Content `json:"content"`
	Fuzzy   int     `json:"fuzzy"` // 0/1
	// Proofread is 0/1, or nil when the project has proofreading off (the
	// API leaves the key out).
	Proofread *int `json:"proofread,omitempty"`
	Updated   Time `json:"updated"` // zero when never translated
}

// IsEmpty reports a term not translated in this language: no text, never
// updated.
func (t Translation) IsEmpty() bool { return t.Content.IsEmpty() && t.Updated.IsZero() }

// Content is a translation's text. The API sends a string for a normal term
// and an object of plural forms ({"one": "…", "other": "…"}, in the
// language's order) for a plural one, so it reads and writes either.
type Content struct {
	Text  string       // a normal term
	Forms []PluralForm // a plural term, in the API's order; nil otherwise
}

// PluralForm is the text of one plural form (one, few, many, other…).
type PluralForm struct {
	Form string
	Text string
}

// IsPlural reports the plural shape (an object of forms).
func (c Content) IsPlural() bool { return c.Forms != nil }

// IsEmpty reports no text at all (every plural form empty too).
func (c Content) IsEmpty() bool {
	for _, f := range c.Forms {
		if f.Text != "" {
			return false
		}
	}
	return c.Text == ""
}

// UnmarshalJSON reads a string, an object of plural forms (key order kept),
// [] (PHP's empty array: plural, no forms) or null.
func (c *Content) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	*c = Content{}
	switch {
	case len(data) == 0:
		return errors.New("content: empty value")
	case string(data) == "null":
		return nil
	case data[0] == '"':
		return json.Unmarshal(data, &c.Text)
	case string(data) == "[]":
		c.Forms = []PluralForm{}
		return nil
	case data[0] == '{':
		forms, err := pluralForms(data)
		if err != nil {
			return fmt.Errorf("content: %w", err)
		}
		c.Forms = forms
		return nil
	}
	return fmt.Errorf("content: want a string or plural forms, got %s", data)
}

// pluralForms reads {"form": "text", …} keeping the key order, which a Go map
// would lose.
func pluralForms(data []byte) ([]PluralForm, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if _, err := dec.Token(); err != nil { // {
		return nil, err
	}
	forms := []PluralForm{}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var text string
		if err := dec.Decode(&text); err != nil {
			return nil, fmt.Errorf("plural form %q: %w", key, err)
		}
		forms = append(forms, PluralForm{Form: key.(string), Text: text})
	}
	return forms, nil
}

// MarshalJSON writes the API's shape back: a string, or an object of plural
// forms in order. HTML is not escaped (<b> stays <b>): translations hold it.
func (c Content) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	if !c.IsPlural() {
		writeJSONString(&b, c.Text)
		return b.Bytes(), nil
	}
	b.WriteByte('{')
	for i, f := range c.Forms {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONString(&b, f.Form)
		b.WriteByte(':')
		writeJSONString(&b, f.Text)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// writeJSONString writes s as a JSON string, without escaping HTML.
func writeJSONString(b *bytes.Buffer, s string) {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)       // a string always encodes
	b.Truncate(b.Len() - 1) // Encode ends with a newline
}

// TranslationsAdd calls translations/add: fills the given terms' empty
// translations in one language; existing ones are left as they are
// (overwrite 0). The counts are Stats.Translations: parsed, added.
func (c *Client) TranslationsAdd(ctx context.Context, projectID int, language string, rows []TranslationRow) (Stats, error) {
	return c.postRows(ctx, "translations/add", projectID, rows, len(rows), url.Values{"language": {language}})
}

// TranslationsUpdate calls translations/update: overwrites the given terms'
// existing translations in one language; a term without one is skipped.
// The counts are Stats.Translations: parsed, updated.
func (c *Client) TranslationsUpdate(ctx context.Context, projectID int, language string, rows []TranslationRow) (Stats, error) {
	return c.postRows(ctx, "translations/update", projectID, rows, len(rows), url.Values{"language": {language}})
}

// TranslationsDelete calls translations/delete: deletes the given terms'
// translations in one language. The counts are Stats.Translations: parsed,
// deleted.
func (c *Client) TranslationsDelete(ctx context.Context, projectID int, language string, terms []TermKey) (Stats, error) {
	return c.postRows(ctx, "translations/delete", projectID, terms, len(terms), url.Values{"language": {language}})
}

// Automatic translation providers (server: ApiV2_Translations_Automatic::PROVIDERS).
const (
	ProviderGoogle    = "google"
	ProviderMicrosoft = "microsoft"
	ProviderDeepL     = "deepl"
)

// Providers lists the automatic translation providers.
var Providers = []string{ProviderGoogle, ProviderMicrosoft, ProviderDeepL}

// AutomaticOptions are the parameters of translations/automatic.
type AutomaticOptions struct {
	Provider string // one of Providers
	// SourceLanguage is the project language to translate from; the
	// provider's code for it is ProviderSourceLanguage.
	SourceLanguage         string
	ProviderSourceLanguage string
	Targets                []AutomaticTarget
	Tags                   []string          // only the terms with these tags (options.tags)
	TagsMatch              string            // TagsMatchAll | Any | Exclude; sent only with Tags
	Options                map[string]string // provider options (format, formality…)
}

// AutomaticTarget is a language to translate to: its project code and the
// provider's code for it.
type AutomaticTarget struct {
	ProjectLanguage  string `json:"project_language"`
	ProviderLanguage string `json:"provider_language"`
}

// AutomaticResult is translations/automatic's result: one entry per
// language pair (server: ApiMachineTranslationsBatchProcessor::prepareMessageForPair).
type AutomaticResult struct {
	Successful []AutomaticPair `json:"successful"`
	Failed     []AutomaticPair `json:"failed"`
}

// AutomaticPair is a language pair, in the provider's codes. Target is ""
// when the server found no pair; Chars is set on success, Message on failure.
type AutomaticPair struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Chars   int    `json:"chars"`
	Message string `json:"message"`
}

// TranslationsAutomatic calls translations/automatic. A pair that failed is
// in AutomaticResult.Failed, not an error; the API refuses the call (an
// *Error, code 4040) when every pair failed or the input is wrong.
func (c *Client) TranslationsAutomatic(ctx context.Context, projectID int, o AutomaticOptions) (AutomaticResult, error) {
	targets := o.Targets
	if targets == nil {
		targets = []AutomaticTarget{}
	}
	params := url.Values{
		"id":                       {idParam(projectID)},
		"provider":                 {o.Provider},
		"source_language":          {o.SourceLanguage},
		"provider_source_language": {o.ProviderSourceLanguage},
		"target_languages":         {jsonParam(targets)},
	}
	options := map[string]any{}
	for k, v := range o.Options {
		options[k] = v
	}
	if len(o.Tags) > 0 {
		options["tags"] = o.Tags
		if o.TagsMatch != "" {
			options["tags_match"] = o.TagsMatch
		}
	}
	if len(options) > 0 {
		params.Set("options", jsonParam(options))
	}
	var result AutomaticResult
	err := c.post(ctx, "translations/automatic", params, &result)
	return result, err
}
