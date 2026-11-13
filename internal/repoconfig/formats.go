package repoconfig

// Format is an API v2 file type.
type Format struct {
	Name   string // human name
	Import bool   // accepted by projects/upload (push)
}

// Formats are the API v2 `type` values, taken from PoeditorSupportedFormats
// (api = true) in the main app. An unknown format is only a warning: the API
// may have gained one since this list was written.
var Formats = map[string]Format{
	"android_strings": {"Android String Resources", true},
	"apple_strings":   {"iOS Strings", true},
	"arb":             {"Flutter ARB", true},
	"csv":             {"CSV", true},
	"i18next":         {"i18next JSON", true},
	"ini":             {"INI", true},
	"json":            {"JSON", true},
	"key_value_json":  {"Key-Value JSON", true},
	"mo":              {"Gettext MO", false},
	"po":              {"Gettext PO", true},
	"pot":             {"Gettext POT", true},
	"properties":      {"Java Properties", true},
	"resw":            {".NET RESW", true},
	"resx":            {".NET RESX", true},
	"rise_360_xliff":  {"Rise 360 XLIFF", true},
	"stringsdict":     {"iOS Stringsdict", true},
	"ts":              {"Qt TS", true},
	"xcstrings":       {"iOS Xcstrings", true},
	"xlf":             {"Angular XLIFF", true},
	"xliff":           {"iOS XLIFF", true},
	"xliff_1_2":       {"XLIFF 1.2", true},
	"xls":             {"Microsoft Excel", true},
	"xlsx":            {"Microsoft Excel", true},
	"xmb":             {"Angular XMB", true},
	"xtb":             {"Angular XTB", true},
	"yml":             {"YAML", true},
}

// PullFilters are the projects/export `filters` values. The API silently
// drops unknown ones, so they are validated here.
var PullFilters = []string{
	"translated", "untranslated",
	"fuzzy", "not_fuzzy",
	"automatic", "not_automatic",
	"ai", "not_ai",
	"tm", "not_tm",
	"proofread", "not_proofread",
}

// TagsMatchModes are the projects/export `tags_match` values.
var TagsMatchModes = []string{"all", "any", "exclude"}

// UpdatingModes are the projects/upload `updating` values.
var UpdatingModes = []string{"terms", "terms_translations", "translations"}
