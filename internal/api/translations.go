package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// TranslationRow is one translation to write: the term it belongs to and its
// text (languages/update; later translations/add and update). The server
// skips a row without a term (ApiV2_Languages_Update::isUserSentDataValid).
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
