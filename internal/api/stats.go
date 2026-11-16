package api

// Stats is what a write endpoint changed. The API returns only the counters
// relevant to the endpoint (terms/add: parsed + added; translations/delete:
// parsed + deleted…); the others stay 0. Returned by projects/upload,
// projects/sync, terms/add|update|delete|add_comment, languages/update and
// translations/add|update|delete.
type Stats struct {
	Terms        *TermStats        `json:"terms,omitempty"`
	Translations *TranslationStats `json:"translations,omitempty"`
}

// TermStats counts terms (server: ApiV2_Terms import stats).
type TermStats struct {
	Parsed           int `json:"parsed"`
	Added            int `json:"added"`
	Updated          int `json:"updated"`
	Deleted          int `json:"deleted"`
	WithAddedComment int `json:"with_added_comment"` // terms/add_comment
}

// TranslationStats counts translations. projects/upload of a multi-language
// file reports zeros here and the real counts per language in Details.
type TranslationStats struct {
	Parsed  int             `json:"parsed"`
	Added   int             `json:"added"`
	Updated int             `json:"updated"`
	Deleted int             `json:"deleted"`
	Details []LanguageStats `json:"details,omitempty"`
}

// LanguageStats are the translation counts of one language.
type LanguageStats struct {
	Language LanguageRef `json:"language"`
	Parsed   int         `json:"parsed"`
	Added    int         `json:"added"`
	Updated  int         `json:"updated"`
}
