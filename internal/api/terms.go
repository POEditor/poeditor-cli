package api

import (
	"context"
	"net/url"
)

// TermKey is a term's identity: its text and its context ("" when it has
// none). Endpoints that change a term find it by both
// (server: ApiV2_TermsHelper::findTerm).
type TermKey struct {
	Term    string `json:"term"`
	Context string `json:"context"`
}

// Term is a term as terms/list returns it
// (server: ApiV2_Terms_List::populateTermArray).
type Term struct {
	TermKey
	Plural    string   `json:"plural"` // the plural's name; "" for a normal term
	Reference string   `json:"reference"`
	Comment   string   `json:"comment"` // the latest comment ("" if none)
	Tags      []string `json:"tags"`
	Created   Time     `json:"created"`
	Updated   Time     `json:"updated"` // zero when never changed
	// Translation is set only when the list was asked with a language.
	Translation *Translation `json:"translation,omitempty"`
}

// IsPlural reports a term with plural forms.
func (t Term) IsPlural() bool { return t.Plural != "" }

// TermsList calls terms/list: every term of a project and, when language is
// not "", each term's translation in that language.
func (c *Client) TermsList(ctx context.Context, projectID int, language string) ([]Term, error) {
	var result struct {
		Terms []Term `json:"terms"`
	}
	params := url.Values{"id": {idParam(projectID)}}
	if language != "" {
		params.Set("language", language)
	}
	if err := c.post(ctx, "terms/list", params, &result); err != nil {
		return nil, err
	}
	return result.Terms, nil
}
