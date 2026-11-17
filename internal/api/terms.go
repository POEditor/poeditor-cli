package api

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// TermKey is a term's identity: its text and its context ("" when it has
// none). Endpoints that change a term find it by both
// (server: ApiV2_TermsHelper::findTerm).
type TermKey struct {
	Term    string `json:"term"`
	Context string `json:"context"`
}

// Check reports a key the server would skip: a term with no text
// (server: Nip_Helper_String::hasContent).
func (k TermKey) Check() error {
	if strings.TrimSpace(k.Term) == "" {
		return errors.New(`no "term"`)
	}
	return nil
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

// NewTerm is a term to add (terms/add; server:
// ApiV2_Terms::prepareTermArrayForImport). A term already in the project is
// left as it is: counted as parsed, not added.
type NewTerm struct {
	TermKey
	Plural    string   `json:"plural,omitempty"`
	Reference string   `json:"reference,omitempty"`
	Comment   string   `json:"comment,omitempty"` // added only to a new term
	Tags      []string `json:"tags,omitempty"`
}

// TermUpdate changes a term found by its TermKey (terms/update; server:
// ApiV2_Terms_Update). A nil field is left as it is; "" clears plural or
// reference.
type TermUpdate struct {
	TermKey
	NewTerm    *string   `json:"new_term,omitempty"`
	NewContext *string   `json:"new_context,omitempty"`
	Reference  *string   `json:"reference,omitempty"`
	Plural     *string   `json:"plural,omitempty"`
	Tags       *[]string `json:"tags,omitempty"` // replaces the term's tags; [] removes them all
}

// Check reports an update the server would skip or apply badly: no term,
// or a new_term with no text (the server would blank the term).
func (u TermUpdate) Check() error {
	if err := u.TermKey.Check(); err != nil {
		return err
	}
	if u.NewTerm != nil && strings.TrimSpace(*u.NewTerm) == "" {
		return errors.New(`"new_term" is empty`)
	}
	return nil
}

// TermComment is a comment to add to a term (terms/add_comment; server:
// ApiV2_Terms_AddComment). The API stores it for the term, in no language.
type TermComment struct {
	TermKey
	Comment string `json:"comment"`
	Pinned  int    `json:"pinned,omitempty"` // 1 pins it
}

// Check reports a comment the server would skip: no term, or no text.
func (c TermComment) Check() error {
	if err := c.TermKey.Check(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Comment) == "" {
		return errors.New(`no "comment"`)
	}
	return nil
}

// TermsAdd calls terms/add. The counts are Stats.Terms: parsed, added.
func (c *Client) TermsAdd(ctx context.Context, projectID int, terms []NewTerm) (Stats, error) {
	return c.postRows(ctx, "terms/add", projectID, terms, len(terms), nil)
}

// TermsUpdateOptions are terms/update's optional parameters.
type TermsUpdateOptions struct {
	// FuzzyTrigger marks every translation of the updated terms fuzzy
	// (fuzzy_trigger).
	FuzzyTrigger bool
}

// TermsUpdate calls terms/update. The counts are Stats.Terms: parsed,
// updated (every term found, changed or not).
func (c *Client) TermsUpdate(ctx context.Context, projectID int, terms []TermUpdate, opts TermsUpdateOptions) (Stats, error) {
	var extra url.Values
	if opts.FuzzyTrigger {
		extra = url.Values{"fuzzy_trigger": {"1"}}
	}
	return c.postRows(ctx, "terms/update", projectID, terms, len(terms), extra)
}

// TermsDelete calls terms/delete. The counts are Stats.Terms: parsed,
// deleted.
func (c *Client) TermsDelete(ctx context.Context, projectID int, terms []TermKey) (Stats, error) {
	return c.postRows(ctx, "terms/delete", projectID, terms, len(terms), nil)
}

// TermsAddComment calls terms/add_comment. The counts are Stats.Terms:
// parsed, with_added_comment.
func (c *Client) TermsAddComment(ctx context.Context, projectID int, comments []TermComment) (Stats, error) {
	return c.postRows(ctx, "terms/add_comment", projectID, comments, len(comments), nil)
}

// postRows sends a bulk endpoint's rows as "data" and returns its counts.
// No rows is an error before any request: the server takes an empty list
// for invalid data.
func (c *Client) postRows(ctx context.Context, endpoint string, projectID int, rows any, n int, extra url.Values) (Stats, error) {
	if n == 0 {
		return Stats{}, errors.New(endpoint + ": nothing to send")
	}
	params := url.Values{"id": {idParam(projectID)}, "data": {jsonParam(rows)}}
	for k, v := range extra {
		params[k] = v
	}
	var result Stats
	err := c.post(ctx, endpoint, params, &result)
	return result, err
}
