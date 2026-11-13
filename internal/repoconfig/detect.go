package repoconfig

import (
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Candidate is a path pattern found in the repo, e.g. "locales/{lang}.json".
type Candidate struct {
	Pattern     string
	LanguageMap map[string]string // POEditor code → code used in the files, when they differ
	Found       []string          // POEditor codes with an existing file, sorted
	Format      string            // guessed from the pattern; "" if unknown
}

// Directories never scanned: dependencies, build output, VCS and IDE data.
var skipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true, "target": true,
	"Pods": true, "Carthage": true, "DerivedData": true, "bin": true, "obj": true,
	"__pycache__": true, "venv": true,
}

const (
	maxDepth   = 8
	maxEntries = 20000
	maxResults = 8
)

// Detect scans root for files whose path contains one of the project's
// language codes (in the spellings platforms use: pt-br, pt-BR, pt_BR,
// Android's pt-rBR) and groups them into {lang} patterns, the patterns that
// match the most languages first.
func Detect(root string, codes []string) []Candidate {
	type hit struct{ code, local string }
	patterns := map[string]map[string]string{} // pattern → code → local spelling

	variants := map[string]string{} // spelling → code
	for _, code := range codes {
		for _, v := range spellings(code) {
			if _, taken := variants[v]; !taken {
				variants[v] = code
			}
		}
	}

	seen := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		seen++
		if seen > maxEntries {
			return fs.SkipAll
		}
		rel, err := filepath.Rel(root, p)
		if err != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || skipDirs[name] || strings.Count(rel, "/") >= maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		for _, h := range matches(rel, variants) {
			pattern, code, local := h[0], h[1], h[2]
			if patterns[pattern] == nil {
				patterns[pattern] = map[string]string{}
			}
			if _, done := patterns[pattern][code]; !done {
				patterns[pattern][code] = local
			}
		}
		return nil
	})

	var out []Candidate
	for pattern, byCode := range patterns {
		c := Candidate{Pattern: pattern, Format: GuessFormat(pattern)}
		for code, local := range byCode {
			c.Found = append(c.Found, code)
			if local != code {
				if c.LanguageMap == nil {
					c.LanguageMap = map[string]string{}
				}
				c.LanguageMap[code] = local
			}
		}
		sort.Strings(c.Found)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Found) != len(out[j].Found) {
			return len(out[i].Found) > len(out[j].Found)
		}
		return out[i].Pattern < out[j].Pattern
	})
	if len(out) > maxResults {
		out = out[:maxResults]
	}
	return out
}

// matches returns [pattern, code, local] for every place in rel where a
// language spelling stands as its own token (bounded by / - _ . or the ends).
func matches(rel string, variants map[string]string) [][3]string {
	var out [][3]string
	for local, code := range variants {
		for from := 0; ; {
			i := strings.Index(rel[from:], local)
			if i < 0 {
				break
			}
			start, end := from+i, from+i+len(local)
			if boundary(rel, start-1) && boundary(rel, end) {
				out = append(out, [3]string{rel[:start] + "{lang}" + rel[end:], code, local})
			}
			from = start + 1
		}
	}
	return out
}

func boundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return strings.ContainsRune("/-_.", rune(s[i]))
}

// spellings are the ways a POEditor code appears in file paths.
func spellings(code string) []string {
	out := []string{code}
	lang, region, ok := strings.Cut(code, "-")
	if !ok {
		return out
	}
	switch len(region) {
	case 2: // pt-br → pt-BR, pt_BR, pt_br, pt-rBR (Android)
		up := strings.ToUpper(region)
		out = append(out, lang+"-"+up, lang+"_"+up, lang+"_"+region, lang+"-r"+up)
	case 4: // zh-hans → zh-Hans, zh_Hans
		title := strings.ToUpper(region[:1]) + strings.ToLower(region[1:])
		out = append(out, lang+"-"+title, lang+"_"+title)
	}
	return out
}

// GuessFormat maps a path to the most likely API format, or "". A .ts file
// is not guessed: it is far more often TypeScript than Qt TS.
func GuessFormat(p string) string {
	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case ".xml":
		if strings.Contains(p, "values-{lang}") || strings.Contains(p, "/values/") {
			return "android_strings"
		}
		return ""
	case ".json":
		return "key_value_json"
	case ".yaml", ".yml":
		return "yml"
	case ".strings":
		return "apple_strings"
	}
	byExt := map[string]string{
		".po": "po", ".pot": "pot", ".mo": "mo", ".arb": "arb", ".properties": "properties",
		".resx": "resx", ".resw": "resw", ".csv": "csv", ".ini": "ini",
		".xlf": "xlf", ".xliff": "xliff", ".xmb": "xmb", ".xtb": "xtb", ".xls": "xls", ".xlsx": "xlsx",
		".stringsdict": "stringsdict", ".xcstrings": "xcstrings",
	}
	return byExt[ext]
}
